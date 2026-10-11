package kinds

import (
	"strings"
	"testing"

	"github.com/guided-traffic/cogpit/internal/model"
)

// TestPodRestartsAloneDoNotWarn: Cogpit's (inferred) warning rule needs a problem besides restarts.
func TestPodRestartsAloneDoNotWarn(t *testing.T) {
	// Restarts alone do not warn.
	obj := pod(map[string]any{"containers": []any{map[string]any{"name": "app"}}}, map[string]any{
		"phase":             "Running",
		"containerStatuses": []any{map[string]any{"name": "app", "ready": true, "restartCount": int64(9), "state": map[string]any{"running": map[string]any{}}}},
	}, nil)
	if c := podWarning(obj, summarisePod(obj)); c != (model.Cell{}) {
		t.Errorf("a ready pod with restarts warns: %+v", c)
	}
}

// TestPodLinks pins the links of the Pods list: Controlled By to the owner (built-in kinds by
// their key), Node to the node, Namespace narrowing the filter (namespaceCell).
func TestPodLinks(t *testing.T) {
	p := pods
	owner := func(apiVersion, kind string) map[string]any {
		return map[string]any{"name": "p", "namespace": "shop", "ownerReferences": []any{
			map[string]any{"apiVersion": apiVersion, "kind": kind, "name": "owner-1", "controller": true},
		}}
	}
	tests := []struct {
		meta map[string]any
		want *model.ObjectRef
	}{
		{owner("apps/v1", "ReplicaSet"), &model.ObjectRef{Kind: "replicasets", Namespace: "shop", Name: "owner-1"}},
		{owner("apps/v1", "StatefulSet"), &model.ObjectRef{Kind: "statefulsets", Namespace: "shop", Name: "owner-1"}},
		{owner("apps/v1", "DaemonSet"), &model.ObjectRef{Kind: "daemonsets", Namespace: "shop", Name: "owner-1"}},
		{owner("batch/v1", "Job"), &model.ObjectRef{Kind: "jobs", Namespace: "shop", Name: "owner-1"}},
		{owner("v1", "Node"), &model.ObjectRef{Kind: "nodes", Name: "owner-1"}},
		{owner("example.com/v1", "Widget"), nil}, // a custom kind: its key needs discovery
	}
	col := p.ColumnIndex("controlledBy")
	for _, tt := range tests {
		obj := pod(map[string]any{"nodeName": "worker-1"}, map[string]any{}, tt.meta)
		cells, _ := podCells(asUnstructured(obj))
		got := cells[col].Link
		if (got == nil) != (tt.want == nil) || (got != nil && *got != *tt.want) {
			t.Errorf("%v: link %+v, want %+v", tt.meta["ownerReferences"], got, tt.want)
		}
		if node := cells[p.ColumnIndex("node")]; node.Link == nil || *node.Link != (model.ObjectRef{Kind: "nodes", Name: "worker-1"}) {
			t.Errorf("node link %+v", node.Link)
		}
	}
	unscheduled := pod(nil, map[string]any{}, nil)
	cells, _ := podCells(asUnstructured(unscheduled))
	if cells[p.ColumnIndex("node")].Link != nil || cells[col].Link != nil {
		t.Error("a pod without node or owner must have no links")
	}
}

func TestPodDetailSections(t *testing.T) {
	obj := pod(map[string]any{
		"serviceAccountName": "web",
		"initContainers":     []any{map[string]any{"name": "migrate", "image": "m:1"}},
		"containers": []any{map[string]any{
			"name": "app", "image": "a:1",
			"env":       []any{map[string]any{"name": "TOKEN", "valueFrom": map[string]any{"secretKeyRef": map[string]any{"name": "api-token", "key": "t"}}}},
			"resources": map[string]any{"requests": map[string]any{"memory": "64Mi", "cpu": "50m"}},
		}},
		"imagePullSecrets": []any{map[string]any{"name": "registry"}},
		"tolerations":      []any{map[string]any{"operator": "Exists"}},
		"volumes": []any{
			map[string]any{"name": "cfg", "configMap": map[string]any{"name": "web-config"}},
			map[string]any{"name": "creds", "secret": map[string]any{"secretName": "api-token"}},
			map[string]any{"name": "scratch", "emptyDir": map[string]any{}},
		},
		"affinity": map[string]any{"podAntiAffinity": map[string]any{
			"requiredDuringSchedulingIgnoredDuringExecution":  []any{map[string]any{}},
			"preferredDuringSchedulingIgnoredDuringExecution": []any{map[string]any{}, map[string]any{}},
		}},
	}, map[string]any{"phase": "Running", "containerStatuses": []any{running(true)}}, map[string]any{"name": "p", "namespace": "shop"})
	sections := podDetail(asUnstructured(obj))
	var titles []string
	for _, s := range sections {
		titles = append(titles, s.Group+"/"+s.Title)
	}
	if got := strings.Join(titles, ","); got != "/Properties,/Pod Volumes,Init Containers/migrate,Containers/app" {
		t.Fatalf("sections = %s", got)
	}
	props := map[string]model.DetailItem{}
	for _, it := range sections[0].Items {
		props[it.Name] = it
	}
	if sa := props["Service Account"]; sa.Link == nil || *sa.Link != (model.ObjectRef{Kind: "serviceaccounts", Namespace: "shop", Name: "web"}) {
		t.Errorf("service account %+v", sa)
	}
	if sec := props["Secrets"]; sec.Value != "api-token\nregistry" || sec.Chips[0].Link == nil || sec.Chips[0].Link.Kind != "secrets" {
		t.Errorf("secrets %+v", sec)
	}
	if tol := props["Tolerations"]; tol.Value != "*" || tol.Fold != "Tolerations" {
		t.Errorf("tolerations %+v", tol)
	}
	if anti := props["Pod Anti Affinities"]; anti.Value != "3" {
		t.Errorf("anti affinities %+v", anti)
	}
	vols := sections[1].Items
	if len(vols) != 3 || vols[0].Name != "ConfigMap" || vols[0].Fold != "volumes" || vols[0].Chips[0].Link == nil ||
		*vols[0].Chips[0].Link != (model.ObjectRef{Kind: "configmaps", Namespace: "shop", Name: "web-config"}) || vols[2].Name != "EmptyDir" {
		t.Errorf("volumes %+v", vols)
	}
	app := map[string]string{}
	for _, it := range sections[3].Items {
		app[it.Name] = it.Value
	}
	if app["Status"] != "Running, ready" || app["Requests"] != "cpu: 50m, memory: 64Mi" || app["Environment"] != "TOKEN: from secret api-token/t" {
		t.Errorf("container items %v", app)
	}
}

func TestTolerationText(t *testing.T) {
	for _, tt := range []struct {
		in   map[string]any
		want string
	}{
		{map[string]any{"key": "dedicated", "operator": "Equal", "value": "gpu", "effect": "NoSchedule"}, "dedicated=gpu:NoSchedule"},
		{map[string]any{"key": "node.kubernetes.io/not-ready", "operator": "Exists", "effect": "NoExecute", "tolerationSeconds": int64(300)}, "node.kubernetes.io/not-ready:NoExecute for 300s"},
		{map[string]any{"operator": "Exists"}, "*"},
		{map[string]any{}, ""},
	} {
		if got := tolerationText(tt.in); got != tt.want {
			t.Errorf("tolerationText(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
