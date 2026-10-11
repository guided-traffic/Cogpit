package fyneui

import (
	"fmt"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
)

// churned returns a copy of snap in which n rows changed, as watch events change them: another
// status, more restarts. Rows are copied, not mutated: snapshots are immutable.
func churned(snap *engine.TableSnapshot, n, round int) *engine.TableSnapshot {
	c := *snap
	c.Rows = append([]model.Row(nil), snap.Rows...)
	c.Revision = snap.Revision + uint64(round) + 1
	for i := range n {
		r := (i*97 + round*31) % len(c.Rows)
		row := c.Rows[r]
		row.Cells = append([]model.Cell(nil), row.Cells...)
		row.Cells[3].Num += int64(round + 1)
		if (i+round)%2 == 0 {
			row.Status = model.StatusPending
			row.Cells[8] = model.Cell{Text: "Pending", Status: model.StatusPending}
		} else {
			row.Status = model.StatusOK
			row.Cells[8] = model.Cell{Text: "Running", Status: model.StatusOK}
		}
		c.Rows[r] = row
	}
	return &c
}

// BenchmarkTableRefresh10k measures replacing the 10,000-row snapshot of the open view: the table
// refresh (visible cells only) and, in the paint variant, a full offscreen render of the
// 1440×900 window with Fyne's software painter (slower than the GPU path; an upper bound).
func BenchmarkTableRefresh10k(b *testing.B) {
	for _, paint := range []bool{false, true} {
		b.Run(fmt.Sprintf("paint=%v", paint), func(b *testing.B) {
			f := connectedFake(b, 10000)
			sh, w, q := newTestShell(b, f, Options{})
			q.until(b, "kinds", func() bool { return len(sh.kinds) > 0 })
			settle(w)
			snaps := [2]*engine.TableSnapshot{f.tables["pods"], churned(f.tables["pods"], 100, 1)}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				sh.view.setSnapshot(snaps[i%2])
				if paint {
					w.Canvas().Capture()
				}
			}
		})
	}
}

// BenchmarkSnapshotEvery100ms measures what one engine snapshot costs the main goroutine on the
// real event path (EventTable → coalescing → sh.do → Table → refresh of the visible cells). The
// engine builds at most one snapshot per view every 100 ms; each here carries 100 changed rows,
// what 1,000 watch events per second change in 100 ms. "%budget" is the share of the 100 ms
// interval the main goroutine spends on it. Painting is left out: the software painter here
// rasterises on the CPU (BenchmarkTableRefresh10k/paint=true), the desktop app paints with GL.
func BenchmarkSnapshotEvery100ms(b *testing.B) {
	f := connectedFake(b, 10000)
	sh, w, q := newTestShell(b, f, Options{})
	q.until(b, "kinds", func() bool { return len(sh.kinds) > 0 })
	settle(w)
	ring := make([]*engine.TableSnapshot, 10)
	for i := range ring {
		ring[i] = churned(f.tables["pods"], 100, i)
	}
	b.ReportAllocs()
	b.ResetTimer()
	start := time.Now()
	for i := 0; i < b.N; i++ {
		f.setTable("pods", ring[i%len(ring)])
		q.drain()
	}
	per := time.Since(start) / time.Duration(b.N)
	b.ReportMetric(float64(per)/float64(100*time.Millisecond)*100, "%budget")
}
