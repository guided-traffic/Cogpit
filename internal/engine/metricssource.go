package engine

import (
	"fmt"
	"strings"

	"github.com/guided-traffic/cogpit/internal/metrics"
)

// The cluster's Metrics source (G12, G13; ui-parity › Cluster settings › Metrics provider: Lens's
// "Metrics source" dropdown, Automatic, Prometheus, Kubernetes Metrics Server, No metrics, and a
// line naming the source in use, captured). Automatic is both providers, each for its part (the
// owner's requirement); the others turn a provider or both off for the cluster
// (metrics.Hub.SetSource): an off provider is never detected nor asked. Kept per context in the
// cluster's settings (cluster-settings.json, metricsSource), set from the Settings page and by
// agents with cluster.settings.set (a UI command: it writes no cluster object and only narrows
// what Cogpit asks), applied to the connected cluster at once.

// MetricsSourceOption is one choice of the setting.
type MetricsSourceOption struct {
	Name  string // the name commands and the settings file use; "auto" is kept as ""
	Label string // the page's label, Lens's
}

// MetricsSources are the choices in Lens's order.
var MetricsSources = []MetricsSourceOption{
	{"auto", "Automatic"},
	{string(metrics.SourcePrometheus), "Prometheus"},
	{string(metrics.SourceMetricsServer), "Kubernetes Metrics Server"},
	{string(metrics.SourceNone), "No metrics"},
}

// metricsSourceAuto is Automatic's name; the settings keep it as "".
const metricsSourceAuto = "auto"

// ParseMetricsSource reads a source's name ("auto", "prometheus", "metrics-server", "none"; case
// and spaces ignored) as the settings keep it.
func ParseMetricsSource(name string) (string, error) {
	n := strings.ToLower(strings.TrimSpace(name))
	if n == "" || n == metricsSourceAuto {
		return "", nil
	}
	if err := CheckMetricsSource(n); err != nil {
		return "", err
	}
	return n, nil
}

// CheckMetricsSource refuses a kept source that is not one of MetricsSources ("" is Automatic).
func CheckMetricsSource(s string) error {
	if s == "" {
		return nil
	}
	names := make([]string, 0, len(MetricsSources))
	for _, o := range MetricsSources {
		if o.Name == s && s != metricsSourceAuto {
			return nil
		}
		names = append(names, o.Name)
	}
	return fmt.Errorf("metrics source %q: one of %s", s, strings.Join(names, ", "))
}

// MetricsSourceName is a kept source's name: "auto" for "".
func MetricsSourceName(s string) string {
	if s == "" {
		return metricsSourceAuto
	}
	return s
}

// metricsSourceOf is a context's source from the UI state.
func metricsSourceOf(u *UIState, context string) metrics.Source {
	if u == nil || context == "" {
		return metrics.SourceAuto
	}
	return sourceOfIntegrationsOff(integrationsOffOf(u, context)) // the two metrics integrations (integrations.go)
}

// MetricsSourceOf names a context's Metrics source as the settings page shows it: "auto",
// "prometheus", "metrics-server" or "none", made by its two metrics integrations.
func (u *UIState) MetricsSourceOf(context string) string {
	return MetricsSourceName(string(metricsSourceOf(u, context)))
}

// SetMetricsSource sets the two metrics integrations of a context's settings as a Metrics source
// does ("" Automatic: both on), starting from what applies now (its choice, else defaultsOff).
func (c *ClusterSettings) SetMetricsSource(defaultsOff IntegrationSet, source string) {
	off := integrationsOffOfSource(source)
	c.SetIntegrations(defaultsOff, IntMetrics&^off, off)
}

// followMetricsSource applies the connected cluster's source to its hub; a change is announced, so
// that open drawers and the Overview read the new state.
func (st *state) followMetricsSource(b *Backend, now UIState) {
	if b == nil || b.Metrics == nil {
		return
	}
	if b.Metrics.SetSource(metricsSourceOf(&now, b.Context)) {
		st.s.goSafe("metrics source", func() { st.s.emit(Event{Type: EventMetrics}) })
	}
}

// metricsInUse names the providers that answer now, the line under the setting.
func metricsInUse(hub *metrics.Hub) string {
	if hub == nil {
		return ""
	}
	var labels []string
	for _, p := range hub.Providers() {
		if len(p.Serves) > 0 && p.Status == metrics.StatusAvailable {
			labels = append(labels, p.Label)
		}
	}
	if len(labels) == 0 {
		return "none answering yet"
	}
	return strings.Join(labels, " and ")
}
