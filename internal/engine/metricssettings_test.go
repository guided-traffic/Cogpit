package engine

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/guided-traffic/cogpit/internal/metrics"
)

// The Prometheus override is kept per context in Cogpit's own preferences, owner-only, and read
// back by the next start; what is not a service address is refused before it is kept.
func TestMetricsSettingsKeepTheOverride(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Cogpit")
	m, err := NewMetricsSettings(dir)
	if err != nil {
		t.Fatal(err)
	}
	if o := m.Override("kind-cogpit"); o != nil {
		t.Fatalf("a new store has %+v", o)
	}
	want := metrics.Override{Namespace: "observability", Service: "thanos-query", Port: "9090", Prefix: "/prometheus"}
	if err := m.SetOverride("kind-cogpit", &want); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(filepath.Join(dir, MetricsSettingsFile))
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && fi.Mode().Perm() != 0o600 {
		t.Errorf("metrics.json is %v, want owner-only", fi.Mode().Perm())
	}
	again, err := NewMetricsSettings(dir)
	if err != nil {
		t.Fatal(err)
	}
	if o := again.Override("kind-cogpit"); o == nil || *o != want {
		t.Errorf("read back %+v, want %+v", o, want)
	}
	if o := again.Override("other"); o != nil {
		t.Errorf("another context has %+v", o)
	}
	if err := again.SetOverride("kind-cogpit", nil); err != nil {
		t.Fatal(err)
	}
	if third, _ := NewMetricsSettings(dir); third.Override("kind-cogpit") != nil {
		t.Error("a forgotten override came back")
	}
	// Without a store an override lasts for the connection: nothing is kept, nothing fails.
	var none *MetricsSettings
	if err := none.SetOverride("x", &want); err != nil || none.Override("x") != nil {
		t.Errorf("a nil store: %v, %+v", err, none.Override("x"))
	}
}

// A file someone edited is read for what is valid in it; an entry that is not a service address
// is left out, so it can never reach the service proxy's path.
func TestMetricsSettingsReadOnlyValidEntries(t *testing.T) {
	dir := t.TempDir()
	body := `{"prometheus":{"a":{"namespace":"monitoring","service":"prometheus","port":"9090"},` +
		`"b":{"namespace":"monitoring","service":"prometheus","port":"9090","prefix":"/../../api/v1/secrets"}}}`
	if err := os.WriteFile(filepath.Join(dir, MetricsSettingsFile), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	m, err := NewMetricsSettings(dir)
	if err != nil {
		t.Fatal(err)
	}
	if m.Override("a") == nil || m.Override("b") != nil {
		t.Errorf("a = %+v, b = %+v; want a kept and b dropped", m.Override("a"), m.Override("b"))
	}
}

func TestCheckPrometheusOverride(t *testing.T) {
	ok := metrics.Override{Namespace: "monitoring", Service: "prometheus-server", Port: "http"}
	for _, tc := range []struct {
		name string
		o    func(o *metrics.Override)
		want string
	}{
		{"valid", func(*metrics.Override) {}, ""},
		{"a port number", func(o *metrics.Override) { o.Port = "9090" }, ""},
		{"a prefix", func(o *metrics.Override) { o.Prefix = "/prom/v1" }, ""},
		{"a namespace with a slash", func(o *metrics.Override) { o.Namespace = "a/b" }, "namespace"},
		{"an empty service", func(o *metrics.Override) { o.Service = "" }, "service"},
		{"port zero", func(o *metrics.Override) { o.Port = "0" }, "port"},
		{"a port too large", func(o *metrics.Override) { o.Port = "70000" }, "port"},
		{"a port name with a space", func(o *metrics.Override) { o.Port = "we b" }, "port"},
		{"a prefix that climbs", func(o *metrics.Override) { o.Prefix = "/../api" }, ".."},
		{"a prefix with a query", func(o *metrics.Override) { o.Prefix = "/x?y=1" }, "prefix"},
		{"a prefix with a dot segment", func(o *metrics.Override) { o.Prefix = "/a/./b" }, ".."},
	} {
		o := ok
		tc.o(&o)
		err := CheckPrometheusOverride(o)
		if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
			t.Errorf("%s: %v, want %q", tc.name, err, tc.want)
		}
	}
	for _, tc := range []struct{ in, ns, svc, port string }{
		{"monitoring/prometheus-server:80", "monitoring", "prometheus-server", "80"},
		{" observability/thanos-query:http ", "observability", "thanos-query", "http"},
	} {
		ns, svc, port, err := ParsePrometheusAddress(tc.in)
		if err != nil || ns != tc.ns || svc != tc.svc || port != tc.port {
			t.Errorf("%q = %q %q %q (%v)", tc.in, ns, svc, port, err)
		}
	}
	for _, bad := range []string{"prometheus", "monitoring/prometheus", "monitoring/prometheus:"} {
		if _, _, _, err := ParsePrometheusAddress(bad); err == nil {
			t.Errorf("%q was read as an address", bad)
		}
	}
}
