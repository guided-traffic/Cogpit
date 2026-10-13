package commands_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/model"
)

// TestSetFilterOptions: ui.setFilter sets the search field's toggles beside its text, a toggle left
// out stays, the view's rows follow them, ui.state shows them and an invalid pattern's reason, and
// each tab keeps its own while a view change clears the text only.
func TestSetFilterOptions(t *testing.T) {
	h := newHarness(t, seed(), engine.Options{})
	agent, gui := engine.OriginExternalAgent, engine.OriginGUI
	rows := func(want ...string) {
		t.Helper()
		u := h.s.UIState()
		snap := waitTable(t, h.s, "pods", func(s *engine.TableSnapshot) bool {
			return s.Synced && s.Filter == u.Filter && s.FilterOptions == u.FilterOptions
		})
		var got []string
		for _, r := range snap.Rows {
			got = append(got, r.Ref.Key())
		}
		if !slices.Equal(got, want) {
			t.Errorf("filter %q %+v: rows %q, want %q", u.Filter, u.FilterOptions, got, want)
		}
	}

	h.must(t, agent, "ui.setFilter", map[string]any{"filter": "^infra$", "regex": true}, nil)
	if u := h.s.UIState(); u.FilterOptions != (engine.FilterOptions{Regex: true}) {
		t.Fatalf("options %+v", u.FilterOptions)
	}
	rows("infra/web-0", "infra/web-1")
	h.must(t, agent, "ui.setFilter", map[string]any{"filter": "^SHOP$"}, nil) // regex stays on
	rows("shop/web-0", "shop/web-1")
	h.must(t, agent, "ui.setFilter", map[string]any{"filter": "^SHOP$", "matchCase": true}, nil)
	rows()
	h.must(t, agent, "ui.setFilter", map[string]any{"filter": "WEB-1 infra", "regex": false}, nil) // terms, case kept
	rows()
	h.must(t, agent, "ui.setFilter", map[string]any{"filter": "web-1 infra", "matchCase": false}, nil)
	rows("infra/web-1")

	// An invalid pattern filters nothing out; ui.state says why.
	h.must(t, agent, "ui.setFilter", map[string]any{"filter": "web-(", "regex": true}, nil)
	rows("infra/web-0", "shop/web-0", "infra/web-1", "shop/web-1")
	var st commands.StateResult
	h.must(t, agent, "ui.state", map[string]any{}, &st)
	if !st.FilterOptions.Regex || st.FilterOptions.MatchCase || !strings.Contains(st.FilterError, "missing closing )") || st.Rows != 4 {
		t.Errorf("ui.state: options %+v, error %q, rows %d", st.FilterOptions, st.FilterError, st.Rows)
	}
	var raw map[string]any
	h.must(t, agent, "ui.state", map[string]any{}, &raw)
	if b, _ := json.Marshal(raw["filterOptions"]); string(b) != `{"matchCase":false,"regex":true}` {
		t.Errorf("ui_state's filterOptions %s", b)
	}
	if err := h.run(t, agent, "ui.setFilter", map[string]any{"regex": "yes"}, nil); err == nil {
		t.Error("a toggle that is not a boolean was taken")
	}

	// Select all takes the rows the toggles show.
	h.must(t, agent, "ui.setFilter", map[string]any{"filter": "^web-1$"}, nil)
	h.must(t, agent, "ui.select", map[string]any{"all": true}, nil)
	if u := h.s.UIState(); len(u.Selected) != 2 || u.Selected[0].Name != "web-1" || u.Selected[1].Name != "web-1" {
		t.Errorf("select all with a pattern: %v", u.Selected)
	}

	// A view change clears the text and keeps the toggles; a fixed tab keeps its own.
	h.must(t, gui, "ui.openView", map[string]any{"view": "services"}, nil)
	if u := h.s.UIState(); u.Filter != "" || u.FilterOptions != (engine.FilterOptions{Regex: true}) {
		t.Errorf("after openView: filter %q options %+v", u.Filter, u.FilterOptions)
	}
	h.must(t, agent, "ui.tab.open", map[string]any{"view": "pods", "fixed": true}, nil)
	h.must(t, gui, "ui.setFilter", map[string]any{"filter": "web", "regex": false, "matchCase": true}, nil)
	h.must(t, agent, "ui.tab.select", map[string]any{"index": 1}, nil)
	if u := h.s.UIState(); u.View != "services" || u.FilterOptions != (engine.FilterOptions{Regex: true}) {
		t.Errorf("back in the first tab: view %q options %+v", u.View, u.FilterOptions)
	}
	h.must(t, agent, "ui.tab.select", map[string]any{"index": 2}, nil)
	u := h.s.UIState()
	if u.View != "pods" || u.Filter != "web" || u.FilterOptions != (engine.FilterOptions{MatchCase: true}) {
		t.Errorf("the fixed tab: view %q filter %q options %+v", u.View, u.Filter, u.FilterOptions)
	}
	if i := slices.IndexFunc(u.Tabs, func(t engine.ViewTab) bool { return t.ID == u.ActiveTab }); i < 0 || u.Tabs[i].FilterOptions != u.FilterOptions {
		t.Errorf("the tab's own options %+v", u.Tabs)
	}
}

// TestNavigateClearsAHidingPattern: ui.navigate clears a search whose pattern hides the object and
// keeps one that shows it, toggles applied.
func TestNavigateClearsAHidingPattern(t *testing.T) {
	h := newHarness(t, seed(), engine.Options{})
	gui := engine.OriginGUI
	ref := model.ObjectRef{Kind: "pods", Namespace: "shop", Name: "web-0"}
	h.must(t, gui, "ui.setFilter", map[string]any{"filter": "^WEB-0$", "regex": true}, nil)
	h.must(t, gui, "ui.navigate", map[string]any{"ref": ref}, nil)
	if u := h.s.UIState(); u.Filter != "^WEB-0$" {
		t.Errorf("a pattern that shows the object was cleared: %q", u.Filter)
	}
	h.must(t, gui, "ui.setFilter", map[string]any{"filter": "^WEB-0$", "matchCase": true}, nil)
	h.must(t, gui, "ui.navigate", map[string]any{"ref": ref}, nil)
	if u := h.s.UIState(); u.Filter != "" || u.FilterOptions != (engine.FilterOptions{Regex: true, MatchCase: true}) {
		t.Errorf("a pattern that hides the object: filter %q options %+v", u.Filter, u.FilterOptions)
	}
}
