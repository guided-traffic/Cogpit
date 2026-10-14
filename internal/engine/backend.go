package engine

import (
	"context"
	"log/slog"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/metrics"
	"github.com/guided-traffic/cogpit/internal/model"
)

// Backend is one connected cluster as the engine sees it.
type Backend struct {
	// Context is the kubeconfig context name ("demo" for the demo cluster).
	Context string
	// Namespace is the context's default namespace.
	Namespace string
	// Cache serves every read.
	Cache *cache.Cache
	// Conn is the connection for streams and actions; nil for the demo cluster, where logs,
	// shells, terminals and mutating commands are disabled.
	Conn *cluster.Conn
	// Actions performs the cluster writes of the action commands. Nil means Conn does; tests
	// substitute a fake here.
	Actions Actions
	// Profiles are the built-in kinds this cluster serves, in sidebar order. AllProfiles adds the
	// kinds of its custom resource definitions.
	Profiles []*model.KindProfile
	// CustomResources makes the engine watch the cluster's custom resource definitions (one watch
	// through the cache, for as long as the backend is connected) and offer a view for each kind
	// they define. The real cluster connector sets it; the demo cluster and backends that leave it
	// off have the built-in kinds only.
	CustomResources bool
	// Metrics polls the cluster's metrics while a view or the drawer that shows them is open
	// (metrics.go, architecture §9). Nil means the backend has none; it is closed with the backend.
	Metrics *metrics.Hub
	// PrometheusDetector is the Prometheus half of the metrics detection, kept so that the cluster's
	// settings can set the override (SetOverride, then Metrics.Forget); nil without a hub of a real
	// cluster.
	PrometheusDetector *metrics.PrometheusDetector
	// MetricsSettings keeps the cluster's Prometheus override in Cogpit's preferences
	// (metricssettings.go); nil keeps it for the connection only.
	MetricsSettings *MetricsSettings
	// Demo marks the synthetic demo cluster.
	Demo bool
	// DemoStats reports the demo cluster's counters; nil for real clusters.
	DemoStats func() any
	// Close stops the cache and closes the connection.
	Close func()
	// SourceFile is the kubeconfig file of a kubeconfig source the connection was made from
	// (kubeconfigs.go); empty for the demo cluster and a connector without sources. When the file goes
	// away, or its source is removed, the session closes the connection.
	SourceFile string

	// Access answers the access reviews of the connected user (access.go); nil allows everything,
	// and the API server decides.
	Access Access

	// custom is the custom kinds watch (customkinds.go); the Session sets it before it publishes
	// the backend, nil without CustomResources.
	custom *customKinds
	// local holds the objects of the local kinds (kindscope.go), made on first use.
	local localStore
	// apps is the Applications view's cache (applications.go), made on first use.
	apps appsStore
	// forwards are the backend's running port forwards (portforward.go); they end with it.
	forwards forwardSet
	// ns is what decides whether a namespaced kind is read namespace by namespace
	// (accessiblens_plan.go).
	ns nsState
	// scrub replaces the cached Secrets' values in what agents read (secretscrub.go, H-82).
	scrub secretScrub
}

// Actions are the cluster writes of the MVP's action commands. *cluster.Conn implements them.
type Actions interface {
	// RestartDeployment restarts a Deployment as `kubectl rollout restart` does; a non-empty uid
	// must match the Deployment's (cluster.ErrReplaced otherwise).
	RestartDeployment(ctx context.Context, namespace, name, uid string) error
	// DeletePod deletes one pod with the default grace period; a non-empty uid must match the
	// pod's, so that a replacement with the same name is not deleted.
	DeletePod(ctx context.Context, namespace, name, uid string) error
}

// Writer returns what performs the backend's writes, or nil when it cannot write (the demo
// cluster).
func (b *Backend) Writer() Actions {
	if b.Actions != nil {
		return b.Actions
	}
	if b.Conn != nil {
		return b.Conn
	}
	return nil
}

// CanWrite reports whether the backend can run mutating commands: false for the demo cluster.
func (b *Backend) CanWrite() bool { return b.Writer() != nil }

// close stops the backend's cache and connection, recovering a panicking Close.
func (b *Backend) close(log *slog.Logger) {
	if b == nil {
		return
	}
	b.ns.stop() // the session's replan loop (accessiblens_plan.go)
	if b.custom != nil {
		b.custom.stop()
	}
	b.forwards.stopAll()
	b.apps.close() // before the cluster's cache: it releases its handles on the workloads
	b.local.close()
	if b.Metrics != nil {
		b.Metrics.Close()
	}
	if b.Close == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			log.Error("engine: closing the backend failed", "context", b.Context, "panic", r)
		}
	}()
	b.Close()
}

// Profile returns the backend's profile for a kind key: a built-in kind ("pods"), or a kind of
// the cluster's custom resource definitions ("gateways.gateway.networking.k8s.io"). A custom
// kind's profile changes when its definition does: callers hold on to it for one operation only.
func (b *Backend) Profile(kind string) (*model.KindProfile, bool) {
	if p := b.builtinProfile(kind); p != nil {
		return p, true
	}
	if set := b.customSet(); set != nil {
		if p, ok := set.byKey[kind]; ok {
			return p, true
		}
	}
	return nil, false
}

// AllProfiles returns every kind the cluster serves: the built-in profiles in sidebar order, then
// the kinds of its custom resource definitions sorted by API group and Kind. Callers must not
// modify the result; it is a snapshot that a change of the definitions replaces.
func (b *Backend) AllProfiles() []*model.KindProfile {
	custom := b.CustomProfiles()
	if len(custom) == 0 {
		return b.Profiles
	}
	out := make([]*model.KindProfile, 0, len(b.Profiles)+len(custom))
	out = append(out, b.Profiles...)
	return append(out, custom...)
}

// ContextInfo is one entry of the context picker.
type ContextInfo struct {
	Name      string `json:"name"`
	Cluster   string `json:"cluster,omitempty"`
	User      string `json:"user,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Server    string `json:"server,omitempty"`
	Current   bool   `json:"current,omitempty"` // the kubeconfig's current-context
	Demo      bool   `json:"demo,omitempty"`    // the synthetic demo cluster
	// Source, KubeContext and File describe a context from a kubeconfig source the user added in
	// Cogpit (kubeconfigs.go): the file's label ("dev.yaml", "kubeconfigs/team.yaml"),
	// the context's name in that file (Name is then "<context> [<label>]") and the file's path.
	// Empty for the demo cluster and a connector without sources.
	Source      string `json:"source,omitempty"`
	KubeContext string `json:"kubeContext,omitempty"`
	File        string `json:"file,omitempty"`
	// ConnectProblem is set when the last attempt to connect to this context failed, or its
	// connection was closed because its kubeconfig file went away; cluster.connectLog has the log.
	ConnectProblem bool `json:"connectProblem,omitempty"`
	// Exec is ExecApproved, ExecRequired or ExecChanged for a source context whose user runs an
	// exec credential plugin or names a token file outside its folder (D-17), which run or are read
	// only once the user allowed it, or whose cluster skips the certificate check (A28), which
	// connects only once the user allowed it; empty otherwise. ExecKind says which: "program",
	// "tokenFile", "both", or "insecure" when the unchecked certificate is all (a plugin's approval
	// covers its context's check).
	Exec     string `json:"exec,omitempty"`
	ExecKind string `json:"execKind,omitempty"`
	// Insecure is set when the context's cluster skips the check of the server's certificate
	// (insecure-skip-tls-verify): the picker warns, a source context asks once before its first
	// connect (Exec), and connecting writes a warning into the connect log and marks the
	// connection (Connection.TLSUnchecked; tlswarning.go).
	Insecure bool `json:"insecure,omitempty"`
}

// Connector lists the contexts a user can connect to and connects one.
type Connector interface {
	Contexts() ([]ContextInfo, error)
	Connect(ctx context.Context, name string) (*Backend, error)
}
