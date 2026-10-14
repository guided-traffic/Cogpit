package commands

import (
	"context"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// metrics.get: the current CPU and memory of pods and nodes, for agents (G12, architecture §9).
// The drawer's Metrics section and the lists' CPU and Memory columns are not commands: they follow
// the open view and drawer.

type metricsGetParams struct {
	Kind      string `json:"kind" jsonschema:"pods or nodes"`
	Namespace string `json:"namespace,omitempty" jsonschema:"pods only: the namespace; omitted: all namespaces"`
	Name      string `json:"name,omitempty" jsonschema:"one pod (with its namespace) or one node; omitted: all of them"`
	Limit     int    `json:"limit,omitempty" jsonschema:"at most this many items; default 500"`
}

// metricsGroup is the Metrics API's group.
const metricsGroup = "metrics.k8s.io"

// metricsReads are the Metrics API reads behind metrics.get: the group's discovery, and get or
// list on its pods and nodes.
var metricsReads = []engine.Verb{
	{Verb: nwVerbGet, Group: metricsGroup, Resource: "pods"},
	{Verb: nwVerbList, Group: metricsGroup, Resource: "pods"},
	{Verb: nwVerbGet, Group: metricsGroup, Resource: "nodes"},
	{Verb: nwVerbList, Group: metricsGroup, Resource: "nodes"},
	// The cluster's Prometheus, when one is found: queries go through the API server's service
	// proxy, on the one service detection found (get services/proxy), and nowhere else.
	{Verb: nwVerbGet, Resource: "services", Subresource: "proxy"},
}

func registerMetrics(g *registrar) {
	registerPrometheusSetting(g)
	add(g, engine.Spec[metricsGetParams]{
		ID: "metrics.get", Label: "Get Metrics", Danger: engine.DangerRead, Verbs: metricsReads,
		Description: "Reads the current CPU (millicores) and memory (bytes) usage of a pod, of the pods of a namespace or of all " +
			"namespaces, of a node, or of all nodes, from the cluster's Kubernetes Metrics Server (metrics.k8s.io), or from its " +
			"Prometheus (read through the API server's service proxy) when the Metrics Server fails or is missing; the result's " +
			"source says which, and its providers say what Cogpit knows of each (answering or not, the last good answer, the last " +
			"failure's class, what it serves). The numbers are the latest measurement (the kubelets' " +
			"every 15 seconds at best), per container for pods; a result newer than 15 seconds " +
			"is served without a request, otherwise one request is sent and nothing keeps polling. Without either the " +
			"result's status says so (notServed, forbidden, unavailable) and carries no items: that is a state, not an error. " +
			"Every pod of a user who may list pods in some namespaces only is read namespace by namespace: forbidden then names " +
			"the namespaces whose pods' metrics the user may not read, whose pods are left out. " +
			"Requests and limits are in object_get's containers sections; node capacity in its Capacity and Allocatable tables.",
		Enabled: func(s *engine.Session, _ metricsGetParams) error { _, err := connected(s); return err },
		Run: func(ctx context.Context, s *engine.Session, p metricsGetParams) (any, error) {
			return s.MetricsGet(ctx, p.Kind, p.Namespace, p.Name, p.Limit)
		},
	})
}
