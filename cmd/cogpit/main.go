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

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/klog/v2"

	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/demo"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/mcp"
	"github.com/guided-traffic/cogpit/internal/ui/fyneui"
)

// version is set by the build (-ldflags "-X main.version=…").
var version = "0.0.1-mvp"

func main() { os.Exit(run()) }

func run() int {
	var (
		kubeconfig  = flag.String("kubeconfig", "", "kubeconfig files, separated by '"+string(os.PathListSeparator)+"'; default: $KUBECONFIG, else ~/.kube/config")
		contextName = flag.String("context", "", "connect to this context at start instead of showing the context picker")
		demoMode    = flag.Bool("demo", false, "connect to the synthetic demo cluster (10,000 pods, 1,000 watch events per second)")
		demoPods    = flag.Int("demo-pods", 10000, "pods in the demo cluster")
		demoEvents  = flag.Float64("demo-events", 1000, "watch events per second in the demo cluster")
		mcpAddr     = flag.String("mcp-addr", "127.0.0.1:0", "MCP listen address; loopback only")
		noMCP       = flag.Bool("no-mcp", false, "do not start the MCP server")
		perf        = flag.Duration("perf", 0, "measure main-loop latency while auto-scrolling the table for this long, print the result as JSON and quit")
		themeName   = flag.String("theme", "dark", "colour theme: dark, light or system")
		verbose     = flag.Bool("v", false, "debug logging")
	)
	flag.Parse()

	logger := newLogger(*verbose)
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

	connector := &engine.ClusterConnector{
		KubeconfigPaths: filepath.SplitList(*kubeconfig),
		IncludeDemo:     true,
		DemoOptions:     demo.Options{Pods: *demoPods, EventsPerSecond: *demoEvents},
		Logger:          logger,
	}
	s := engine.New(engine.Options{Connector: connector, Audit: audit, Logger: logger})
	defer s.Close()
	if err := commands.RegisterAll(s.Registry()); err != nil {
		fmt.Fprintln(os.Stderr, "cogpit:", err)
		return 1
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	opts := fyneui.Options{AutoConnect: *contextName, Theme: fyneui.ThemeMode(*themeName)}
	if *demoMode {
		opts.AutoConnect = engine.DemoContext
	}
	if !*noMCP {
		srv, err := mcp.New(s, mcp.Options{Addr: *mcpAddr, Version: version, Logger: logger})
		if err == nil {
			err = srv.Start(ctx)
		}
		if err != nil {
			logger.Warn("MCP server not started", "err", err)
		} else {
			defer func() { _ = srv.Close() }()
			// The command carries the token: print it to the terminal that started Cogpit, never
			// through the logger.
			opts.MCPCommand = srv.ClaudeAddCommand()
			fmt.Printf("Cogpit MCP server: %s\nRegister it with Claude Code (the token changes on every start):\n\n  %s\n\n", srv.URL(), opts.MCPCommand)
		}
	}
	if *perf > 0 {
		opts.PerfDuration = *perf
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
// commands (D-2).
func openAuditLog(logger *slog.Logger) (io.Writer, func()) {
	dir, err := os.UserConfigDir()
	if err != nil {
		logger.Warn("audit log disabled", "err", err)
		return nil, func() {}
	}
	dir = filepath.Join(dir, "Cogpit")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		logger.Warn("audit log disabled", "err", err)
		return nil, func() {}
	}
	f, err := os.OpenFile(filepath.Join(dir, "audit.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		logger.Warn("audit log disabled", "err", err)
		return nil, func() {}
	}
	return f, func() { _ = f.Close() }
}
