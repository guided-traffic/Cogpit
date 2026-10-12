package integration_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// The charts' ranges are UI state the engine holds for the open drawer: an agent sets one with
// ui.setMetricsRange, sees every chart's range in ui.state, and the drawer's targets carry it (the
// window's range controls follow them).
func TestMetricsRangeCommand(t *testing.T) {
	s, ctx := metricsSession(t)
	var ref model.ObjectRef
	if !testkit.WaitFor(ctx, func() bool {
		tbl := s.Table("pods")
		if tbl == nil || !tbl.Synced {
			return false
		}
		for i := range tbl.Rows {
			if cellOf(tbl, i, "cpu").Text != "" {
				ref = tbl.Rows[i].Ref
				return true
			}
		}
		return false
	}) {
		t.Fatal("no pod with metrics")
	}
	call := func(params string) (any, error) {
		return s.Dispatch(ctx, engine.Invocation{ID: "ui.setMetricsRange", Params: json.RawMessage(params), Origin: engine.OriginExternalAgent})
	}
	refJSON, err := json.Marshal(ref)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := call(`{"ref":` + string(refJSON) + `,"range":"6h"}`); !errors.Is(err, engine.ErrDrawerNotOpen) {
		t.Errorf("with the drawer closed: %v, want ErrDrawerNotOpen", err)
	}

	s.UpdateUI(func(u *engine.UIState) { u.Drawer = &ref })
	var m *engine.ObjectMetrics
	if !testkit.WaitFor(ctx, func() bool {
		m, _ = s.ObjectMetrics(ref)
		return m != nil && len(m.Targets) >= 2
	}) {
		t.Fatalf("no targets: %+v", m)
	}
	for _, tg := range m.Targets {
		if tg.Window != time.Hour || tg.Range != "1h" {
			t.Errorf("chart %q starts at %v (%q), want the default 1h", tg.Name, tg.Window, tg.Range)
		}
	}
	container := m.Targets[1].Name
	res, err := call(`{"ref":` + string(refJSON) + `,"container":"` + container + `","range":"7d"}`)
	if err != nil {
		t.Fatal(err)
	}
	r, ok := res.(commands.MetricsRangeResult)
	if !ok || len(r.Charts) != len(m.Targets) || r.Charts[0].Range != "1h" || r.Charts[1].Container != container || r.Charts[1].Range != "7d" {
		t.Errorf("result %+v", res)
	}
	if m, _ = s.ObjectMetrics(ref); m.Targets[1].Window != 7*24*time.Hour || m.Targets[0].Window != time.Hour {
		t.Errorf("targets after the change: %q %q", m.Targets[0].Range, m.Targets[1].Range)
	}
	st, err := s.Dispatch(ctx, engine.Invocation{ID: "ui.state", Params: json.RawMessage(`{}`), Origin: engine.OriginExternalAgent})
	if err != nil {
		t.Fatal(err)
	}
	if charts := st.(commands.StateResult).DrawerCharts; len(charts) != len(m.Targets) || charts[1].Range != "7d" || charts[0].Container != "" {
		t.Errorf("ui.state's drawer charts = %+v", charts)
	}

	for _, bad := range []string{`"range":"45d"`, `"range":"30s"`, `"range":"soon"`, `"container":"no-such-container","range":"1h"`} {
		if _, err := call(`{"ref":` + string(refJSON) + `,` + bad + `}`); err == nil {
			t.Errorf("%s was accepted", bad)
		}
	}
	if _, err := call(`{"ref":` + string(refJSON) + `,"container":"` + container + `","range":"default"}`); err != nil {
		t.Fatal(err)
	}
	if m, _ = s.ObjectMetrics(ref); m.Targets[1].Range != "1h" {
		t.Errorf("default puts the chart back to 1h, got %q", m.Targets[1].Range)
	}
	if _, err := call(`{"ref":{"kind":"services","namespace":"default","name":"x"}}`); err == nil {
		t.Error("a kind without charts was accepted")
	}
}

func TestChartRangeParseAndFormat(t *testing.T) {
	for in, want := range map[string]time.Duration{
		"": 0, "default": 0, "5m": 5 * time.Minute, "1h": time.Hour, "90m": 90 * time.Minute,
		"7d": 7 * 24 * time.Hour, " 30d ": 30 * 24 * time.Hour,
	} {
		if got, err := engine.ParseChartRange(in); err != nil || got != want {
			t.Errorf("ParseChartRange(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	for _, bad := range []string{"1.5d", "-2d", "0d", "week", "d"} {
		if _, err := engine.ParseChartRange(bad); err == nil {
			t.Errorf("ParseChartRange(%q) accepted", bad)
		}
	}
	for d, want := range map[time.Duration]string{
		0: "", 5 * time.Minute: "5m", time.Hour: "1h", 90 * time.Minute: "90m", 36 * time.Hour: "36h",
		7 * 24 * time.Hour: "7d", 90 * time.Second: "1m30s",
	} {
		if got := engine.FormatChartRange(d); got != want {
			t.Errorf("FormatChartRange(%v) = %q, want %q", d, got, want)
		}
	}
}
