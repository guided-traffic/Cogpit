package fyneui

import (
	"testing"
)

// The tests of the table's header and keyboard features (ui-parity › Tables).

// TestNamespaceFilterHiddenOnClusterScopedViews: Lens shows no namespace filter on a
// cluster-scoped view; a namespaced view shows it again.
func TestNamespaceFilterHiddenOnClusterScopedViews(t *testing.T) {
	f := kindsFake(t, "nodes")
	sh, _, q := newTestShell(t, f, Options{})
	q.until(t, "the nodes view", func() bool { return sh.view.snap != nil && sh.view.snap.View == "nodes" })
	if sh.view.nsBox.Visible() {
		t.Error("the namespace filter shows on Nodes")
	}
	sh.dispatch("ui.openView", openViewParams{View: "pods"}, nil)
	q.until(t, "the pods view", func() bool { return sh.view.snap != nil && sh.view.snap.View == "pods" })
	if !sh.view.nsBox.Visible() {
		t.Error("the namespace filter is hidden on Pods")
	}
}
