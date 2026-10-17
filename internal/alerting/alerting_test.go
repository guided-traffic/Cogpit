package alerting

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// Classify: the owner's colours (critical red, warning and major orange, info yellow), severity none
// and the meta alerts left out, a missing or unknown label as warning (D-77).
func TestClassify(t *testing.T) {
	for _, c := range []struct {
		name, label string
		want        Severity
		keep        bool
	}{
		{"KubePodCrashLooping", "critical", SeverityCritical, true},
		{"X", " Critical ", SeverityCritical, true},
		{"X", "warning", SeverityWarning, true},
		{"KustomizationNotReady", "major", SeverityWarning, true},
		{"X", "info", SeverityInfo, true},
		{"X", "INFO", SeverityInfo, true},
		{"X", "", SeverityWarning, true},
		{"X", "page", SeverityWarning, true},
		{"X", "none", "", false},
		{"Watchdog", "none", "", false},
		{"Watchdog", "critical", "", false},
		{"InfoInhibitor", "info", "", false},
	} {
		got, keep := Classify(c.name, c.label)
		if got != c.want || keep != c.keep {
			t.Errorf("Classify(%q, %q) = %q, %v; want %q, %v", c.name, c.label, got, keep, c.want, c.keep)
		}
	}
	if SeverityCritical.Rank() >= SeverityWarning.Rank() || SeverityWarning.Rank() >= SeverityInfo.Rank() || Severity("x").Rank() <= SeverityInfo.Rank() {
		t.Error("ranks out of order")
	}
}

func wire(t *testing.T, alerts ...map[string]any) []byte {
	t.Helper()
	b, err := json.Marshal(alerts)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func wireAlertOf(name, severity string, at time.Time, ann map[string]string) map[string]any {
	labels := map[string]string{"alertname": name}
	if severity != "" {
		labels["severity"] = severity
	}
	return map[string]any{"fingerprint": "fp-" + name + severity, "labels": labels, "annotations": ann,
		"startsAt": at.Format(time.RFC3339Nano), "status": map[string]any{"state": "active"}}
}

// decodeAlerts: Alertmanager's v2 shape read, the left-out alerts not counted, description falling
// back to message, the bounds kept, and any other shape refused as not an Alertmanager.
func TestDecodeAlerts(t *testing.T) {
	at := time.Date(2026, 10, 17, 9, 15, 0, 0, time.UTC)
	long := strings.Repeat("ä", maxText) // two bytes a rune: cut at a rune boundary
	body := wire(t,
		wireAlertOf("A", "critical", at, map[string]string{"summary": "s", "description": "d", "runbook_url": " https://r "}),
		wireAlertOf("B", "major", at, map[string]string{"message": "old style", "summary": long}),
		wireAlertOf("Watchdog", "none", at, nil),
		wireAlertOf("C", "none", at, nil),
		wireAlertOf("D", "", at, nil),
	)
	alerts, total, truncated, err := decodeAlerts(body, 10)
	if err != nil || total != 3 || truncated || len(alerts) != 3 {
		t.Fatalf("decodeAlerts = %d alerts, total %d, truncated %v, %v", len(alerts), total, truncated, err)
	}
	a, b, d := alerts[0], alerts[1], alerts[2]
	if a.Name != "A" || a.Severity != SeverityCritical || a.Summary != "s" || a.Description != "d" || a.Runbook != "https://r" ||
		!a.StartsAt.Equal(at) || a.Labels["alertname"] != "A" || a.SeverityLabel != "critical" {
		t.Errorf("A = %+v", a)
	}
	if b.Severity != SeverityWarning || b.SeverityLabel != "major" || b.Description != "old style" || len(b.Summary) > maxText || !strings.HasPrefix(long, b.Summary) {
		t.Errorf("B = %q %q, summary %d bytes", b.Severity, b.Description, len(b.Summary))
	}
	if d.Severity != SeverityWarning || d.SeverityLabel != "" {
		t.Errorf("D without a severity = %+v", d)
	}

	// At most max kept; total counts them all.
	alerts, total, truncated, _ = decodeAlerts(body, 2)
	if len(alerts) != 2 || total != 3 || !truncated {
		t.Errorf("bounded: %d alerts, total %d, truncated %v", len(alerts), total, truncated)
	}
	// An empty list is an Alertmanager without alerts.
	if alerts, total, _, err := decodeAlerts([]byte(`[]`), 10); err != nil || total != 0 || len(alerts) != 0 {
		t.Errorf("empty: %v, %d, %v", alerts, total, err)
	}
	for _, other := range []string{`{"status":"success"}`, `[{"name":"x"}]`, `<html>`, `[1,2]`, `{"data":[]}`} {
		if _, _, _, err := decodeAlerts([]byte(other), 10); err != errNotAlertmanager {
			t.Errorf("%s: %v, want errNotAlertmanager", other, err)
		}
	}
}

func TestBoundLabels(t *testing.T) {
	in := map[string]string{"alertname": "A", "severity": "warning", "namespace": "ns"}
	for i := range 100 {
		in["l"+strings.Repeat("x", i%3)+string(rune('a'+i%26))+strings.Repeat("y", i/26)] = strings.Repeat("v", 2000)
	}
	out := boundLabels(in)
	if len(out) > maxLabels || out["alertname"] != "A" || out["severity"] != "warning" || out["namespace"] != "ns" {
		t.Errorf("%d labels, alertname %q", len(out), out["alertname"])
	}
	for k, v := range out {
		if len(v) > maxLabel {
			t.Errorf("%s: %d bytes", k, len(v))
		}
	}
}
