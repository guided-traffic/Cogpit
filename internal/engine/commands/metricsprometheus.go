package commands

import (
	"context"
	"strings"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/metrics"
)

// cluster.prometheus and cluster.setPrometheus: which Prometheus the connected cluster's metrics
// come from (G12; Lens: Cluster settings → Metrics, the Prometheus service address and a path
// prefix). The setting is kept in Cogpit's own preferences per context, never in the kubeconfig or
// the cluster.

type prometheusParams struct {
	Address string `json:"address,omitempty" jsonschema:"the Prometheus service as namespace/service:port, the port by number or name (monitoring/prometheus-server:80); omitted or auto: detect it by its well-known service names"`
	Prefix  string `json:"prefix,omitempty" jsonschema:"the path Prometheus serves under (its --web.route-prefix), such as /prometheus; omitted: none"`
}

// autoAddress names detection in cluster.setPrometheus.
const autoAddress = "auto"

func registerPrometheusSetting(g *registrar) {
	add(g, engine.Spec[struct{}]{
		ID: "cluster.prometheus", Label: "Prometheus Setting", Danger: engine.DangerRead,
		Description: "Returns which Prometheus the connected cluster's metrics come from: auto (found among the " +
			"cluster's Services by its well-known names) or the service address namespace/service:port and path prefix " +
			"the user set, and whether the setting is kept in Cogpit's preferences. The metrics providers' state is in " +
			"metrics_get's providers.",
		Enabled: func(s *engine.Session, _ struct{}) error { _, err := connected(s); return err },
		Run: func(_ context.Context, s *engine.Session, _ struct{}) (any, error) {
			return s.PrometheusSetting()
		},
	})
	add(g, engine.Spec[prometheusParams]{
		ID: "cluster.setPrometheus", Label: "Set Prometheus", Danger: engine.DangerUI,
		Description: "Sets which Prometheus the connected cluster's metrics come from, as Lens's cluster settings do: a " +
			"service address namespace/service:port and an optional path prefix, read through the API server's service " +
			"proxy, or auto to find it by its well-known service names again. It is kept in Cogpit's own preferences for " +
			"this context, never written to the kubeconfig or the cluster, and sends no request itself: the next chart " +
			"or list that needs Prometheus probes that service alone (one GET of its buildinfo). Returns the setting.",
		Enabled: func(s *engine.Session, _ prometheusParams) error { _, err := connected(s); return err },
		Run: func(_ context.Context, s *engine.Session, p prometheusParams) (any, error) {
			addr := strings.TrimSpace(p.Address)
			if addr == "" || strings.EqualFold(addr, autoAddress) {
				return s.SetPrometheusOverride(nil)
			}
			ns, svc, port, err := engine.ParsePrometheusAddress(addr)
			if err != nil {
				return nil, err
			}
			return s.SetPrometheusOverride(&metrics.Override{Namespace: ns, Service: svc, Port: port, Prefix: strings.TrimSpace(p.Prefix)})
		},
	})
}
