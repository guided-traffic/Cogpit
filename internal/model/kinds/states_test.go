package kinds_test

import (
	"slices"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

// statesOf builds the row of an object with a kind's profile and returns its states as the status
// filter names them.
func statesOf(t *testing.T, p *model.KindProfile, obj map[string]any) []string {
	t.Helper()
	row := model.BuildRow(p, "test", &unstructured.Unstructured{Object: obj})
	var out []string
	for _, s := range model.StatesOf(p).Of(&row) {
		out = append(out, s.State)
	}
	return out
}

func builtin(t *testing.T, key string) *model.KindProfile {
	t.Helper()
	p, ok := kinds.ByKey(key)
	if !ok {
		t.Fatalf("no profile %s", key)
	}
	return p
}

// TestKindStates: the status filter's states of Cogpit's own kinds: a pod's Status, a
// deployment's and a node's conditions one by one, an event's Type, an autoscaler's condition
// types one by one, a Flux object's state, an Argo CD Application's Sync and Health qualified by
// their columns, and a kind without a state column its rows' status tokens.
func TestKindStates(t *testing.T) {
	meta := func(name string) map[string]any { return map[string]any{"name": name, "namespace": "shop"} }
	cases := []struct {
		name string
		p    *model.KindProfile
		obj  map[string]any
		want []string
	}{
		{"pod", builtin(t, "pods"), map[string]any{"metadata": meta("web"), "status": map[string]any{"phase": "Pending"}}, []string{"Pending"}},
		{"deployment", builtin(t, "deployments"), map[string]any{"metadata": meta("web"), "status": map[string]any{"conditions": []any{
			map[string]any{"type": "Progressing", "status": "True"}, map[string]any{"type": "Available", "status": "True"},
			map[string]any{"type": "ReplicaFailure", "status": "False"}}}}, []string{"Available", "Progressing"}},
		{"node", builtin(t, "nodes"), map[string]any{"metadata": map[string]any{"name": "n1"}, "spec": map[string]any{"unschedulable": true},
			"status": map[string]any{"conditions": []any{map[string]any{"type": "Ready", "status": "True"}}}}, []string{"SchedulingDisabled", "Ready"}},
		{"event", builtin(t, "events"), map[string]any{"metadata": meta("e1"), "type": "Warning", "reason": "BackOff"}, []string{"Warning"}},
		{"autoscaler", builtin(t, "horizontalpodautoscalers"), map[string]any{"metadata": meta("web"), "status": map[string]any{"conditions": []any{
			map[string]any{"type": "AbleToScale", "status": "True"}, map[string]any{"type": "ScalingActive", "status": "True"}}}},
			[]string{"AbleToScale", "ScalingActive"}},
		{"kustomization", curatedProfile(t, fluxKustomizations), map[string]any{"metadata": meta("apps"), "spec": map[string]any{"suspend": true}},
			[]string{"Suspended"}},
		{"stateful set", builtin(t, "statefulsets"), map[string]any{"metadata": meta("db"), "spec": map[string]any{"replicas": int64(2)},
			"status": map[string]any{"readyReplicas": int64(2), "replicas": int64(2)}}, []string{"OK"}},
		{"config map", builtin(t, "configmaps"), map[string]any{"metadata": meta("cfg")}, nil},
	}
	for _, c := range cases {
		if got := statesOf(t, c.p, c.obj); !slices.Equal(got, c.want) {
			t.Errorf("%s: states %q, want %q", c.name, got, c.want)
		}
	}

	argo := curatedProfile(t, kinds.ArgoApplicationsKey)
	app := map[string]any{"metadata": meta("web"), "status": map[string]any{
		"sync": map[string]any{"status": "OutOfSync"}, "health": map[string]any{"status": "Degraded"}}}
	if got := statesOf(t, argo, app); !slices.Equal(got, []string{"Sync: OutOfSync", "Health: Degraded"}) {
		t.Errorf("Application states %q", got)
	}
	if got := model.StatesOf(argo).Columns(); !slices.Equal(got, []string{"Sync", "Health"}) {
		t.Errorf("Application state columns %q", got)
	}
}
