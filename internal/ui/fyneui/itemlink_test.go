package fyneui

import (
	"context"
	"encoding/json"
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
)

// An item action marked Link is the value itself: the value is a link that runs it, and the other
// actions are buttons after it. A command with its own dialog opens it from the row menu too.
func TestDrawerItemLinkAndRowMenuDialog(t *testing.T) {
	f := kindsFake(t, "pods")
	ref := model.ObjectRef{Context: "demo", Kind: "pods", Namespace: "shop", Name: "web-0"}
	quick := json.RawMessage(`{"ref":{"kind":"pods","namespace":"shop","name":"web-0"},"open":true}`)
	f.details[ref.String()] = &engine.DetailModel{Ref: ref, Title: "Pod: web-0", Sections: []model.DetailSection{{
		Title: "app", Items: []model.DetailItem{{Name: "Ports", Value: "http: 8080/TCP", Actions: []model.ItemAction{
			{Label: "Forward and open", Command: "test.forward", Params: quick, Link: true},
			{Label: "Forward…", Command: "test.forward"},
		}}},
	}}}
	var got []model.ItemAction
	itemActionDialogs["test.forward"] = func(_ *shell, _ model.ObjectRef, a model.ItemAction) { got = append(got, a) }
	t.Cleanup(func() { delete(itemActionDialogs, "test.forward") })
	if err := engine.Register(f.reg, engine.Spec[refParams]{ID: "test.forward", Label: "Forward", Kinds: []string{"pods"}, Order: 50,
		Danger: engine.DangerUI, Run: func(context.Context, *engine.Session, refParams) (any, error) { return nil, nil }}); err != nil {
		t.Fatal(err)
	}
	f.ui.Selected, f.ui.Drawer = []model.ObjectRef{ref}, &ref
	sh, _, q := newTestShell(t, f, Options{})
	q.until(t, "the drawer", func() bool { return sh.drawer.last != nil && len(findAll[*linkText](sh.drawer.details)) > 0 })

	var link *linkText
	for _, l := range findAll[*linkText](sh.drawer.details) {
		if l.text == "http: 8080/TCP" {
			link = l
		}
	}
	buttons := findAll[*widget.Button](sh.drawer.details)
	if link == nil || len(buttons) != 1 || buttons[0].Text != "Forward…" {
		t.Fatalf("the port is not a link with one button after it: link %v, buttons %d", link != nil, len(buttons))
	}
	if link.Size().Width < 50 || link.Position().X+link.Size().Width > buttons[0].Position().X {
		t.Errorf("the button overlaps the value: link ends at %v, button starts at %v", link.Position().X+link.Size().Width, buttons[0].Position().X)
	}
	test.Tap(link)
	test.Tap(buttons[0])
	if len(got) != 2 || !got[0].Link || string(got[0].Params) != string(quick) || got[1].Link {
		t.Fatalf("actions run: %+v", got)
	}

	// The row menu and the toolbar open the command's own dialog with the object's params.
	m := openMenu(t, sh, q, 0)
	item(t, m, "Forward").Action()
	if len(got) != 3 || got[2].Command != "test.forward" || len(got[2].Params) == 0 {
		t.Fatalf("row menu: %+v", got)
	}
}
