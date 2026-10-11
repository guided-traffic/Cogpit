package engine

import (
	"context"
	"log/slog"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/cluster"
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
	// Demo marks the synthetic demo cluster.
	Demo bool
	// DemoStats reports the demo cluster's counters; nil for real clusters.
	DemoStats func() any
	// Close stops the cache and closes the connection.
	Close func()

	// custom is the custom kinds watch (customkinds.go); the Session sets it before it publishes
	// the backend, nil without CustomResources.
	custom *customKinds
}

// Actions are the cluster writes of the MVP's action commands. *cluster.Conn implements them.
type Actions interface {
	// RestartDeployment restarts a Deployment as `kubectl rollout restart` does.
	RestartDeployment(ctx context.Context, namespace, name string) error
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
	if b.custom != nil {
		b.custom.stop()
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
}

// Connector lists the contexts a user can connect to and connects one.
type Connector interface {
	Contexts() ([]ContextInfo, error)
	Connect(ctx context.Context, name string) (*Backend, error)
}
