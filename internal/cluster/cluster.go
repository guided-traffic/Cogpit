// Package cluster owns connections to clusters: kubeconfig loading, client-go configuration and
// authentication, discovery, the counting transport, and the few direct API calls (actions,
// streams) that the cache cannot serve.
//
// Layer L1. Only cluster, cache, terminal, helm and metrics talk to the API server.
//
// Errors from this package name the context and the problem, never a credential: kubeconfig
// files are validated through kubeconfig.Parse, whose errors are made safe, before client-go
// loads them, and connection errors are rewritten into a ConnectError.
package cluster

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/flowcontrol"

	"github.com/guided-traffic/cogpit/internal/kubeconfig"
)

// ErrNotImplemented marks a Phase 0 stub.
var ErrNotImplemented = errors.New("not implemented")

// Defaults for Options (architecture §5; preferences later).
const (
	// DefaultQPS and DefaultBurst bound lists, gets and writes per cluster.
	DefaultQPS   = 5
	DefaultBurst = 10
	// DefaultConnectTimeout bounds Connect's reachability check. It is generous because an exec
	// credential plugin may need a moment (a cloud CLI refreshing a token) on the first request.
	DefaultConnectTimeout = 15 * time.Second
	// ServedResourcesTTL is how long a discovery result is reused.
	ServedResourcesTTL = 10 * time.Minute
	// partialServedResourcesTTL is how long a discovery result with failed groups (an aggregated
	// API that is down) is reused before discovery is tried again.
	partialServedResourcesTTL = time.Minute
	// discoveryQPS and discoveryBurst are discovery's own budget (kubectl's values).
	discoveryQPS   = 50
	discoveryBurst = 300
)

// Options say which kubeconfig context to connect to.
type Options struct {
	// KubeconfigPaths are the kubeconfig files, merged in order. Empty means client-go's default
	// loading rules: $KUBECONFIG, else ~/.kube/config. Tests always pass explicit paths.
	KubeconfigPaths []string
	// Context is the context name; empty means the files' current-context.
	Context string
	// QPS and Burst bound lists, gets and writes per cluster (not established watches).
	// Zero means the defaults 5 and 10.
	QPS   float32
	Burst int
	// ConnectTimeout bounds Connect's reachability check only; later requests are not bounded
	// by it, because watches stay open. Zero means DefaultConnectTimeout.
	ConnectTimeout time.Duration
	// ExecPolicy, when set, is asked before a context whose user runs an exec credential plugin
	// gets a client. It receives the plugin as client-go will run it (taken from the very
	// configuration the client is built from, so a file changed meanwhile cannot slip another
	// command past it); an error refuses the connection before the plugin runs. Nil runs plugins
	// as kubectl does.
	ExecPolicy func(*kubeconfig.ExecPlugin) error
}

// loadingRules returns client-go's loading rules for paths: exactly these files in this order,
// or the default rules ($KUBECONFIG, else ~/.kube/config) when paths is empty.
func loadingRules(paths []string) *clientcmd.ClientConfigLoadingRules {
	var rules *clientcmd.ClientConfigLoadingRules
	if len(paths) > 0 {
		rules = &clientcmd.ClientConfigLoadingRules{Precedence: append([]string(nil), paths...)}
	} else {
		rules = clientcmd.NewDefaultClientConfigLoadingRules()
	}
	// The default rules copy a kubeconfig from a legacy location (~/.kube/.kubeconfig) to
	// ~/.kube/config when the latter is missing. That is an implicit kubeconfig write, which
	// Cogpit never does.
	rules.MigrationRules = nil
	// Missing files are reported by checkFiles, not by client-go's warner on stderr.
	rules.WarnIfAllMissing = false
	return rules
}

// DefaultKubeconfigFiles returns the files the default loading reads for paths (see Options):
// paths themselves, else the files $KUBECONFIG names, else ~/.kube/config, absolute and clean. It
// reads no file; the engine uses it to offer a file of a synced ~/.kube once, not twice.
func DefaultKubeconfigFiles(paths []string) []string {
	var out []string
	for _, p := range loadingRules(paths).GetLoadingPrecedence() {
		if p == "" {
			continue
		}
		if abs, err := filepath.Abs(p); err == nil {
			out = append(out, filepath.Clean(abs))
		}
	}
	return out
}

// checkFiles reads and validates the kubeconfig files of a loading precedence. It returns the
// absolute paths of the files that exist and parse, and one safe error per file that does not
// parse. A missing file is an error only when the caller named it (explicit); in the default
// precedence a missing ~/.kube/config is normal.
func checkFiles(precedence []string, explicit bool) (usable []string, problems []error) {
	seen := map[string]bool{}
	for _, p := range precedence {
		if p == "" {
			continue
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			abs = p
		}
		if seen[abs] {
			continue
		}
		seen[abs] = true
		data, err := os.ReadFile(abs)
		switch {
		case errors.Is(err, os.ErrNotExist):
			if explicit {
				problems = append(problems, fmt.Errorf("kubeconfig file %s does not exist", abs))
			}
			continue
		case err != nil:
			problems = append(problems, fmt.Errorf("kubeconfig file %s cannot be read: %w", abs, err))
			continue
		}
		if _, err := kubeconfig.Parse(data); err != nil {
			problems = append(problems, fmt.Errorf("kubeconfig file %s: %w", abs, err))
			continue
		}
		usable = append(usable, abs)
	}
	return usable, problems
}

// ListContexts returns the contexts of the merged kubeconfig files (default loading rules when
// paths is empty), sorted by name. A file that is missing (when named explicitly) or does not
// parse is skipped: ListContexts then returns the contexts of the other files together with an
// error that names each skipped file, so one broken file does not hide the rest.
func ListContexts(paths []string) ([]kubeconfig.Context, error) {
	rules := loadingRules(paths)
	usable, problems := checkFiles(rules.GetLoadingPrecedence(), len(paths) > 0)
	rules.Precedence = usable
	cfg, err := rules.Load()
	if err != nil {
		problems = append(problems, fmt.Errorf("kubeconfig: %w", kubeconfig.SafeError(err)))
	}
	out := kubeconfig.FromConfig(cfg)
	if out == nil {
		out = []kubeconfig.Context{}
	}
	return out, errors.Join(problems...)
}

// ConnectError says why Connect failed. Its text names the context and the problem, never a
// credential; Err is the underlying error, kept for errors.Is and errors.As.
type ConnectError struct {
	Context string
	Problem string
	Err     error
}

func (e *ConnectError) Error() string {
	if e.Context == "" {
		return e.Problem
	}
	return fmt.Sprintf("context %q: %s", e.Context, e.Problem)
}

func (e *ConnectError) Unwrap() error { return e.Err }

// Conn is one connection to one kubeconfig context: its client configuration, clients, cached
// discovery and request counter. It is safe for concurrent use.
type Conn struct {
	contextName string
	namespace   string
	paths       []string
	server      string
	version     string
	config      *rest.Config
	httpClient  *http.Client
	kube        kubernetes.Interface
	dyn         dynamic.Interface
	meta        metadata.Interface
	disco       discovery.DiscoveryInterface
	counter     *RequestCounter
	warnings    *warningLog

	servedMu  sync.Mutex
	served    map[schema.GroupVersionResource]bool
	servedAt  time.Time
	servedTTL time.Duration

	objects objectMapper // the REST mapper of the generic object writes (objects.go)
	access  reviewClient // the access reviews' client, with their own budget (access.go)

	closed atomic.Bool
}

// Connect builds a connection for opts.Context. It verifies the server is reachable with one
// discovery request (GET /version, bounded by opts.ConnectTimeout) and returns a *ConnectError a
// user can act on otherwise. Credentials never appear in the error.
func Connect(ctx context.Context, opts Options) (*Conn, error) {
	name := opts.Context
	rules := loadingRules(opts.KubeconfigPaths)
	usable, problems := checkFiles(rules.GetLoadingPrecedence(), len(opts.KubeconfigPaths) > 0)
	if len(problems) > 0 {
		return nil, &ConnectError{Context: name, Problem: errors.Join(problems...).Error()}
	}
	rules.Precedence = usable

	cc := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, &clientcmd.ConfigOverrides{CurrentContext: name})
	raw, err := cc.RawConfig()
	if err != nil {
		return nil, &ConnectError{Context: name, Problem: "cannot load kubeconfig: " + kubeconfig.SafeError(err).Error()}
	}
	if name == "" {
		name = raw.CurrentContext
	}
	if name == "" {
		return nil, &ConnectError{Problem: "no context given and the kubeconfig sets no current-context"}
	}
	kctx, ok := raw.Contexts[name]
	if !ok || kctx == nil {
		return nil, &ConnectError{Context: name, Problem: "no such context in the kubeconfig files " + describePaths(usable)}
	}

	// A token file outside the kubeconfig's folder is read by ClientConfig below: ask first (D-17).
	if err := checkTokenFilePolicy(opts.ExecPolicy, raw, name, kctx); err != nil {
		return nil, &ConnectError{Context: name, Problem: err.Error(), Err: err}
	}
	cfg, err := cc.ClientConfig()
	if err != nil {
		return nil, &ConnectError{Context: name, Problem: "invalid kubeconfig: " + kubeconfig.SafeError(err).Error()}
	}
	if err := checkExecPolicy(opts.ExecPolicy, raw, name, kctx, cfg); err != nil {
		return nil, &ConnectError{Context: name, Problem: err.Error(), Err: err}
	}
	// clientcmd's persister writes tokens a legacy auth provider (oidc) refreshes back into the
	// user's kubeconfig file. Cogpit makes no implicit kubeconfig writes: refreshed tokens stay in
	// memory (gap H-4, docs/security/credentials.md).
	cfg.AuthConfigPersister = memoryPersister{}
	namespace, _, err := cc.Namespace()
	if err != nil || namespace == "" {
		namespace = "default"
	}

	cfg.QPS, cfg.Burst = opts.QPS, opts.Burst
	if cfg.QPS <= 0 {
		cfg.QPS = DefaultQPS
	}
	if cfg.Burst <= 0 {
		cfg.Burst = DefaultBurst
	}
	// One token bucket for the cluster. Left to client-go, the typed clientset and the dynamic
	// client would each build their own from QPS and Burst, doubling the budget; copies of
	// RESTConfig() share this one too.
	cfg.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(cfg.QPS, cfg.Burst)
	// The API server's warnings, logged once per connection, not once per request (warnings.go).
	warnings := newWarningLog(name)
	cfg.WarningHandlerWithContext = warnings
	// No cfg.Timeout: it would end every watch after that long.
	server := kubeconfig.StripUserInfo(cfg.Host)
	counter := NewRequestCounter()
	counter.prefix = serverPathPrefix(cfg.Host)
	cfg.Wrap(counter.Wrap)

	httpClient, err := rest.HTTPClientFor(cfg)
	if err != nil {
		return nil, &ConnectError{Context: name, Problem: "cannot build the client: " + kubeconfig.SafeError(err).Error(), Err: err}
	}
	// One HTTP client for both clientsets: one connection pool (one HTTP/2 connection) per
	// cluster, not one per client.
	kube, err := kubernetes.NewForConfigAndClient(cfg, httpClient)
	if err != nil {
		return nil, &ConnectError{Context: name, Problem: "cannot build the client: " + kubeconfig.SafeError(err).Error(), Err: err}
	}
	dyn, err := dynamic.NewForConfigAndClient(cfg, httpClient)
	if err != nil {
		return nil, &ConnectError{Context: name, Problem: "cannot build the client: " + kubeconfig.SafeError(err).Error(), Err: err}
	}
	// The metadata client serves metadata-only informers (Secrets, H-8): it asks the server for
	// PartialObjectMetadata, so object content never crosses the wire.
	meta, err := metadata.NewForConfigAndClient(cfg, httpClient)
	if err != nil {
		return nil, &ConnectError{Context: name, Problem: "cannot build the client: " + kubeconfig.SafeError(err).Error(), Err: err}
	}
	// Discovery gets its own, larger budget, as kubectl gives it: a cluster without aggregated
	// discovery answers with one request per group version (and client-go repeats the round once
	// when a group fails), which at 5 per second would take many seconds.
	discoCfg := rest.CopyConfig(cfg)
	discoCfg.QPS, discoCfg.Burst = discoveryQPS, discoveryBurst
	discoCfg.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(discoveryQPS, discoveryBurst)
	disco, err := discovery.NewDiscoveryClientForConfigAndClient(discoCfg, httpClient)
	if err != nil {
		return nil, &ConnectError{Context: name, Problem: "cannot build the client: " + kubeconfig.SafeError(err).Error(), Err: err}
	}

	c := &Conn{
		contextName: name,
		namespace:   namespace,
		paths:       usable,
		server:      server,
		config:      cfg,
		httpClient:  httpClient,
		kube:        kube,
		dyn:         dyn,
		meta:        meta,
		disco:       disco,
		counter:     counter,
		warnings:    warnings,
	}
	if err := c.checkReachable(ctx, opts.ConnectTimeout); err != nil {
		c.Close()
		return nil, err
	}
	return c, nil
}

// memoryPersister keeps an auth provider's refreshed configuration in memory only: the provider
// holds it itself, so persisting is a no-op that reports success.
type memoryPersister struct{}

func (memoryPersister) Persist(map[string]string) error { return nil }

// checkReachable makes the one cheap request Connect pays for: GET /version. It proves the
// server answers and accepts the credentials (a request with bad credentials is rejected before
// authorization, even for /version).
func (c *Conn) checkReachable(ctx context.Context, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = DefaultConnectTimeout
	}
	tctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	info, err := discovery.ToDiscoveryInterfaceWithContext(c.disco).ServerVersionWithContext(tctx)
	if err == nil {
		if info != nil {
			c.version = info.GitVersion
		}
		return nil
	}
	return &ConnectError{Context: c.contextName, Problem: c.describeConnectFailure(ctx, tctx, timeout, err), Err: err}
}

// describeConnectFailure turns a failed request into a sentence a user can act on.
func (c *Conn) describeConnectFailure(ctx, tctx context.Context, timeout time.Duration, err error) string {
	var ue *url.Error
	switch {
	case ctx.Err() != nil:
		return "connecting was cancelled"
	case tctx.Err() != nil:
		return fmt.Sprintf("no answer from the API server %s within %s", c.server, timeout)
	case apierrors.IsUnauthorized(err):
		return fmt.Sprintf("the API server %s rejected the credentials (401 Unauthorized); log in again or check the user entry", c.server)
	case apierrors.IsForbidden(err):
		return fmt.Sprintf("the API server %s denied access (403 Forbidden)", c.server)
	case errors.As(err, &ue):
		// The URL adds nothing but noise; the inner error says what went wrong (connection
		// refused, TLS, DNS, an exec credential plugin that is not on PATH).
		return fmt.Sprintf("cannot reach the API server %s: %s", c.server, kubeconfig.SafeError(ue.Err).Error())
	default:
		return fmt.Sprintf("the API server %s did not answer as expected: %s", c.server, kubeconfig.SafeError(err).Error())
	}
}

// serverPathPrefix returns the path part of a server URL, without a trailing slash: requests to a
// cluster behind a path-routing proxy carry it in front of /api and /apis.
func serverPathPrefix(host string) string {
	u, err := url.Parse(host)
	if err != nil {
		return ""
	}
	return strings.TrimRight(u.Path, "/")
}

func describePaths(paths []string) string {
	if len(paths) == 0 {
		return "(none found)"
	}
	return strings.Join(paths, ", ")
}

// Context returns the kubeconfig context name.
func (c *Conn) Context() string { return c.contextName }

// Namespace returns the context's default namespace ("default" when the context sets none).
func (c *Conn) Namespace() string { return c.namespace }

// KubeconfigPaths returns the kubeconfig files the connection was loaded from, in precedence
// order. A terminal for this context puts its own file in front of these.
func (c *Conn) KubeconfigPaths() []string { return append([]string(nil), c.paths...) }

// Server returns the API server URL, without user information.
func (c *Conn) Server() string { return c.server }

// ServerVersion returns the API server's version as Connect saw it ("v1.36.1").
func (c *Conn) ServerVersion() string { return c.version }

// RESTConfig returns a copy of the client configuration, with the counting transport installed.
func (c *Conn) RESTConfig() *rest.Config { return rest.CopyConfig(c.config) }

// Kube returns the typed clientset.
func (c *Conn) Kube() kubernetes.Interface { return c.kube }

// Dynamic returns the dynamic client the cache builds its informers from.
func (c *Conn) Dynamic() dynamic.Interface { return c.dyn }

// Metadata returns the metadata client the cache builds its metadata-only informers from.
func (c *Conn) Metadata() metadata.Interface { return c.meta }

// Counter returns the connection's API request counter.
func (c *Conn) Counter() *RequestCounter { return c.counter }

// ServedResources returns the group/version/resources the cluster serves in their preferred
// version and that support list and watch (discovery verbs: whether the user may list them is
// an access-review question). The result is cached for ServedResourcesTTL; groups whose
// discovery fails (an aggregated API that is down) are left out, and such a partial result is
// cached for a minute only. The map is the caller's to keep.
func (c *Conn) ServedResources(ctx context.Context) (map[schema.GroupVersionResource]bool, error) {
	c.servedMu.Lock()
	defer c.servedMu.Unlock()
	if c.served != nil && time.Since(c.servedAt) < c.servedTTL {
		return copyGVRs(c.served), nil
	}
	if c.disco == nil {
		return nil, fmt.Errorf("context %q: discovery is not available", c.contextName)
	}
	lists, err := discovery.ToDiscoveryInterfaceWithContext(c.disco).ServerPreferredResourcesWithContext(ctx)
	partial := false
	if err != nil {
		var failed *discovery.ErrGroupDiscoveryFailed
		if !errors.As(err, &failed) || len(lists) == 0 {
			return nil, fmt.Errorf("context %q: discovery: %w", c.contextName, err)
		}
		partial = true
		groups := make([]string, 0, len(failed.Groups))
		for gv := range failed.Groups {
			groups = append(groups, gv.String())
		}
		slog.Warn("cluster: discovery left out groups that did not answer", "context", c.contextName, "groups", groups)
	}
	out := map[schema.GroupVersionResource]bool{}
	for _, l := range discovery.FilteredBy(discovery.SupportsAllVerbs{Verbs: []string{"list", "watch"}}, lists) {
		gv, err := schema.ParseGroupVersion(l.GroupVersion)
		if err != nil {
			continue
		}
		for _, r := range l.APIResources {
			if strings.Contains(r.Name, "/") { // a subresource
				continue
			}
			out[gv.WithResource(r.Name)] = true
		}
	}
	c.served, c.servedAt, c.servedTTL = out, time.Now(), ServedResourcesTTL
	if partial {
		c.servedTTL = partialServedResourcesTTL
	}
	return copyGVRs(out), nil
}

// InvalidateServedResources drops the cached discovery result, so the next ServedResources asks
// the server again. Call it when CustomResourceDefinitions or APIServices change.
func (c *Conn) InvalidateServedResources() {
	c.servedMu.Lock()
	c.served = nil
	c.servedMu.Unlock()
}

func copyGVRs(in map[schema.GroupVersionResource]bool) map[schema.GroupVersionResource]bool {
	out := make(map[schema.GroupVersionResource]bool, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// Close releases the connection's idle network connections. Requests in flight (watches of a
// cache that is still running) are not cancelled: stop the cache first.
func (c *Conn) Close() {
	if !c.closed.CompareAndSwap(false, true) {
		return
	}
	if c.httpClient != nil {
		c.httpClient.CloseIdleConnections()
	}
}

// toolDirs are directories where package managers and users put the programs a kubeconfig's exec
// credential plugins and a terminal's kubectl usually are.
var toolDirs = []string{"/opt/homebrew/bin", "/usr/local/bin", "/opt/local/bin"}

// EnsureToolPath appends the usual tool directories (/opt/homebrew/bin, /usr/local/bin,
// /opt/local/bin, $HOME/.local/bin, $HOME/bin) that exist and are missing from PATH. A GUI app
// started from Finder or a desktop launcher does not inherit the PATH a login shell sets, so exec
// credential plugins (aws, gcloud, kubelogin) and kubectl would not be found.
//
// This is a stopgap for gap H-6 (docs/security/credentials.md): reading the login shell's
// environment is not designed yet. Call it once at start-up, before the first Connect; it changes
// the process environment. It does nothing on Windows.
func EnsureToolPath() {
	if runtime.GOOS == "windows" {
		return
	}
	dirs := append([]string(nil), toolDirs...)
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		dirs = append(dirs, filepath.Join(home, ".local", "bin"), filepath.Join(home, "bin"))
	}
	path := os.Getenv("PATH")
	have := map[string]bool{}
	for _, d := range filepath.SplitList(path) {
		if d != "" {
			have[filepath.Clean(d)] = true
		}
	}
	var added []string
	for _, d := range dirs {
		if have[d] {
			continue
		}
		if fi, err := os.Stat(d); err == nil && fi.IsDir() {
			added = append(added, d)
			have[d] = true
		}
	}
	if len(added) == 0 {
		return
	}
	parts := added
	if path != "" {
		parts = append([]string{path}, added...)
	}
	if err := os.Setenv("PATH", strings.Join(parts, string(os.PathListSeparator))); err != nil {
		slog.Warn("cluster: cannot extend PATH", "err", err)
		return
	}
	slog.Info("cluster: added tool directories to PATH", "dirs", added)
}
