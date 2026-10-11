package engine

import (
	"context"

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
	// Profiles are the kinds this cluster serves, in sidebar order.
	Profiles []*model.KindProfile
	// Demo marks the synthetic demo cluster.
	Demo bool
	// DemoStats reports the demo cluster's counters; nil for real clusters.
	DemoStats func() any
	// Close stops the cache and closes the connection.
	Close func()
}

// Profile returns the backend's profile for a kind key.
func (b *Backend) Profile(kind string) (*model.KindProfile, bool) {
	for _, p := range b.Profiles {
		if p.Key == kind {
			return p, true
		}
	}
	return nil, false
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
