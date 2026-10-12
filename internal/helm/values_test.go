package helm

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	chart "helm.sh/helm/v4/pkg/chart/v2"
	"helm.sh/helm/v4/pkg/release/common"
	rspb "helm.sh/helm/v4/pkg/release/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// A release's values from its Secret: the user-supplied ones, or every value with the chart's
// defaults coalesced under them, as helm get values [--all] prints them; an unreadable Secret is
// an error that quotes nothing of it.
func TestReleaseValues(t *testing.T) {
	rls := &rspb.Release{
		Name: "web", Namespace: "apps", Version: 4,
		Info: &rspb.Info{Status: common.StatusDeployed, FirstDeployed: t0, LastDeployed: t0},
		Chart: &chart.Chart{
			Metadata: &chart.Metadata{Name: "web", Version: "1.4.0", APIVersion: "v2"},
			Values:   map[string]any{"replicas": 1, "image": map[string]any{"tag": "1.0", "pull": "Always"}},
		},
		Config: map[string]any{"image": map[string]any{"tag": "2.0"}, "password": "hunter2"},
	}
	secret := &unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{"name": "sh.helm.release.v1.web.v4", "namespace": "apps"},
		"data":     map[string]any{"release": base64.StdEncoding.EncodeToString([]byte(encode(t, rls, true)))},
	}}
	user, err := ReleaseValues(secret, false)
	if err != nil || user.YAML != "image:\n  tag: \"2.0\"\npassword: hunter2\n" || user.All || user.Revision != 4 || user.Chart != "web" || user.ChartVersion != "1.4.0" {
		t.Errorf("user-supplied %+v, %v", user, err)
	}
	all, err := ReleaseValues(secret, true)
	if err != nil || !all.All || all.YAML != "image:\n  pull: Always\n  tag: \"2.0\"\npassword: hunter2\nreplicas: 1\n" {
		t.Errorf("all %+v, %v", all, err)
	}
	broken := secret.DeepCopy()
	broken.Object["data"] = map[string]any{"release": base64.StdEncoding.EncodeToString([]byte("not a release: hunter2"))}
	if _, err := ReleaseValues(broken, false); err == nil || strings.Contains(err.Error(), "hunter2") {
		t.Errorf("an unreadable Secret: %v", err)
	}
	empty := rls
	empty.Config = nil
	secret.Object["data"] = map[string]any{"release": base64.StdEncoding.EncodeToString([]byte(encode(t, empty, true)))}
	if v, err := ReleaseValues(secret, false); err != nil || v.YAML != "" {
		t.Errorf("no user-supplied values: %+v, %v", v, err)
	}
}

// Values parse as helm reads a values file; a parse error names the line, never a value.
func TestParseValues(t *testing.T) {
	v, err := ParseValues("a: 1\nb:\n  c: hunter2\n")
	if err != nil || v["a"] != float64(1) || v["b"].(map[string]any)["c"] != "hunter2" {
		t.Errorf("values %v, %v", v, err)
	}
	if v, err := ParseValues("  \n"); err != nil || len(v) != 0 {
		t.Errorf("empty values %v, %v", v, err)
	}
	for _, bad := range []string{"a: 1\nb: [hunter2\n", "- hunter2\n", "{hunter2: [1}: x\n", "? [hunter2]\n: 1\n"} {
		_, err := ParseValues(bad)
		if err == nil || strings.Contains(err.Error(), "hunter2") {
			t.Errorf("%q: %v", bad, err)
		}
	}
	if _, err := ParseValues("a: 1\nb: [x\n"); err == nil || !strings.Contains(err.Error(), "line") {
		t.Errorf("no line: %v", err)
	}
}

// Scrubbing replaces every string and number of the values that is long enough, longest first, in
// plain and base64 form, and cuts long texts.
func TestScrubValues(t *testing.T) {
	vals := map[string]any{
		"password": "hunter2", "nested": []any{map[string]any{"token": "tok-abc-123"}}, "pin": float64(98765),
		"short": "abc", "multi": "line one\nsecond line", "inner": "hunter2-extended",
	}
	text := `rejected "hunter2-extended" and hunter2, tok-abc-123, 98765, abc, second line, ` + base64.StdEncoding.EncodeToString([]byte("hunter2"))
	got := ScrubValues(text, vals)
	for _, leak := range []string{"hunter2", "tok-abc-123", "98765", "second line", base64.StdEncoding.EncodeToString([]byte("hunter2"))} {
		if strings.Contains(got, leak) {
			t.Errorf("%q in %q", leak, got)
		}
	}
	if !strings.Contains(got, "abc,") || strings.Count(got, withheldValue) != 6 {
		t.Errorf("scrubbed %q", got)
	}
	if long := ScrubValues(strings.Repeat("x", 3*maxErrorText), nil); len(long) > maxErrorText+20 {
		t.Errorf("%d bytes not cut", len(long))
	}
}

// A value with quotes, backslashes or other characters a quoting escapes is scrubbed in the forms
// error texts quote it in: Go's %q, JSON (an admission webhook's message), ASCII-only quoting;
// and a number in the form a template renders it (1e+06), not only the plain digits (review of
// the fourth run, R-4).
func TestScrubValuesEscapedForms(t *testing.T) {
	pw := "p\"4ss\\w0rd\tnäme<x>"
	vals := map[string]any{"pw": pw, "big": float64(1000000), "huge": float64(12345678901234567890)}
	for _, text := range []string{
		fmt.Sprintf("denied: %q", pw),
		fmt.Sprintf("denied: %+q", pw),
		`admission webhook denied: {"pw":"p\"4ss\\w0rd\tnäme<x>"}`,
		"replicas 1e+06 too high, id 1.2345678901234567e+19",
	} {
		got := ScrubValues(text, vals)
		for _, leak := range []string{"4ss", "w0rd", "1e+06", "1.2345678901234567e+19"} {
			if strings.Contains(got, leak) {
				t.Errorf("%q left in %q", leak, got)
			}
		}
	}
}

// The SDK's log lines during an upgrade pass a handler that scrubs the values from the message
// and from every attribute, groups included.
func TestScrubHandler(t *testing.T) {
	var buf bytes.Buffer
	vals := map[string]any{"password": "hunter2"}
	log := slog.New(scrubHandler{next: slog.NewTextHandler(&buf, nil), values: vals})
	log.With("pre", "hunter2").WithGroup("g").Warn("upgrade failed: hunter2", "error", errorString("bad hunter2"), slog.Group("in", "v", "hunter2"))
	if strings.Contains(buf.String(), "hunter2") || !strings.Contains(buf.String(), "upgrade failed") {
		t.Errorf("logged %s", buf.String())
	}
}

// The Helm SDK's coalescing warnings, which it writes with the standard library's log package,
// reach Cogpit's log without the value they quote; other lines pass unchanged.
func TestRedactSDKLog(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(RedactSDKLog(slog.NewTextHandler(&buf, nil)))
	log.Info("warning: destination for auth.password is a table. Ignoring non-table value (hunter2 (x))")
	log.Info("warning: cannot overwrite table with non table for web.tls (map[cert:hunter2\nkey:y])")
	log.Info("warning: skipped value for web.x: Not a table.", "k", "v")
	out := buf.String()
	if strings.Contains(out, "hunter2") || !strings.Contains(out, "destination for auth.password is a table. Ignoring non-table value ("+withheldValue+")") ||
		!strings.Contains(out, "non table for web.tls ("+withheldValue+")") || !strings.Contains(out, "skipped value for web.x: Not a table.") {
		t.Errorf("logged %s", out)
	}
}

type errorString string

func (e errorString) Error() string { return string(e) }
