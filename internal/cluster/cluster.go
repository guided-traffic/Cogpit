// Package cluster owns connections to clusters: kubeconfig loading, client-go configuration and
// authentication, discovery, the counting transport, and the few direct API calls (actions,
// streams) that the cache cannot serve.
//
// Layer L1. Only cluster, cache, terminal, helm and metrics talk to the API server.
package cluster

import (
	"context"
	"errors"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/guided-traffic/cogpit/internal/kubeconfig"
)

// ErrNotImplemented marks a Phase 0 stub.
var ErrNotImplemented = errors.New("not implemented")

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
}

// ListContexts returns the contexts of the merged kubeconfig files (default loading rules when
// paths is empty), sorted by name.
func ListContexts(paths []string) ([]kubeconfig.Context, error) {
	_ = paths
	return nil, ErrNotImplemented
}

// Conn is one connection to one kubeconfig context: its client configuration, clients, cached
// discovery and request counter. It is safe for concurrent use.
type Conn struct {
	contextName string
	namespace   string
	paths       []string
	config      *rest.Config
	kube        kubernetes.Interface
	dyn         dynamic.Interface
	counter     *RequestCounter
}

// Connect builds a connection for opts.Context. It verifies the server is reachable with one
// discovery request and returns an error a user can act on otherwise. Credentials never appear
// in the error.
func Connect(ctx context.Context, opts Options) (*Conn, error) {
	_, _ = ctx, opts
	return nil, ErrNotImplemented
}

// Context returns the kubeconfig context name.
func (c *Conn) Context() string { return c.contextName }

// Namespace returns the context's default namespace ("default" when the context sets none).
func (c *Conn) Namespace() string { return c.namespace }

// KubeconfigPaths returns the kubeconfig files the connection was loaded from, in precedence
// order. A terminal for this context puts its own file in front of these.
func (c *Conn) KubeconfigPaths() []string { return c.paths }

// RESTConfig returns a copy of the client configuration, with the counting transport installed.
func (c *Conn) RESTConfig() *rest.Config { return rest.CopyConfig(c.config) }

// Kube returns the typed clientset.
func (c *Conn) Kube() kubernetes.Interface { return c.kube }

// Dynamic returns the dynamic client the cache builds its informers from.
func (c *Conn) Dynamic() dynamic.Interface { return c.dyn }

// Counter returns the connection's API request counter.
func (c *Conn) Counter() *RequestCounter { return c.counter }

// ServedResources returns the group/version/resources the cluster serves and the user may list
// and watch, from cached discovery.
func (c *Conn) ServedResources(ctx context.Context) (map[schema.GroupVersionResource]bool, error) {
	_ = ctx
	return nil, ErrNotImplemented
}

// Close releases the connection's resources.
func (c *Conn) Close() {}
