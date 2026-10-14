package engine

import (
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/metrics"
)

// The Metrics source setting: its names, the settings' checks, and the hub following the connected
// cluster's setting at once (reconcile).
func TestMetricsSourceSetting(t *testing.T) {
	for in, want := range map[string]string{"": "", "auto": "", " Prometheus ": "prometheus", "metrics-server": "metrics-server", "NONE": "none"} {
		if got, err := ParseMetricsSource(in); err != nil || got != want {
			t.Errorf("ParseMetricsSource(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if _, err := ParseMetricsSource("thanos"); err == nil {
		t.Error("an unknown source was accepted")
	}
	if (ClusterSettings{MetricsSource: "auto"}).Check() == nil {
		t.Error(`"auto" kept as is: the settings keep Automatic as ""`)
	}
	cs := ClusterSettings{MetricsSource: "bogus"}
	cs.normalize()
	if cs.MetricsSource != "" || !(ClusterSettings{MetricsSource: "none"}).Equal(ClusterSettings{MetricsSource: "none"}) ||
		(ClusterSettings{MetricsSource: "none"}).Equal(ClusterSettings{}) {
		t.Errorf("normalize or Equal: %+v", cs)
	}
	if MetricsSourceName("") != "auto" || MetricsSourceName("none") != "none" {
		t.Error("MetricsSourceName")
	}

	// The connected cluster's hub follows the UI state's setting.
	hub := metrics.NewHub(metrics.Fixed(onePodProvider{}), metrics.Options{ListInterval: time.Hour})
	defer hub.Close()
	s := New(Options{})
	defer s.Close()
	st := &state{s: s}
	b := &Backend{Context: "ctx", Metrics: hub}
	st.followMetricsSource(b, UIState{ClusterSettings: map[string]ClusterSettings{"ctx": {MetricsSource: "none"}}})
	if s := hub.State(metrics.Scope{Kind: metrics.KindPods}); s.Status != metrics.StatusNotServed {
		t.Errorf("state with metrics off: %+v", s)
	}
	if hub.SetSource(metrics.SourceNone) {
		t.Error("the source was not applied")
	}
	st.followMetricsSource(b, UIState{})
	if hub.SetSource(metrics.SourceAuto) {
		t.Error("Automatic was not applied")
	}
	if metricsInUse(nil) != "" || metricsInUse(hub) != "none answering yet" {
		t.Errorf("in use: %q", metricsInUse(hub))
	}
}
