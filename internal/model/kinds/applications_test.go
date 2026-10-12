package kinds_test

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	utiljson "k8s.io/apimachinery/pkg/util/json"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

// appCase is one recorded workload placed in an application: testdata/<kind>/<recording>.yaml
// with its namespace and recommended labels replaced.
type appCase struct {
	kind, recording, namespace, instance, version, managedBy string
}

// appCases group recorded workloads into four applications: shop/storefront (a healthy
// Deployment, a degraded one, a rolling Stateful Set, a Cron Job, and the recorded healthy Job,
// which that recording's Cron Job controls), shop/search in two versions, batch/nightly (a failed
// Job) and ops/agents (a Daemon Set); a Deployment without the instance label belongs to none.
var appCases = []appCase{
	{"deployments", "healthy", "shop", "storefront", "1.4.2", "Helm"},
	{"deployments", "degraded", "shop", "storefront", "1.4.2", "Helm"},
	{"statefulsets", "rolling", "shop", "storefront", "1.4.2", "Helm"},
	{"cronjobs", "healthy", "shop", "storefront", "1.4.2", "Helm"},
	{"jobs", "healthy", "shop", "storefront", "1.4.2", "Helm"},
	{"deployments", "rolling-out", "shop", "search", "2.0.0", "kustomize"},
	{"statefulsets", "healthy", "shop", "search", "2.1.0", "Helm"},
	{"jobs", "failing", "batch", "nightly", "", ""},
	{"daemonsets", "healthy", "ops", "agents", "0.9", "argocd"},
	{"deployments", "minimal", "shop", "", "", ""},
}

func appWorkloads(t testing.TB) []kinds.AppWorkload {
	t.Helper()
	var out []kinds.AppWorkload
	for _, c := range appCases {
		obj := recorded(t, c.kind, c.recording)
		obj.SetNamespace(c.namespace)
		labels := map[string]string{}
		for k, v := range map[string]string{
			kinds.LabelAppInstance: c.instance, kinds.LabelAppVersion: c.version, kinds.LabelAppManagedBy: c.managedBy,
			kinds.LabelAppName: c.instance, kinds.LabelAppPartOf: "shop",
		} {
			if v != "" && c.instance != "" {
				labels[k] = v
			}
		}
		obj.SetLabels(labels)
		p, _ := kinds.ByKey(c.kind)
		out = append(out, kinds.AppWorkload{Kind: c.kind, Object: obj, Status: model.BuildRow(p, "test", obj).Status})
	}
	// The Cron Job's run: a Job it controls, which the Cron Job stands for.
	run := recorded(t, "jobs", "running")
	run.SetNamespace("shop")
	run.SetLabels(map[string]string{kinds.LabelAppInstance: "storefront"})
	_ = unstructured.SetNestedSlice(run.Object, []any{map[string]any{
		"apiVersion": "batch/v1", "kind": "CronJob", "name": "report", "uid": "x", "controller": true,
	}}, "metadata", "ownerReferences")
	out = append(out, kinds.AppWorkload{Kind: "jobs", Object: run, Status: model.StatusPending})
	return out
}

// TestApplicationsGolden groups the recorded workloads and renders each application as the view
// and the drawer show it; the recorded Application objects render as their goldens too.
func TestApplicationsGolden(t *testing.T) {
	p := kinds.Applications()
	apps := kinds.GroupApplications(appWorkloads(t))
	var names []string
	for _, a := range apps {
		names = append(names, a.GetNamespace()+"/"+a.GetName())
	}
	if got := strings.Join(names, ", "); got != "batch/nightly, ops/agents, shop/search, shop/storefront" {
		t.Fatalf("applications %s", got)
	}
	var grouped []golden
	for _, a := range apps {
		grouped = append(grouped, render(t, p, a))
	}
	checkGolden(t, filepath.Join("testdata", "applications", "grouped.golden.json"), encode(t, grouped))
	for _, c := range loadCases(t, "applications") {
		t.Run(c.name, func(t *testing.T) {
			checkGolden(t, strings.TrimSuffix(c.path, ".yaml")+".golden.json", encode(t, render(t, p, c.obj)))
		})
	}
}

// TestGroupApplications states the grouping rules outright: by namespace and instance, a Cron
// Job's Job and an unlabelled workload left out, the oldest workload's age, distinct sorted
// versions, the worst status and the violations, a stable UID.
func TestGroupApplications(t *testing.T) {
	p := kinds.Applications()
	ws := appWorkloads(t)
	apps := kinds.GroupApplications(ws)
	byName := map[string]*unstructured.Unstructured{}
	for _, a := range apps {
		byName[a.GetNamespace()+"/"+a.GetName()] = a
	}
	cell := func(app, id string) model.Cell {
		row := model.BuildRow(p, "test", byName[app])
		return row.Cells[p.ColumnIndex(id)]
	}
	store := byName["shop/storefront"]
	var refs []string
	for _, r := range kinds.ApplicationWorkloads(store) {
		refs = append(refs, r.Kind+"/"+r.Name)
	}
	// The two Jobs a Cron Job controls (the recorded healthy one and the added run) are left out.
	if got := strings.Join(refs, ", "); got != "deployments/gateway, deployments/web, statefulsets/search, cronjobs/nightly-report" {
		t.Errorf("storefront's workloads %s", got)
	}
	if got := cell("shop/search", "version").Text; got != "2.0.0, 2.1.0" {
		t.Errorf("search's versions %q", got)
	}
	if got := cell("shop/search", "managedBy").Text; got != "Helm, kustomize" {
		t.Errorf("search's managers %q", got)
	}
	if got := cell("batch/nightly", "status"); got.Text != "Failing" || got.Status != model.StatusError {
		t.Errorf("nightly's status %+v", got)
	}
	if got := cell("batch/nightly", "violations").Num; got != 1 {
		t.Errorf("nightly's violations %d", got)
	}
	if got := cell("shop/storefront", "status"); got.Status == model.StatusOK {
		t.Errorf("storefront with a degraded Deployment is %+v", got)
	}
	if got := cell("ops/agents", "status"); got.Text != "Healthy" || got.Status != model.StatusOK {
		t.Errorf("agents' status %+v", got)
	}
	again := kinds.GroupApplications(ws)
	if again[0].GetUID() != apps[0].GetUID() || apps[0].GetUID() == apps[1].GetUID() {
		t.Error("an application's UID is not stable, or two share one")
	}
	if l := cell("shop/storefront", "namespace").Link; l == nil || l.Kind != "namespaces" {
		t.Errorf("namespace link %+v", l)
	}
	if kinds.GroupApplications(nil) == nil || len(kinds.GroupApplications([]kinds.AppWorkload{{Kind: "pods", Object: ws[0].Object}})) != 0 {
		t.Error("pods make applications, or nothing makes nil")
	}
}

// TestApplicationsProfile: a virtual kind of Cogpit's own, titled and keyed as documented.
func TestApplicationsProfile(t *testing.T) {
	p := kinds.Applications()
	if p.Key != "applications" || p.Group != model.LocalGroup || p.Derive == nil || p.APIObject() || p.Local || !p.Namespaced {
		t.Errorf("profile %+v", p)
	}
	if _, builtin := kinds.ByKey(p.Key); builtin {
		t.Error("Applications listed as a built-in kind")
	}
	var titles []string
	for _, c := range p.Columns {
		title := c.Title
		if c.Hidden {
			title = "(" + title + ")"
		}
		titles = append(titles, title)
	}
	if got := strings.Join(titles, ", "); got != "Instance, Namespace, Managed by, Version, Violations, Age, Status, (Name), (Workloads)" {
		t.Errorf("columns %s", got)
	}
}

func FuzzApplications(f *testing.F) {
	p := kinds.Applications()
	for _, c := range loadCases(f, "applications") {
		f.Add(c.json)
	}
	f.Add([]byte(`{"spec":{"workloads":[1,{"key":"pods"}]},"status":{"violations":-3}}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		for _, decode := range []func([]byte, any) error{utiljson.Unmarshal, json.Unmarshal} {
			var m map[string]any
			if decode(data, &m) != nil || m == nil {
				continue
			}
			obj := &unstructured.Unstructured{Object: m}
			if err := exercise(p, obj); err != nil {
				t.Fatal(err)
			}
			// The same object as every workload kind: grouping must not panic either.
			var ws []kinds.AppWorkload
			for _, k := range kinds.ApplicationWorkloadKinds {
				ws = append(ws, kinds.AppWorkload{Kind: k, Object: obj, Status: model.StatusWarning})
			}
			for _, app := range kinds.GroupApplications(ws) {
				if err := exercise(p, app); err != nil {
					t.Fatal(err)
				}
			}
		}
	})
}
