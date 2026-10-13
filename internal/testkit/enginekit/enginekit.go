// Package enginekit holds test helpers that need the engine: a connector over an in-memory source,
// so that a test runs the real engine and its commands (the window's tests among them, which may
// not import the cache or the cluster layer themselves), and a recorder of the writes the Flux
// actions send. Test-only code, as all of testkit.
package enginekit

import (
	"context"
	"slices"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// MemoryConnector connects its one context to a fresh cache over Src: the given built-in profiles,
// the custom kinds watch on when CustomResources is set, and Writer for the writes. The backend has
// a (zero) connection, so that commands that need one are enabled; nothing dials a cluster.
type MemoryConnector struct {
	Context         string
	Src             *testkit.MemorySource
	Profiles        []*model.KindProfile
	CustomResources bool
	Writer          engine.Actions
}

// Contexts lists the one context, as the current one.
func (c MemoryConnector) Contexts() ([]engine.ContextInfo, error) {
	return []engine.ContextInfo{{Name: c.Context, Current: true}}, nil
}

// Connect makes the backend.
func (c MemoryConnector) Connect(context.Context, string) (*engine.Backend, error) {
	ch := cache.New(c.Src, cache.Options{WarmPeriod: time.Minute})
	return &engine.Backend{Context: c.Context, Namespace: "default", Cache: ch, Profiles: c.Profiles,
		CustomResources: c.CustomResources, Close: ch.Close, Conn: &cluster.Conn{}, Actions: c.Writer}, nil
}

// FluxRecorder records the Flux actions' patches (engine.FluxActions) as "resource namespace/name
// body", the body as cluster.FluxPatch makes it at At; the other writes do nothing.
type FluxRecorder struct {
	At time.Time

	mu      sync.Mutex
	patches []string
}

var (
	_ engine.Actions     = (*FluxRecorder)(nil)
	_ engine.FluxActions = (*FluxRecorder)(nil)
)

// RestartDeployment implements engine.Actions; it records nothing.
func (r *FluxRecorder) RestartDeployment(context.Context, string, string, string) error { return nil }

// DeletePod implements engine.Actions; it records nothing.
func (r *FluxRecorder) DeletePod(context.Context, string, string, string) error { return nil }

// PatchFlux records the patch.
func (r *FluxRecorder) PatchFlux(_ context.Context, gvr schema.GroupVersionResource, namespace, name, uid string, req cluster.FluxRequest) error {
	body, err := cluster.FluxPatch(r.At, uid, req)
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.patches = append(r.patches, gvr.Resource+" "+namespace+"/"+name+" "+string(body))
	r.mu.Unlock()
	return nil
}

// Patches returns the recorded patches in order.
func (r *FluxRecorder) Patches() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.patches)
}
