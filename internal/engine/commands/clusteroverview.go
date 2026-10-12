package commands

import (
	"context"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// The cluster Overview (ui-parity › Cluster overview and charts, G12): a page, not a kind. The
// engine builds it from the cache and the metrics hub; ui.openView "cluster/overview" shows it, and
// agents read it here.

type clusterOverviewParams struct {
	Namespaces []string `json:"namespaces,omitempty" jsonschema:"namespaces whose issues to list (a node's issues are always listed); omitted: the namespace filter the user has set"`
	Series     bool     `json:"series,omitempty" jsonschema:"true: include each chart's points over its time range (while the page is open; otherwise there are none)"`
}

func registerClusterOverview(g *registrar) {
	add(g, engine.Spec[clusterOverviewParams]{
		ID: "cluster.overview", Label: "Cluster Overview", MenuPath: "View/Cluster Overview", Danger: engine.DangerRead,
		Verbs: cacheReads,
		Description: "Reads the cluster Overview: the node groups (master: control-plane nodes; worker: the others), the " +
			"group and time range the charts show, the charts CPU (cores), Memory (bytes) and Pods for that group with " +
			"their current usage, requests and limits (sums over the pods that have not finished on those nodes), " +
			"allocatable capacity and capacity, and the Issues: pods with a warning, nodes not Ready or under pressure, " +
			"Deployments, Stateful Sets and Daemon Sets short of ready pods, and warning events of the last hour, each " +
			"with its object (ref), kind, message and since when. Read from the cache; while the page is open the " +
			"charts carry the metrics history (with series: true, their points). ui_openView with view " +
			"cluster/overview shows the page to the user; ui_select with an issue's ref opens its details there.",
		Enabled: func(s *engine.Session, _ clusterOverviewParams) error { _, err := connected(s); return err },
		Run: func(ctx context.Context, s *engine.Session, p clusterOverviewParams) (any, error) {
			return s.ClusterOverview(ctx, p.Namespaces, p.Series)
		},
	})
}
