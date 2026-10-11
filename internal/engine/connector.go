package engine

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"slices"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/demo"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

// DemoContext is the context name of the synthetic demo cluster.
const DemoContext = "demo"

// errDemoUnavailable is returned when the demo package provides no source.
var errDemoUnavailable = errors.New("the demo cluster is not available in this build")

// ClusterConnector is the Connector of the desktop app and the CLI: the contexts of the user's
// kubeconfig files through package cluster, and optionally the synthetic demo cluster.
type ClusterConnector struct {
	// KubeconfigPaths are the kubeconfig files, merged in order. Empty means client-go's default
	// loading rules ($KUBECONFIG, else ~/.kube/config).
	KubeconfigPaths []string
	// IncludeDemo offers the demo cluster as the context DemoContext. It shadows a kubeconfig
	// context of the same name.
	IncludeDemo bool
	// DemoOptions size the demo cluster.
	DemoOptions demo.Options
	// WarmPeriod is how long the cache keeps an unused informer; zero means
	// cache.DefaultWarmPeriod.
	WarmPeriod time.Duration
	// Logger receives connection warnings; nil means slog.Default().
	Logger *slog.Logger
}

func (c *ClusterConnector) logger() *slog.Logger {
	if c.Logger != nil {
		return c.Logger
	}
	return slog.Default()
}

// Contexts returns the kubeconfig contexts, sorted by name, and the demo cluster last when
// IncludeDemo is set. If the kubeconfig cannot be loaded, the demo entry is still returned,
// together with the error.
func (c *ClusterConnector) Contexts() ([]ContextInfo, error) {
	ctxs, err := cluster.ListContexts(c.KubeconfigPaths)
	out := make([]ContextInfo, 0, len(ctxs)+1)
	for _, k := range ctxs {
		if c.IncludeDemo && k.Name == DemoContext {
			c.logger().Warn("engine: the kubeconfig context is shadowed by the demo cluster", "context", k.Name)
			continue
		}
		out = append(out, ContextInfo{
			Name: k.Name, Cluster: k.Cluster, User: k.User, Namespace: k.Namespace,
			Server: k.Server, Current: k.Current,
		})
	}
	if c.IncludeDemo {
		out = append(out, ContextInfo{Name: DemoContext, Namespace: "default", Demo: true})
	}
	if err != nil {
		return out, fmt.Errorf("kubeconfig: %w", err)
	}
	return out, nil
}

// Connect connects one context: the demo cluster, or a kubeconfig context whose served kinds
// (from discovery) select the built-in profiles. If discovery fails, every built-in profile is
// offered and a kind the cluster lacks shows its error in its view. A real cluster's backend also
// watches its custom resource definitions (Backend.CustomResources).
func (c *ClusterConnector) Connect(ctx context.Context, name string) (*Backend, error) {
	if c.IncludeDemo && name == DemoContext {
		return c.connectDemo()
	}
	conn, err := cluster.Connect(ctx, cluster.Options{KubeconfigPaths: c.KubeconfigPaths, Context: name})
	if err != nil {
		return nil, err
	}
	profiles := kinds.Builtin()
	customResources := true
	served, err := conn.ServedResources(ctx)
	if err != nil {
		c.logger().Warn("engine: discovery failed; offering every built-in kind", "context", name, "err", err)
	} else {
		profiles = servedProfiles(profiles, served)
		customResources = served[crdsGVR]
	}
	// The local kinds (port forwards) are Cogpit's own objects: every real cluster offers them.
	profiles = append(slices.Clip(profiles), kinds.Local()...)
	ch := cache.New(cache.DynamicSource(conn.Dynamic(), conn.Metadata()), cache.Options{WarmPeriod: c.WarmPeriod, Logger: c.Logger})
	ctxName := conn.Context()
	if ctxName == "" {
		ctxName = name
	}
	var once sync.Once
	return &Backend{
		Context: ctxName, Namespace: conn.Namespace(), Cache: ch, Conn: conn, Profiles: profiles,
		CustomResources: customResources,
		Close: func() {
			once.Do(func() {
				ch.Close()
				conn.Close()
			})
		},
	}, nil
}

// servedProfiles keeps the profiles whose group, version and resource the cluster serves.
func servedProfiles(all []*model.KindProfile, served map[schema.GroupVersionResource]bool) []*model.KindProfile {
	out := make([]*model.KindProfile, 0, len(all))
	for _, p := range all {
		if served[p.GVR()] {
			out = append(out, p)
		}
	}
	return out
}

// demoProfiles keeps the built-in profiles of the kinds the demo cluster serves, as discovery does
// for a real cluster: a kind the demo has no objects for is not offered, rather than shown as an
// error.
func demoProfiles() []*model.KindProfile {
	served := map[schema.GroupVersionResource]bool{}
	for _, gvr := range demo.Resources() {
		served[gvr] = true
	}
	return servedProfiles(kinds.Builtin(), served)
}

// connectDemo builds the synthetic cluster and starts its churn.
func (c *ClusterConnector) connectDemo() (*Backend, error) {
	dc := demo.New(c.DemoOptions)
	src := dc.Source()
	if src == nil {
		return nil, errDemoUnavailable
	}
	ch := cache.New(src, cache.Options{WarmPeriod: c.WarmPeriod, Logger: c.Logger})
	churn, cancel := context.WithCancel(context.Background())
	log := c.logger()
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error("engine: demo churn panicked", "panic", fmt.Sprint(r), "stack", string(debug.Stack()))
			}
		}()
		dc.Start(churn)
	}()
	var once sync.Once
	return &Backend{
		Context: DemoContext, Namespace: "default", Cache: ch, Profiles: demoProfiles(), Demo: true,
		DemoStats: func() any { return dc.Stats() },
		Close: func() {
			once.Do(func() {
				cancel()
				dc.Stop()
				ch.Close()
			})
		},
	}, nil
}
