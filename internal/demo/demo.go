// Package demo is a synthetic cluster for `cogpit --demo`: 10,000 pods, a few hundred services
// with known cluster IPs, deployments, namespaces, nodes and events, and a configurable churn of
// watch events (default 1,000 per second). It feeds the real cache through a cache.Source, so the
// demo exercises the same informers, engine and UI as a real cluster.
//
// Layer L2: it implements a cache.Source.
package demo

import (
	"context"

	"github.com/guided-traffic/cogpit/internal/cache"
)

// Options size the synthetic cluster.
type Options struct {
	Pods            int     // default 10,000
	Services        int     // default 300
	Namespaces      int     // default 20
	Nodes           int     // default 50
	EventsPerSecond float64 // watch events per second across all kinds; default 1,000; 0 with NoChurn
	NoChurn         bool    // a static cluster, for tests
	Seed            int64   // random seed; the same seed gives the same cluster
}

// Stats describe the synthetic cluster for the status bar.
type Stats struct {
	Objects      int     `json:"objects"`
	EventsSent   int64   `json:"eventsSent"`
	EventsPerSec float64 `json:"eventsPerSec"`
}

// Cluster is a running synthetic cluster.
type Cluster struct {
	opts Options
}

// New builds the synthetic cluster's initial objects. Nothing changes until Start.
func New(opts Options) *Cluster { return &Cluster{opts: opts} }

// Source returns the cache.Source the cache builds its informers from.
func (c *Cluster) Source() cache.Source { return nil }

// Start runs the churn until ctx ends or Stop is called.
func (c *Cluster) Start(ctx context.Context) { _ = ctx }

// Stop ends the churn.
func (c *Cluster) Stop() {}

// Stats returns counts for the status bar.
func (c *Cluster) Stats() Stats { return Stats{} }
