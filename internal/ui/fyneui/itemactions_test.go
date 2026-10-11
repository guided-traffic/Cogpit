package fyneui

import (
	"encoding/json"
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
)

// A drawer item's action is a button beside its value that dispatches its command with its
// params; a command with its own GUI dialog opens that instead.
func TestDrawerItemActions(t *testing.T) {
	f := kindsFake(t, "pods")
	ref := model.ObjectRef{Context: "demo", Kind: "pods", Namespace: "shop", Name: "web-0"}
	params := json.RawMessage(`{"ref":{"context":"demo","kind":"pods","namespace":"shop","name":"web-0"},"container":"app"}`)
	f.details[ref.String()] = &engine.DetailModel{Ref: ref, Title: "Pod: web-0", Sections: []model.DetailSection{{
		Title: "app", Items: []model.DetailItem{
			{Name: "Ports", Value: "http: 8080/TCP", Actions: []model.ItemAction{{Label: "Logs…", Command: "pod.logs", Params: params}}},
			{Name: "Other", Value: "x", Actions: []model.ItemAction{{Label: "Own dialog", Command: "test.own"}}},
		},
	}}}
	opened := 0
	itemActionDialogs["test.own"] = func(*shell, model.ObjectRef, model.ItemAction) { opened++ }
	t.Cleanup(func() { delete(itemActionDialogs, "test.own") })
	f.ui.Selected, f.ui.Drawer = []model.ObjectRef{ref}, &ref
	sh, _, q := newTestShell(t, f, Options{})
	q.until(t, "the drawer", func() bool { return sh.drawer.last != nil && len(findAll[*widget.Button](sh.drawer.details)) == 2 })

	buttons := findAll[*widget.Button](sh.drawer.details)
	test.Tap(buttons[0])
	q.until(t, "pod.logs", func() bool { return len(f.callsOf("pod.logs")) == 1 })
	if got := string(f.callsOf("pod.logs")[0].Params); got != string(params) {
		t.Errorf("params %s, want %s", got, params)
	}
	test.Tap(buttons[1])
	if opened != 1 {
		t.Errorf("the registered dialog opened %d times", opened)
	}
}
