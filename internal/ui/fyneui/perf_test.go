package fyneui

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
)

// TestPerfGateRunsEveryPhase runs the gate measurement offscreen over the real engine and demo
// cluster, with short phases: start-up, scroll, a switch through every view (each shows rows),
// and typing into the YAML editor. It proves the measurement, not the timings.
func TestPerfGateRunsEveryPhase(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the 10,000-pod demo cluster")
	}
	defer func(s, v, y time.Duration, b int) {
		gateScrollFor, gateViewDwell, gateYAMLFor, gateYAMLBytes = s, v, y, b
	}(gateScrollFor, gateViewDwell, gateYAMLFor, gateYAMLBytes)
	gateScrollFor, gateViewDwell, gateYAMLFor, gateYAMLBytes = 300*time.Millisecond, 500*time.Millisecond, 300*time.Millisecond, 200<<10

	s := engine.New(engine.Options{Connector: &engine.ClusterConnector{IncludeDemo: true}})
	defer s.Close()
	if err := commands.RegisterAll(s.Registry()); err != nil {
		t.Fatal(err)
	}
	a := test.NewTempApp(t)
	a.Settings().SetTheme(newTheme(ThemeDark))
	w := test.NewTempWindow(t, nil)
	q := &mainQueue{}
	var got PerfStats
	reported := false
	opts := Options{AutoConnect: engine.DemoContext, PerfMode: PerfGate, PerfReport: func(ps PerfStats) { got, reported = ps, true }}
	sh := newShell(s, a, w, opts, q.do)
	w.SetContent(sh.root)
	w.Resize(fyne.NewSize(1440, 900))
	sh.start()
	sh.startPerf()
	defer func() {
		sh.stop()
		q.drain()
	}()
	deadline := time.Now().Add(60 * time.Second)
	for !reported {
		if time.Now().After(deadline) {
			t.Fatal("the gate measurement did not finish")
		}
		q.drain()
		time.Sleep(2 * time.Millisecond)
	}

	var names []string
	for _, p := range got.Phases {
		names = append(names, p.Name)
	}
	if len(names) != 4 || names[0] != "startup" || names[1] != "scroll" || names[2] != "views" || names[3] != "yaml" {
		t.Fatalf("phases %v", names)
	}
	views := got.Phases[2]
	if len(views.Switches) != 6 {
		t.Fatalf("switches %+v", views.Switches)
	}
	for _, sw := range views.Switches {
		if sw.RowsAfterMs < 0 || sw.Rows == 0 {
			t.Errorf("view %s never showed rows: %+v", sw.View, sw)
		}
	}
	keys := got.Phases[3].Keystrokes
	if keys == nil || keys.Count == 0 || keys.DocBytes < 200<<10 || keys.WorkMaxMs <= 0 {
		t.Errorf("keystrokes %+v", keys)
	}
	if got.Samples == 0 || got.DurationS <= 0 {
		t.Errorf("scroll stats %+v", got)
	}
	t.Logf("switches %+v; keys %+v", views.Switches, *keys)
}
