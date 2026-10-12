package metrics

import (
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestQuoteLabel(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"web", `"web"`},
		{`a"b`, `"a\"b"`},
		{`a\b`, `"a\\b"`},
		{"a\nb", `"a\nb"`},
		{"a\x00b", `"a\x00b"`},
		{`x"} or vector(1) #`, `"x\"} or vector(1) #"`},
		{"ünï", `"ünï"`},
	} {
		if got := quoteLabel(tc.in); got != tc.want {
			t.Errorf("quoteLabel(%q) = %s, want %s", tc.in, got, tc.want)
		}
	}
}

// Whatever a name holds, the literal reads back as exactly that name: it cannot end early.
func FuzzQuoteLabel(f *testing.F) {
	for _, s := range []string{"", "web", `"`, `\`, `\"`, "\n", `a"} or up{x="`, "\x00\x7f"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if !utf8.ValidString(s) {
			t.Skip()
		}
		q := quoteLabel(s)
		got, err := strconv.Unquote(q)
		if err != nil || got != s {
			t.Fatalf("quoteLabel(%q) = %s reads back as %q (%v)", s, q, got, err)
		}
	})
}

func TestQueries(t *testing.T) {
	pod := podPlan("web", "a", rangeRate(30*time.Second))
	for name, tc := range map[string]struct{ got, want string }{
		"cpu":    {pod.cpu, `sum by (container) (rate(container_cpu_usage_seconds_total{namespace="web",pod="a",container!="",container!="POD"}[4m]))`},
		"mem":    {pod.mem, `sum by (container) (container_memory_working_set_bytes{namespace="web",pod="a",container!="",container!="POD"})`},
		"rx":     {pod.rx, `sum(rate(container_network_receive_bytes_total{namespace="web",pod="a"}[4m]))`},
		"tx":     {pod.tx, `sum(rate(container_network_transmit_bytes_total{namespace="web",pod="a"}[4m]))`},
		"fs":     {pod.fsUsage, `sum by (container) (container_fs_usage_bytes{namespace="web",pod="a",container!="",container!="POD"})`},
		"fsr":    {pod.fsReads, `sum by (container) (rate(container_fs_reads_bytes_total{namespace="web",pod="a",container!="",container!="POD"}[4m]))`},
		"list":   {podListRate(metricCPU, ""), `sum by (namespace, pod, container) (rate(container_cpu_usage_seconds_total{container!="",container!="POD"}[4m]))`},
		"week":   {podPlan("web", "a", rangeRate(605*time.Second)).rx, `sum(rate(container_network_receive_bytes_total{namespace="web",pod="a"}[665s]))`},
		"listns": {podListGauge(metricMemory, "web"), `sum by (namespace, pod, container) (container_memory_working_set_bytes{namespace="web",container!="",container!="POD"})`},
	} {
		if tc.got != tc.want {
			t.Errorf("%s:\n got %s\nwant %s", name, tc.got, tc.want)
		}
	}

	evil := podPlan(`ns"} or up{a="`, `p\"`, rateWindow)
	if !strings.Contains(evil.cpu, `namespace="ns\"} or up{a=\""`) || !strings.Contains(evil.cpu, `pod="p\\\""`) {
		t.Errorf("names were not quoted: %s", evil.cpu)
	}

	node := nodePlan("n1.example", rangeRate(30*time.Second))
	want := `sum(rate(container_cpu_usage_seconds_total{id="/",node="n1.example"}[4m])) or ` +
		`sum(rate(container_cpu_usage_seconds_total{id="/",instance=~"n1\\.example(:[0-9]+)?"}[4m]))`
	if node.cpu != want {
		t.Errorf("node cpu:\n got %s\nwant %s", node.cpu, want)
	}
}

func TestParseResult(t *testing.T) {
	t.Run("matrix", func(t *testing.T) {
		got, err := parseResult([]byte(`{"status":"success","data":{"resultType":"matrix","result":[
			{"metric":{"container":"app"},"values":[[1000,"0.5"],[1030.5,"NaN"],[1060,"+Inf"],[1090,"1.25"]]}]}}`))
		if err != nil || len(got) != 1 {
			t.Fatalf("%v %v", got, err)
		}
		pts := got[0].points
		if len(pts) != 2 || pts[0] != (point{1000000, 0.5}) || pts[1] != (point{1090000, 1.25}) {
			t.Errorf("points = %+v (NaN and Inf must be dropped)", pts)
		}
	})
	t.Run("vector", func(t *testing.T) {
		got, err := parseResult([]byte(`{"status":"success","data":{"resultType":"vector","result":[{"metric":{"pod":"a"},"value":[1000.25,"3"]}]}}`))
		if err != nil || len(got) != 1 || got[0].points[0] != (point{1000250, 3}) {
			t.Errorf("%+v %v", got, err)
		}
	})
	t.Run("empty", func(t *testing.T) {
		got, err := parseResult([]byte(`{"status":"success","data":{"resultType":"matrix","result":[]}}`))
		if err != nil || len(got) != 0 {
			t.Errorf("%+v %v", got, err)
		}
	})
	for name, body := range map[string]string{
		"error status": `{"status":"error","errorType":"bad_data","error":"parse error"}`,
		"not json":     `<html>`,
		"bad sample":   `{"status":"success","data":{"resultType":"matrix","result":[{"metric":{},"values":[[1,"x"]]}]}}`,
		"odd type":     `{"status":"success","data":{"resultType":"streams","result":[]}}`,
	} {
		if _, err := parseResult([]byte(body)); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestRangeStep(t *testing.T) {
	h := NewHub(nil, Options{})
	defer h.Close()
	st := &scopeState{subs: map[*Subscription]struct{}{}}
	window, step := h.rangeOfLocked(st)
	if window.Hours() != 1 || step.Seconds() != 30 {
		t.Errorf("default window %s step %s, want 1h at 30s (window/120)", window, step)
	}
	st.subs[&Subscription{windows: []time.Duration{5 * time.Minute, 24 * time.Hour}}] = struct{}{}
	window, step = h.rangeOfLocked(st)
	// 24h across 1000 points is 87.2s, rounded up to whole seconds; 5m/120 and 1h/120 are finer.
	if window.Hours() != 24 || step != 87*time.Second {
		t.Errorf("window %s step %s", window, step)
	}
	if float64(window/step) > 1000 {
		t.Errorf("%d points exceed the bound", window/step)
	}
}
