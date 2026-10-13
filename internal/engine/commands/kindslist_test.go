package commands_test

import (
	"slices"
	"testing"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

// TestKindsListActionsCacheAndServed: each kind carries the tools of its objects' row menu and
// drawer in Lens's order, of a selection's menu and its other tools; cached follows Cogpit's cache;
// with all, the built-in kinds the cluster does not serve follow, marked, without actions.
func TestKindsListActionsCacheAndServed(t *testing.T) {
	h := newHarness(t, seed(), engine.Options{})
	waitTable(t, h.s, "pods", func(ts *engine.TableSnapshot) bool { return ts.Synced })
	list := func(all bool) map[string]commands.KindInfo {
		var got []commands.KindInfo
		h.must(t, engine.OriginExternalAgent, "kinds.list", map[string]any{"all": all}, &got)
		out := map[string]commands.KindInfo{}
		for _, k := range got {
			if _, dup := out[k.Kind]; dup {
				t.Errorf("%s listed twice", k.Kind)
			}
			out[k.Kind] = k
		}
		return out
	}
	got := list(false)
	pods, ok := got["pods"]
	if !ok {
		t.Fatal("no pods")
	}
	want := []string{"pod_attach", "pod_shell", "pod_evict", "pod_logs"}
	if i := slices.Index(pods.Actions, "object_edit"); i < 0 || pods.Actions[len(pods.Actions)-1] != "object_delete" ||
		!slices.Equal(filter(pods.Actions, want), want) {
		t.Errorf("pods' actions %v: want %v in that order, then Edit, Delete last", pods.Actions, want)
	}
	if !slices.Contains(pods.SelectionActions, "object_deleteMany") {
		t.Errorf("pods' selection actions %v lack object_deleteMany", pods.SelectionActions)
	}
	for _, name := range pods.Commands {
		if slices.Contains(pods.Actions, name) || slices.Contains(pods.SelectionActions, name) {
			t.Errorf("%s is listed among the actions and the other commands", name)
		}
	}
	if dep := got["deployments"]; !slices.Contains(dep.Actions, "deployment_scale") || slices.Contains(dep.Actions, "pod_shell") {
		t.Errorf("deployments' actions %v", dep.Actions)
	}
	if !pods.Cached {
		t.Error("pods: not cached while its view is open")
	}
	if got["services"].Cached {
		t.Error("services: cached though nothing read them")
	}
	for key, k := range got {
		if k.NotServed {
			t.Errorf("%s: notServed without all", key)
		}
	}
	if _, ok := got["configmaps"]; ok {
		t.Error("configmaps, which the test cluster does not serve, listed without all")
	}

	all := list(true)
	cm, ok := all["configmaps"]
	if !ok || !cm.NotServed || len(cm.Actions) != 0 || cm.Listable != nil || cm.Cached || cm.Title == "" || len(cm.Columns) == 0 {
		t.Errorf("configmaps with all: %+v", cm)
	}
	if all["pods"].NotServed {
		t.Error("pods marked not served")
	}
	if n := len(all) - len(got); n != len(kinds.Builtin())-len(served(got)) {
		t.Errorf("%d kinds not served listed, want every built-in one the cluster lacks", n)
	}
}

// filter keeps the elements of s that are in keep, in s's order.
func filter(s, keep []string) []string {
	var out []string
	for _, x := range s {
		if slices.Contains(keep, x) {
			out = append(out, x)
		}
	}
	return out
}

// served are the listed kinds that are built-in ones.
func served(got map[string]commands.KindInfo) []string {
	var out []string
	for _, p := range kinds.Builtin() {
		if _, ok := got[p.Key]; ok {
			out = append(out, p.Key)
		}
	}
	return out
}
