package engine

import "testing"

// The bundle installs Prometheus where detection looks for Cogpit's own first (two packages of one
// layer name it: internal/helm and internal/metrics).
func TestMetricsStackNames(t *testing.T) {
	if err := MetricsStackNames(); err != nil {
		t.Fatal(err)
	}
	ns := MetricsNamespaceObject()
	if !CogpitsMetricsNamespace(ns) || ns.GetName() != "cogpit-metrics" {
		t.Errorf("the namespace Cogpit creates: %v", ns.Object)
	}
	ns.SetLabels(map[string]string{labelManagedBy: "someone-else", labelPartOf: metricsStackPartOf})
	if CogpitsMetricsNamespace(ns) {
		t.Error("a namespace another tool manages counts as Cogpit's")
	}
}
