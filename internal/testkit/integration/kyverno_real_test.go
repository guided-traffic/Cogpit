//go:build kind

package integration_test

import (
	"context"
	"slices"
	"sort"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// TestRealKyverno checks the Kyverno integration (D-79) on a real cluster that runs Kyverno (make
// test-real, REAL_RUN=TestRealKyverno). The cluster's own policies are left alone: in a namespace of
// its own (cogpit-real-kyverno, deleted at the end) the test makes two namespaced Kyverno Policies
// that only background scans apply (admission: false, so no webhook changes and nothing is ever
// refused), one in Audit and one in Enforce, and the objects that violate them; once Kyverno's
// reports name them, what Cogpit marks must be what the reports say, a namespaced Policy's link
// must name it, and the cache's estimate of the reports with and without the trim is logged
// (numbers only, never a name).
func TestRealKyverno(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	s, _, _ := realSession(ctx, t, engine.Options{})
	b := s.Backend()
	if _, ok := b.Profile(kinds.PolicyReportsKey); !ok {
		t.Skip("the cluster serves no policy reports")
	}
	policies := schema.GroupVersionResource{Group: "kyverno.io", Version: "v1", Resource: "policies"}
	if _, ok := b.Profile("policies.kyverno.io"); !ok {
		t.Skip("the cluster serves no Kyverno Policies")
	}
	ns := realNamespace(ctx, t, s, "kyverno")
	dyn := b.Conn.Dynamic()
	create := func(gvr schema.GroupVersionResource, obj *unstructured.Unstructured) {
		t.Helper()
		obj.SetUID("")
		obj.SetCreationTimestamp(metav1.Time{})
		if _, err := dyn.Resource(gvr).Namespace(ns).Create(ctx, obj, metav1.CreateOptions{}); err != nil {
			t.Fatalf("creating %s %s: %v", obj.GetKind(), obj.GetName(), err)
		}
	}
	configMaps := schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}
	create(configMaps, testkit.Object("v1", "ConfigMap", ns, "no-owner", map[string]any{"data": map[string]any{"a": "b"}}))
	owned := testkit.Object("v1", "ConfigMap", ns, "has-owner", map[string]any{"data": map[string]any{"a": "b"}})
	owned.SetLabels(map[string]string{"owner": "cogpit"})
	create(configMaps, owned)
	create(testkit.ServicesGVR, testkit.Object("v1", "Service", ns, "no-team", map[string]any{
		"spec": map[string]any{"ports": []any{map[string]any{"port": int64(80)}}},
	}))
	// Each policy matches the test's own objects by name: a cluster may make objects of its own in
	// every namespace (a trust bundle's ConfigMaps), which are not the test's to judge.
	policy := func(name, action, kind, label string, names ...any) *unstructured.Unstructured {
		return testkit.Object("kyverno.io/v1", "Policy", ns, name, map[string]any{"spec": map[string]any{
			"admission": false, "background": true,
			"rules": []any{map[string]any{
				"name": "check-" + label,
				"match": map[string]any{"any": []any{map[string]any{"resources": map[string]any{
					"kinds": []any{kind}, "names": names}}}},
				"validate": map[string]any{
					"failureAction": action,
					"message":       "The label " + label + " is required.",
					"pattern":       map[string]any{"metadata": map[string]any{"labels": map[string]any{label: "?*"}}},
				},
			}},
		}})
	}
	create(policies, policy("cogpit-configmap-owner", "Audit", "ConfigMap", "owner", "no-owner", "has-owner"))
	create(policies, policy("cogpit-service-team", "Enforce", "Service", "team", "no-team"))

	// What Kyverno's reports say about the namespace, past Cogpit.
	polr := schema.GroupVersionResource{Group: kinds.PolicyReportGroup, Version: "v1alpha2", Resource: "policyreports"}
	reported := func() []string {
		l, err := dyn.Resource(polr).Namespace(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil
		}
		var out []string
		for _, r := range l.Items {
			scope, _ := kinds.PolicyReportScope(r.Object)
			for _, res := range kinds.PolicyReportResults(r.Object, kinds.IsPolicyViolation) {
				out = append(out, scope.Kind+"/"+scope.Name+":"+res.Policy)
			}
		}
		sort.Strings(out)
		return out
	}
	want := []string{"ConfigMap/no-owner:" + ns + "/cogpit-configmap-owner", "Service/no-team:" + ns + "/cogpit-service-team"}
	waitFor(ctx, t, "Kyverno's background reports of the test's objects", func() bool { return slices.Equal(reported(), want) })

	var res commands.KyvernoViolationsResult
	waitFor(ctx, t, "kyverno_violations to list them", func() bool {
		r, err := s.Dispatch(ctx, engine.Invocation{ID: "kyverno.violations", Params: []byte(`{"namespaces":["` + ns + `"]}`), Origin: engine.OriginExternalAgent})
		if err != nil {
			return false
		}
		res = r.(commands.KyvernoViolationsResult)
		return res.Total == len(want)
	})
	var got []string
	for _, v := range res.Violations {
		got = append(got, map[string]string{"configmaps": "ConfigMap", "services": "Service"}[v.Object.Kind]+"/"+v.Object.Name+":"+v.Policy)
		if ref := v.PolicyRef(); ref == nil || ref.Kind != "policies.kyverno.io" || ref.Namespace != ns {
			t.Errorf("%s's policy link %+v", v.Object.Name, ref)
		}
	}
	sort.Strings(got)
	if !slices.Equal(got, want) {
		t.Errorf("kyverno_violations %v, the reports %v", got, want)
	}

	for view, marked := range map[string][]string{"configmaps": {"no-owner"}, "services": {"no-team"}} {
		s.UpdateUI(func(u *engine.UIState) { u.View, u.Namespaces, u.Filter = view, []string{ns}, "" })
		var tbl *engine.TableSnapshot
		if !testkit.WaitFor(ctx, func() bool {
			tbl = s.Table(view)
			return tbl != nil && tbl.Synced && slices.Equal(warned(tbl), marked)
		}) {
			t.Errorf("%s: marked %v, want %v", view, warned(tbl), marked)
		}
	}
	m, err := s.DetailWait(ctx, model.ObjectRef{Context: b.Context, Kind: "services", Namespace: ns, Name: "no-team"})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(m.Sections, func(sec model.DetailSection) bool {
		return sec.Group == "Policy violations" && sec.Title == ns+"/cogpit-service-team"
	}) {
		t.Errorf("the drawer's sections %+v", m.Sections)
	}

	// The memory the cluster's reports take in the cache, trimmed and whole (D-79's number).
	trimmed := b.Cache.Acquire(cache.Scope{GVR: polr, Trim: "kyverno-violations"})
	defer trimmed.Release()
	whole := b.Cache.Acquire(cache.Scope{GVR: polr})
	defer whole.Release()
	waitFor(ctx, t, "both scopes to list", func() bool { return trimmed.HasSynced() && whole.HasSynced() })
	for _, st := range b.Cache.StatsWithMemory() {
		if st.Scope.GVR == polr && st.Scope.Namespace == "" {
			t.Logf("policy reports %s: %d objects, %d bytes estimated", st.Scope, st.Objects, st.Bytes)
		}
	}
}
