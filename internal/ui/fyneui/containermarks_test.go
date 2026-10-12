package fyneui

import (
	"strings"
	"testing"
)

// A pod's Containers cell draws one square per container in the container's status colour, the
// text ("1/2") gives way to them, and the tooltip names each container and its state (owner,
// 2026-10-12, from Lens).
func TestContainersCellDrawsSquares(t *testing.T) {
	f := kindsFake(t, "pods")
	sh, _, q := newTestShell(t, f, Options{Theme: ThemeDark})
	q.until(t, "the pods", func() bool { return sh.view.snap != nil && len(sh.view.snap.Rows) > 0 })
	settle(sh.win)
	col := columnOf(t, sh, "containers")
	found := false
	for row, r := range sh.view.snap.Rows {
		ci := sh.view.cols[col]
		if ci >= len(r.Cells) || r.Cells[ci].Marks == nil {
			continue
		}
		marks := *r.Cells[ci].Marks
		c := cellAt(t, sh, row, col)
		if c.text.Text != "" || c.shown != len(marks) {
			t.Fatalf("row %d: text %q, %d of %d squares shown", row, c.text.Text, c.shown, len(marks))
		}
		th := c.Theme()
		for i, m := range marks {
			if !c.marks[i].Visible() || c.marks[i].FillColor != themeColor(th, markColorName(m.Status)) {
				t.Errorf("row %d square %d: visible %v, colour %v for %s", row, i, c.marks[i].Visible(), c.marks[i].FillColor, m.Status)
			}
			if !strings.Contains(c.tipText(), m.Text) {
				t.Errorf("row %d: tooltip %q lacks %q", row, c.tipText(), m.Text)
			}
		}
		found = true
		break
	}
	if !found {
		t.Fatal("no pod row with container marks in the fixture")
	}
}
