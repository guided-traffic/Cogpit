package kinds

import (
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func argoWindowApp(name, ns, server string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "argoproj.io/v1alpha1", "kind": "Application",
		"metadata": map[string]any{"name": name, "namespace": "argocd"},
		"spec":     map[string]any{"project": "team", "destination": map[string]any{"namespace": ns, "server": server}},
	}}
}

func argoWindowProject(windows ...map[string]any) *unstructured.Unstructured {
	l := make([]any, 0, len(windows))
	for _, w := range windows {
		l = append(l, w)
	}
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "argoproj.io/v1alpha1", "kind": "AppProject",
		"metadata": map[string]any{"name": "team", "namespace": "argocd"},
		"spec":     map[string]any{"syncWindows": l},
	}}
}

// TestArgoSyncBlocked (A33): a manual sync against an AppProject's windows, as Argo CD checks it.
func TestArgoSyncBlocked(t *testing.T) {
	// 2026-10-14 is a Wednesday; at 10:30 UTC.
	now := time.Date(2026, 10, 14, 10, 30, 0, 0, time.UTC)
	app := argoWindowApp("shop-web", "shop", "https://kubernetes.default.svc")
	denyNow := map[string]any{"kind": "deny", "schedule": "0 10 * * *", "duration": "1h", "applications": []any{"shop-*"}}
	denyLater := map[string]any{"kind": "deny", "schedule": "0 22 * * *", "duration": "1h", "applications": []any{"*"}}
	allowNow := map[string]any{"kind": "allow", "schedule": "0 9 * * 3", "duration": "4h", "namespaces": []any{"shop"}}
	allowLater := map[string]any{"kind": "allow", "schedule": "0 22 * * *", "duration": "2h", "clusters": []any{"https://kubernetes.default.svc"}}
	otherApp := map[string]any{"kind": "deny", "schedule": "0 10 * * *", "duration": "1h", "applications": []any{"billing-*"}}
	berlin := map[string]any{"kind": "deny", "schedule": "0 12 * * *", "duration": "1h", "applications": []any{"*"}, "timeZone": "Europe/Berlin"}
	for _, c := range []struct {
		name    string
		windows []map[string]any
		blocked string // a part of the reason, "" for not blocked
	}{
		{"no windows", nil, ""},
		{"an active deny", []map[string]any{denyNow}, "deny window \"0 10 * * *\""},
		{"an active deny that allows manual syncs", []map[string]any{withManual(denyNow)}, ""},
		{"a deny not active", []map[string]any{denyLater}, ""},
		{"a deny for another app", []map[string]any{otherApp}, ""},
		{"an active allow", []map[string]any{allowNow}, ""},
		{"allow windows, none active", []map[string]any{allowLater}, "no allow window"},
		{"allow windows, none active, manual allowed", []map[string]any{withManual(allowLater)}, ""},
		{"deny beats allow", []map[string]any{allowNow, denyNow}, "deny window"},
		{"a deny in Berlin's noon (10:00 UTC)", []map[string]any{berlin}, "Europe/Berlin"},
		{"andOperator: name matches, namespace not", []map[string]any{{"kind": "deny", "schedule": "0 10 * * *", "duration": "1h",
			"applications": []any{"shop-*"}, "namespaces": []any{"billing"}, "andOperator": true}}, ""},
		{"andOperator: both match", []map[string]any{{"kind": "deny", "schedule": "0 10 * * *", "duration": "1h",
			"applications": []any{"shop-*"}, "namespaces": []any{"shop"}, "andOperator": true}}, "deny window"},
	} {
		got, err := ArgoSyncBlocked(app, argoWindowProject(c.windows...), now)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if (c.blocked == "") != (got == "") || !strings.Contains(got, c.blocked) {
			t.Errorf("%s: %q, want %q", c.name, got, c.blocked)
		}
	}
	if got, _ := ArgoSyncBlocked(app, argoWindowProject(denyNow), now); !strings.Contains(got, "until 2026-10-14T11:00:00Z") {
		t.Errorf("the deny's end: %q", got)
	}
	for _, bad := range []map[string]any{
		{"kind": "maybe", "schedule": "0 10 * * *", "duration": "1h"},
		{"kind": "deny", "schedule": "not cron", "duration": "1h"},
		{"kind": "deny", "schedule": "0 10 * * *", "duration": "soon"},
		{"kind": "deny", "schedule": "0 10 * * *", "duration": "1h", "timeZone": "Mars/Olympus"},
	} {
		if _, err := ArgoSyncBlocked(app, argoWindowProject(bad), now); err == nil {
			t.Errorf("a window that cannot be read passed: %v", bad)
		}
	}
}

func withManual(w map[string]any) map[string]any {
	out := map[string]any{"manualSync": true}
	for k, v := range w {
		out[k] = v
	}
	return out
}
