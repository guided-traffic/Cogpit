package kinds_test

import (
	"testing"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

func FuzzLimitRanges(f *testing.F) { fuzzProfile(f, "limitranges") }

func TestLimitRangesStatusTexts(t *testing.T) {
	const none = model.StatusNone
	// A limit range has no state: no row is coloured (inferred).
	checkStatusTexts(t, []statusText{
		{"limitranges", "healthy", "name", "defaults", none, none},
		{"limitranges", "failing", "namespace", "batch", none, none},
		{"limitranges", "minimal", "age", "1h", none, none},
		{"limitranges", "terminating", "name", "storage-bounds", none, model.StatusTerminating},
	})
}

// TestLimitRangeTable pins the drawer's Limits table: a row per type and resource, the bounds in
// their columns.
func TestLimitRangeTable(t *testing.T) {
	p, _ := kinds.ByKey("limitranges")
	for _, c := range loadCases(t, "limitranges") {
		if c.name != "healthy" {
			continue
		}
		sections := p.Detail(c.obj)
		if len(sections) != 1 || sections[0].Table == nil {
			t.Fatalf("sections: %+v", sections)
		}
		tbl := sections[0].Table
		want := [][]string{
			{"Container", "cpu", "10m", "2", "100m", "500m", "10"},
			{"Container", "memory", "16Mi", "2Gi", "128Mi", "512Mi", ""},
			{"Pod", "cpu", "", "4", "", "", ""},
			{"Pod", "memory", "", "4Gi", "", "", ""},
			{"PersistentVolumeClaim", "storage", "1Gi", "100Gi", "", "", ""},
		}
		if len(tbl.Rows) != len(want) {
			t.Fatalf("rows: %q", tbl.Rows)
		}
		for i, row := range want {
			for j, cell := range row {
				if tbl.Rows[i][j] != cell {
					t.Errorf("row %d column %s = %q, want %q", i, tbl.Columns[j], tbl.Rows[i][j], cell)
				}
			}
		}
	}
}
