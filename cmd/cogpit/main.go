// Command cogpit is the Cogpit desktop app: a native Kubernetes IDE.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/pprof"
	"sync"
	"time"

	// Cron Jobs' Next execution reads spec.timeZone; Windows has no zone database of its own.
	_ "time/tzdata"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/klog/v2"

	"github.com/guided-traffic/cogpit/internal/app"
	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/demo"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/helm"
	"github.com/guided-traffic/cogpit/internal/mcp"
	"github.com/guided-traffic/cogpit/internal/ui/fyneui"
)

// version is set by the build (-ldflags "-X main.version=…").
var version = "0.0.1-mvp"

func main() {
	// Not deferred: on a panic of the main goroutine the defers run before the runtime writes the
	// crash, and done would remove the report (crash.go).
	previousCrash, done := captureCrashes(crashReportDir())
	code := run(previousCrash)
	done()
	os.Exit(code)
}

func run(previousCrash string) int {
	var (
		kubeconfig  = flag.String("kubeconfig", "", "kubeconfig files, separated by '"+string(os.PathListSeparator)+"', offered as sources of their own for this run; default: the files $KUBECONFIG lists. ~/.kube is always a sync folder")
		contextName = flag.String("context", "", "connect to this context at start instead of showing the context picker")
		demoMode    = flag.Bool("demo", false, "connect to the synthetic demo cluster (10,000 pods, 1,000 watch events per second)")
		demoPods    = flag.Int("demo-pods", 10000, "pods in the demo cluster")
		demoEvents  = flag.Float64("demo-events", 1000, "watch events per second in the demo cluster")
		mcpAddr     = flag.String("mcp-addr", "127.0.0.1:0", "MCP listen address; loopback only")
		noMCP       = flag.Bool("no-mcp", false, "do not start the MCP server")
		perf        = flag.Duration("perf", 0, "measure main-loop latency while auto-scrolling the table for this long, print the result as JSON and quit")
		perfMode    = flag.String("perf-mode", "", "with --perf (or alone for gate): scroll, or gate for start-up, scrolling, every view and typing into 5 MB of YAML in one run")
		cpuProfile  = flag.String("cpuprofile", "", "write a CPU profile of the first --cpuprofile-for to this file")
		cpuFor      = flag.Duration("cpuprofile-for", 5*time.Second, "how long --cpuprofile records")
		themeName   = flag.String("theme", "", "colour theme for this run: dark, light or system; default: the Preferences' theme (dark until set)")
		verbose     = flag.Bool("v", false, "debug logging")
	)
	flag.Parse()

	if *cpuProfile != "" {
		stop, err := startCPUProfile(*cpuProfile, *cpuFor)
		if err != nil {
			fmt.Fprintln(os.Stderr, "cogpit:", err)
			return 1
		}
		defer stop()
	}

	logger := slog.New(helm.RedactSDKLog(newLogger(*verbose).Handler())) // Helm values never reach the log
	slog.SetDefault(logger)
	klog.SetSlogLogger(logger)
	// client-go's informer goroutines re-panic after logging by default, which would end the app
	// because one watch failed (D-11: crash-tolerant UI). Log instead; the cache reports the stopped
	// informer as an error state.
	utilruntime.ReallyCrash = false
	utilruntime.PanicHandlers = append(utilruntime.PanicHandlers, func(_ context.Context, r any) {
		logger.Error("recovered panic in a client-go goroutine", "panic", fmt.Sprint(r))
	})

	// A GUI app started from Finder lacks the shell's PATH; exec credential plugins and kubectl
	// live there (gap H-6).
	cluster.EnsureToolPath()

	audit, closeAudit := openAuditLog(logger)
	defer closeAudit()

	clusterSettings := openClusterSettings(logger) // the session's, and the connector's proxies
	connector := &engine.ClusterConnector{
		KubeconfigPaths: filepath.SplitList(*kubeconfig),
		IncludeDemo:     true,
		DemoOptions:     demo.Options{Pods: *demoPods, EventsPerSecond: *demoEvents},
		Logger:          logger,
		Sources:         openKubeconfigSources(logger, cluster.NamedKubeconfigFiles(filepath.SplitList(*kubeconfig))),
		MetricsSettings: openMetricsSettings(logger),
		ClusterSettings: clusterSettings,
	}
	s := engine.New(engine.Options{Connector: connector, Audit: audit, Logger: logger, Drains: openDrainJournal(logger),
		Helm: openHelmJournal(logger), Hotbar: openHotbar(logger), HelmRepos: openHelmRepos(logger),
		Preferences: openPreferences(logger), ClusterSettings: clusterSettings})
	defer s.Close()
	defer s.WatchKubeconfigs(0)() // sync folders: files that appear or leave are picked up
	if err := commands.RegisterAll(s.Registry()); err != nil {
		fmt.Fprintln(os.Stderr, "cogpit:", err)
		return 1
	}

	opts := fyneui.Options{AutoConnect: *contextName, Theme: fyneui.ThemeMode(*themeName)}
	if previousCrash != "" {
		at := time.Now()
		if fi, err := os.Stat(previousCrash); err == nil {
			at = fi.ModTime()
		}
		logger.Warn("the last run of Cogpit crashed", "report", previousCrash)
		opts.StartWarning = crashNotice(previousCrash, at)
	}
	if *demoMode {
		opts.AutoConnect = engine.DemoContext
	}
	if !*noMCP {
		srv, err := mcp.New(s, mcp.Options{Addr: *mcpAddr, Version: version, Logger: logger})
		if err != nil {
			logger.Warn("MCP server not started", "err", err)
		} else {
			// The window starts the server, unless the user switched it off in an earlier run, and
			// switches it from the status bar. Quitting closes it for good. --no-mcp wins over
			// the stored choice: no server, no item.
			defer func() { _ = srv.Close() }()
			opts.MCP = &mcpControl{srv: srv, session: s, logger: logger, out: os.Stdout}
		}
	}
	if *perf > 0 || *perfMode != "" {
		opts.PerfDuration = *perf
		opts.PerfMode = *perfMode
		opts.PerfReport = func(ps fyneui.PerfStats) {
			b, _ := json.Marshal(ps)
			fmt.Println(string(b))
		}
	}
	if err := fyneui.Run(s, opts); err != nil {
		fmt.Fprintln(os.Stderr, "cogpit:", err)
		return 1
	}
	return 0
}

// newLogger logs to stderr and to <user cache dir>/Cogpit/cogpit.log, so a Cogpit.app started
// from Finder keeps its log too.
func newLogger(verbose bool) *slog.Logger {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	var w io.Writer = os.Stderr
	if dir, err := os.UserCacheDir(); err == nil {
		dir = filepath.Join(dir, "Cogpit")
		if err := os.MkdirAll(dir, 0o700); err == nil {
			if f, err := os.OpenFile(filepath.Join(dir, "cogpit.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600); err == nil {
				w = io.MultiWriter(os.Stderr, f)
			}
		}
	}
	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: level}))
}

// openAuditLog opens <user config dir>/Cogpit/audit.log, owner-only, for agent-initiated
// commands (D-2; app.Dirs.OpenAuditLog, shared with cogpit-cli).
func openAuditLog(logger *slog.Logger) (io.Writer, func()) {
	dirs, err := app.UserDirs()
	if err != nil {
		logger.Warn("audit log disabled", "err", err)
		return nil, func() {}
	}
	f, err := dirs.OpenAuditLog()
	if err != nil {
		logger.Warn("audit log disabled", "err", err)
		return nil, func() {}
	}
	return f, func() { _ = f.Close() }
}

// openDrainJournal keeps the drains Cogpit starts in <user config dir>/Cogpit/drains.json,
// owner-only, so that a drain a crash interrupts is offered again (D-11). Without a configuration
// directory there is no journal, and node.drain is refused rather than run without one.
func openDrainJournal(logger *slog.Logger) *engine.Drains {
	dir, err := os.UserConfigDir()
	if err != nil {
		logger.Warn("drain journal disabled; node drains are refused", "err", err)
		return nil
	}
	return engine.NewDrains(engine.NewFileDrainJournal(filepath.Join(dir, "Cogpit")))
}

// openHelmJournal keeps the Helm rollbacks and uninstalls Cogpit starts in
// <user config dir>/Cogpit/helm.json, owner-only, so that one a crash interrupts is offered again
// (D-11). Without a configuration directory there is no journal, and they are refused.
func openHelmJournal(logger *slog.Logger) *engine.HelmOps {
	dir, err := os.UserConfigDir()
	if err != nil {
		logger.Warn("Helm journal disabled; release rollbacks and uninstalls are refused", "err", err)
		return nil
	}
	return engine.NewHelmOps(engine.NewFileHelmJournal(filepath.Join(dir, "Cogpit")))
}

// openHotbar keeps the hotbar's banks of pinned clusters in <user config dir>/Cogpit/hotbar.json,
// owner-only (G13). Without a configuration directory the hotbar lives in memory for the run.
func openHotbar(logger *slog.Logger) *engine.Hotbar {
	dir, err := os.UserConfigDir()
	if err != nil {
		logger.Warn("the hotbar is not kept after this run", "err", err)
		return engine.NewHotbar("")
	}
	return engine.NewHotbar(filepath.Join(dir, "Cogpit"))
}

// openPreferences keeps Cogpit's preferences (theme, time zone, editor, terminal, tables) in
// <user config dir>/Cogpit/preferences.json, owner-only (G13). Without a configuration directory
// they live in memory for the run.
func openPreferences(logger *slog.Logger) *engine.PreferencesStore {
	dir, err := os.UserConfigDir()
	if err != nil {
		logger.Warn("preferences are not kept after this run", "err", err)
		return engine.NewPreferencesStore("")
	}
	return engine.NewPreferencesStore(filepath.Join(dir, "Cogpit"))
}

// openClusterSettings keeps each cluster's display name, icon and terminal settings in
// <user config dir>/Cogpit/cluster-settings.json and its uploaded icons in cluster-icons/, owner-only,
// never in the kubeconfig (G13). Without a configuration directory they live in memory for the run.
func openClusterSettings(logger *slog.Logger) *engine.ClusterSettingsStore {
	dir, err := os.UserConfigDir()
	if err != nil {
		logger.Warn("cluster settings are not kept after this run", "err", err)
		return engine.NewClusterSettingsStore("")
	}
	return engine.NewClusterSettingsStore(filepath.Join(dir, "Cogpit"))
}

// openMetricsSettings keeps the Prometheus each cluster's metrics come from, as the user set it,
// in <user config dir>/Cogpit/metrics.json, owner-only (never in the kubeconfig). Without a
// configuration directory a setting lasts for the connection.
func openMetricsSettings(logger *slog.Logger) *engine.MetricsSettings {
	dir, err := os.UserConfigDir()
	if err != nil {
		logger.Warn("metrics settings are not kept", "err", err)
		return nil
	}
	m, err := engine.NewMetricsSettings(filepath.Join(dir, "Cogpit"))
	if err != nil {
		logger.Warn("metrics settings could not be read; starting without", "err", err)
	}
	return m
}

// openHelmRepos keeps the Helm chart repositories the user adds in
// <user config dir>/Cogpit/helm/repositories.json and their indexes in <user cache dir>/Cogpit/helm,
// each owner-only: Cogpit's own folders, never Helm's repositories.yaml or cache (G11). Without a
// cache directory the indexes go under the configuration directory; without a configuration
// directory there are no repositories.
func openHelmRepos(logger *slog.Logger) *engine.HelmRepos {
	dir, err := os.UserConfigDir()
	if err != nil {
		logger.Warn("Helm repositories disabled; charts cannot be installed", "err", err)
		return nil
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		cache = filepath.Join(dir, "Cogpit", "cache")
	} else {
		cache = filepath.Join(cache, "Cogpit")
	}
	return engine.NewHelmRepos(filepath.Join(dir, "Cogpit"), cache)
}

// openKubeconfigSources keeps the kubeconfig files and sync folders the user
// adds in <user config dir>/Cogpit (kubeconfigs.json, exec-approvals.json), each owner-only.
// Without a configuration directory there are none, and the kubeconfig files are merged as kubectl
// merges them.
//
// Every context comes from a source file read on its own; nothing is merged (D-28). The user's own
// ~/.kube (%USERPROFILE%\.kube on Windows), config included, is a sync folder from the start, as
// in Lens, removable like any other. The files named for this run (named: --kubeconfig, else
// $KUBECONFIG) are sources of their own, removable for the session. Cogpit reads these files,
// never writes them, and a credential plugin of any of them needs the user's approval
// (trust-boundaries H-1). Without a home directory there is no default folder. cogpit-cli builds
// the same sources (app.Dirs.KubeconfigSources).
func openKubeconfigSources(logger *slog.Logger, named []string) *engine.KubeconfigSources {
	dirs, err := app.UserDirs()
	if err != nil {
		logger.Warn("kubeconfig sources disabled; the kubeconfig files are merged as kubectl does", "err", err)
		return nil
	}
	return dirs.KubeconfigSources(named)
}

// startCPUProfile records a CPU profile for d (or until stop is called) into path; for the gate
// measurements of the start-up.
func startCPUProfile(path string, d time.Duration) (stop func(), err error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		_ = f.Close()
		return nil, err
	}
	var once sync.Once
	stop = func() {
		once.Do(func() {
			pprof.StopCPUProfile()
			_ = f.Close()
		})
	}
	time.AfterFunc(d, stop)
	return stop, nil
}
