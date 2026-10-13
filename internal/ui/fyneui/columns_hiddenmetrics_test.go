package fyneui

import (
	"slices"
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
)

// A column hidden from the ⋮ list while the cluster's settings hide the Pods' CPU sends the view's
// own hidden columns only: the CPU column, which the cluster's settings hide, does not become one of
// the view's, so it comes back when the metrics are shown again (the column list read every
// hidden column off the snapshot, which hides the metrics' columns too).
func TestColumnsMenuLeavesHiddenMetricsToTheClusterSettings(t *testing.T) {
	s, sh, _, until := demoSettingsShell(t)
	mustDispatch(t, s, "cluster.settings.set", map[string]any{"context": engine.DemoContext, "hiddenMetrics": []string{"cpu"}})
	until("the CPU column hidden", func() bool { return sh.view.snap != nil && !shownColumn(sh, model.ColumnCPU) })

	sh.view.openColumns()
	test.Tap(columnCheck(t, sh, "Namespace"))
	until("the Namespace column hidden", func() bool { return sh.view.snap != nil && !shownColumn(sh, "namespace") })
	if got := s.UIState().HiddenColumns["pods"]; !slices.Equal(got, []string{"namespace"}) {
		t.Errorf("the Pods view's own hidden columns are %q, want [namespace]", got)
	}

	mustDispatch(t, s, "cluster.settings.set", map[string]any{"context": engine.DemoContext, "hiddenMetrics": []string{}})
	until("the CPU column back", func() bool { return sh.view.snap != nil && shownColumn(sh, model.ColumnCPU) })
	if shownColumn(sh, "namespace") {
		t.Error("the Namespace column the user hid is back")
	}
}
