package commands

import (
	"context"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// The Workloads Overview (resource-parity › Workloads › Overview): a page, not a kind. The engine
// builds it from the cache; ui.openView "workloads/overview" shows it, and agents read it here.

type overviewParams struct {
	engine.OnCluster
	Namespaces []string `json:"namespaces,omitempty" jsonschema:"namespaces to count; omitted: the namespace filter the user has set (every namespace for a cluster the window does not show)"`
	Events     bool     `json:"events,omitempty" jsonschema:"true: include the page's events table, the events of those namespaces as the page lists them (newest first)"`
}

// OverviewResult is workloads.overview's result.
type OverviewResult struct {
	engine.WorkloadsOverviewModel
	// Events is the page's events table: Type, Source, Namespace, Involved Object, Message, Count,
	// Age, Last Seen. Only with events: true.
	Events *engine.TableSnapshot `json:"events,omitempty"`
}

func registerOverview(g *registrar) {
	add(g, engine.Spec[overviewParams]{
		ID: "workloads.overview", Label: "Workloads Overview", MenuPath: "View/Workloads Overview", Danger: engine.DangerRead,
		Verbs: cacheReads,
		Description: "Reads the Workloads Overview: for each workload kind the cluster serves (pods, deployments, daemon sets, " +
			"stateful sets, replica sets, jobs, cron jobs) the number of objects in the selected namespaces and their health, " +
			"as counts per status token (ok, succeeded, pending, warning, error, terminating, unknown, and none); with events, " +
			"also the events of those namespaces. Read from the cache. ui_openView with view workloads/overview shows the page " +
			"to the user.",
		Enabled: func(s *engine.Session, p overviewParams) error { _, err := connectedOn(s, p.Context); return err },
		Run: func(ctx context.Context, s *engine.Session, p overviewParams) (any, error) {
			m, events, err := s.WorkloadsOverview(ctx, p.Context, p.Namespaces, p.Events)
			if err != nil {
				return nil, err
			}
			return OverviewResult{WorkloadsOverviewModel: *m, Events: events}, nil
		},
	})
}
