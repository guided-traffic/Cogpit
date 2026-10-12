// Package metrics finds out where a cluster's resource usage comes from and polls it while
// something visible shows it (architecture §9).
//
// A [Provider] answers "what do these pods and nodes use right now". The first one is the
// Kubernetes Metrics Server ([Server], the metrics.k8s.io API); Prometheus through the API
// server's service proxy comes next and plugs into the same places: a [Detector] finds a provider
// (and says why it found none), a [Hub] polls it while subscribed and keeps the latest results and
// a short time series per watched object. Metrics cannot be watched, so polling is the documented
// exception to "never poll what can be watched" (AGENTS.md, cache first), and it happens only
// while a view or a drawer that shows metrics is open.
//
// A missing API, a missing permission and an unreachable metrics-server are [State]s, not errors:
// the UI shows them and the lists stay blank.
//
// Layer L2: it talks to the API server, through the cluster's counting transport.
package metrics

import (
	"context"
	"errors"

	"github.com/guided-traffic/cogpit/internal/model"
)

// Status says whether a cluster has usable metrics.
type Status string

// Statuses.
const (
	// StatusUnknown: nothing asked yet.
	StatusUnknown Status = "unknown"
	// StatusAvailable: a provider answers.
	StatusAvailable Status = "available"
	// StatusNotServed: the cluster serves no metrics API (no metrics-server installed).
	StatusNotServed Status = "notServed"
	// StatusForbidden: the API exists, but the user may not read it.
	StatusForbidden Status = "forbidden"
	// StatusUnavailable: the API is registered but fails (metrics-server down or not ready), or
	// the request failed for another reason.
	StatusUnavailable Status = "unavailable"
)

// State is what is known about a cluster's metrics: which provider, or why none.
type State struct {
	Status Status `json:"status"`
	// Provider is the provider's id ("metrics-server") when Status is available.
	Provider string `json:"provider,omitempty"`
	// Label is the provider as Lens names it in its source line: "Kubernetes Metrics Server".
	Label string `json:"label,omitempty"`
	// Message says why metrics are not available, in a sentence the UI shows.
	Message string `json:"message,omitempty"`
}

// Usable reports whether a provider answers.
func (s State) Usable() bool { return s.Status == StatusAvailable }

// Capabilities say what a provider can draw. The Metrics Server gives the current CPU and memory
// only; Prometheus will add history (a time range of its own), network and filesystem.
type Capabilities struct {
	// History: the provider answers range queries, so a chart needs no series kept in memory.
	History bool `json:"history,omitempty"`
	// Network and Filesystem: the provider has these variants.
	Network    bool `json:"network,omitempty"`
	Filesystem bool `json:"filesystem,omitempty"`
}

// ErrNoData means the provider has no measurement for the object yet (a pod that started a moment
// ago, a node the metrics-server has not scraped): not an error state of the cluster's metrics.
var ErrNoData = errors.New("no metrics for the object yet")

// Provider reads current usage. Implementations are safe for concurrent use.
type Provider interface {
	// ID is the provider's stable id: "metrics-server".
	ID() string
	// Label is the provider as the UI names it: "Kubernetes Metrics Server".
	Label() string
	// Capabilities say which variants the provider can draw.
	Capabilities() Capabilities
	// ListPods returns the usage of the pods of a namespace, or of all namespaces for "".
	ListPods(ctx context.Context, namespace string) ([]model.PodUsage, error)
	// GetPod returns one pod's usage, or ErrNoData.
	GetPod(ctx context.Context, namespace, name string) (model.PodUsage, error)
	// ListNodes returns the usage of every node.
	ListNodes(ctx context.Context) ([]model.NodeUsage, error)
	// GetNode returns one node's usage, or ErrNoData.
	GetNode(ctx context.Context, name string) (model.NodeUsage, error)
}

// Detector finds a cluster's provider. It returns the provider when the state is available, else
// nil and the state that says why (not served, forbidden, unavailable). It makes its own requests
// (discovery) and is asked rarely: the hub keeps the answer for a while.
type Detector interface {
	Detect(ctx context.Context) (Provider, State)
}

// DetectorFunc adapts a function to a Detector.
type DetectorFunc func(ctx context.Context) (Provider, State)

// Detect calls f.
func (f DetectorFunc) Detect(ctx context.Context) (Provider, State) { return f(ctx) }

// Fixed is a Detector that always finds p: the synthetic demo cluster's.
func Fixed(p Provider) Detector {
	return DetectorFunc(func(context.Context) (Provider, State) {
		return p, State{Status: StatusAvailable, Provider: p.ID(), Label: p.Label()}
	})
}
