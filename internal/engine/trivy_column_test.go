package engine

import (
	"slices"
	"testing"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// ratingCells returns the Pods view's Image Rating column: its index, and each pod's cell by name.
func ratingCells(snap *TableSnapshot) (int, map[string]model.Cell) {
	ci := slices.IndexFunc(snap.Columns, func(c model.Column) bool { return c.ID == trivyColumnID })
	out := map[string]model.Cell{}
	for _, r := range snap.Rows {
		if ci >= 0 && ci < len(r.Cells) {
			out[r.Ref.Name] = r.Cells[ci]
		}
	}
	return ci, out
}

// The Pods list's Image Rating (D-78's second phase): a pod's worst image rating with its count, in
// the rating's colour, sorted by the rating, then the count; empty for a pod without a report; the
// column gone while the integration is off.
func TestTrivyPodsColumn(t *testing.T) {
	src := trivyFixture()
	// Another pod of the same rating with fewer findings: sorted after the one with more.
	src.Set(testkit.PodsGVR, trivyPod("team", "lone", nil, "app=nginx:1.16.1"))
	src.Set(vulnReportsGVR, vulnReport("team", "Pod", "lone", "app", "library/nginx", "1.16.1", 2, 5))
	s := newCRDSession(t, src)
	b := s.Backend()
	waitFor(t, "the reports listed", func() bool { return trivyOn(b).synced() })
	s.UpdateUI(func(u *UIState) { u.View, u.Sort = "pods", SortSpec{Column: trivyColumnID, Desc: true} })
	snap := waitTable(t, s, "pods", func(ts *TableSnapshot) bool {
		_, cells := ratingCells(ts)
		return ts.Synced && cells["web-5d9c-a"].Text != "" && ts.Sort.Column == trivyColumnID
	})
	ci, cells := ratingCells(snap)
	if col := snap.Columns[ci]; col.Title != "Image Rating" || col.Type != model.ColumnRank {
		t.Errorf("column %+v", col)
	}
	for name, want := range map[string]model.Cell{
		"web-5d9c-a":         {Text: "Critical · 39", Status: model.StatusError},
		"lone":               {Text: "Critical · 2", Status: model.StatusError},
		"nightly-29000000-x": {Text: "Critical · 1", Status: model.StatusError},
		"bare":               {Text: "Critical · 1", Status: model.StatusError},
		"once-y":             {Text: "Clean", Status: model.StatusOK},
		"db-0":               {},
	} {
		if got := cells[name]; got.Text != want.Text || got.Status != want.Status {
			t.Errorf("%s: %+v, want %q %q", name, got, want.Text, want.Status)
		}
	}
	var order []string
	for _, r := range snap.Rows {
		order = append(order, r.Ref.Name)
	}
	if !slices.Equal(order[:2], []string{"web-5d9c-a", "lone"}) || order[len(order)-2] != "once-y" || order[len(order)-1] != "db-0" {
		t.Errorf("sorted by the rating, descending, the pod without one last: %v", order)
	}

	// A new report reaches the list.
	src.Set(testkit.PodsGVR, trivyPod("shop", "db-0", nil, "db=postgres:16"))
	src.Set(vulnReportsGVR, vulnReport("shop", "Pod", "db-0", "db", "library/postgres", "16", 0, 0, 0, 3))
	waitTable(t, s, "pods", func(ts *TableSnapshot) bool {
		_, cells := ratingCells(ts)
		return cells["db-0"].Text == "Low · 3" && cells["db-0"].Status == model.StatusSucceeded
	})

	// Off: the column is gone.
	setTrivy(s, false)
	waitTable(t, s, "pods", func(ts *TableSnapshot) bool { ci, _ := ratingCells(ts); return ts.Synced && ci < 0 })
	if cols := s.ViewColumns(testPods); slices.ContainsFunc(cols, func(c model.Column) bool { return c.ID == trivyColumnID }) {
		t.Error("off, the view still has the column")
	}
}

// The integration reads the reports in a trimmed scope of its own (cache.Scope.Trim): its objects
// hold no vulnerability list, while the reports' own view reads them whole.
func TestTrivyReadsTrimmedReports(t *testing.T) {
	src := trivyFixture()
	whole := vulnReport("team", "Pod", "bare", "shell", "library/alpine", "3.12", 1)
	whole.Object["report"].(map[string]any)["vulnerabilities"] = []any{map[string]any{
		"vulnerabilityID": "CVE-2022-37434", "severity": "CRITICAL", "title": "zlib: heap-based buffer over-read",
	}}
	src.Set(vulnReportsGVR, whole)
	s := newCRDSession(t, src)
	b := s.Backend()
	waitFor(t, "the reports listed", func() bool { return trivyOn(b).synced() })
	tr := trivyOn(b)
	if sc := tr.h.Scope(); sc.Trim != trivyTrim {
		t.Fatalf("scope %+v", sc)
	}
	o, ok := tr.h.Get("team", "pod-bare-shell")
	if !ok {
		t.Fatal("no trimmed report")
	}
	if _, has := o.Object["report"].(map[string]any)["vulnerabilities"]; has {
		t.Error("the trimmed report keeps its vulnerabilities")
	}
	if r, ok := model.ParseVulnerabilityReport(o); !ok || r.Counts.Critical != 1 || r.Tag != "3.12" {
		t.Errorf("the trimmed report's summary: %+v", r)
	}
	p, _ := b.Profile(trivyReportsKind)
	h := b.Acquire(p)
	defer h.Release()
	waitFor(t, "the reports' own scope", h.HasSynced)
	o, _ = h.Get("team", "pod-bare-shell")
	if v := model.NestedSlice(o.Object, "report", "vulnerabilities"); len(v) != 1 {
		t.Errorf("the reports' own view lost the vulnerabilities: %v", v)
	}
}

func TestTrivyWorstImage(t *testing.T) {
	r := func(counts ...int64) *model.VulnerabilityReport {
		counts = append(counts, 0, 0, 0, 0, 0)
		return &model.VulnerabilityReport{Counts: model.VulnerabilityCounts{Critical: counts[0], High: counts[1], Medium: counts[2], Low: counts[3], Unknown: counts[4]}}
	}
	images := []podImage{{image: "a", report: r(0, 120)}, {image: "b"}, {image: "c", report: r(3)}, {image: "d", report: r(39)}}
	if w, ok := worstImage(images); !ok || w.image != "d" {
		t.Errorf("worst %+v", w)
	}
	if _, ok := worstImage([]podImage{{image: "b"}}); ok {
		t.Error("an image without a report is no rating")
	}
}
