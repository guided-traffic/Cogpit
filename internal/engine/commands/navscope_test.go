package commands_test

import (
	"errors"
	"testing"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
)

// ui.setNavigatorScope as agents and the GUI dispatch it: up to the section and back without
// touching the connection, into another cluster through a connect; ui.state reports the scope, and
// the tab bar's index counts the scope's tabs. An agent runs it without the user's acceptance (D-2:
// a UI command; connecting only reads).
func TestNavigatorScopeCommand(t *testing.T) {
	h := newHarness(t, seed(), engine.Options{})
	agent, gui := engine.OriginExternalAgent, engine.OriginGUI
	var st commands.StateResult
	h.must(t, agent, "ui.state", nil, &st)
	if st.NavigatorScope != "cluster:test" {
		t.Fatalf("connected: scope %q", st.NavigatorScope)
	}
	h.must(t, gui, "ui.tab.open", map[string]any{"view": "services", "fixed": true}, nil)

	h.must(t, agent, "ui.setNavigatorScope", map[string]any{"scope": "section:local"}, &st)
	if st.NavigatorScope != engine.NavScopeLocal || st.Context != "test" || st.Connection.State != engine.Connected {
		t.Errorf("the section: scope %q, %s %q", st.NavigatorScope, st.Connection.State, st.Context)
	}
	h.must(t, agent, "ui.setNavigatorScope", map[string]any{"scope": "cluster:other"}, &st)
	if st.NavigatorScope != "cluster:other" || st.Context != "other" || st.Connection.State != engine.Connected {
		t.Fatalf("into other: scope %q, %s %q", st.NavigatorScope, st.Connection.State, st.Context)
	}
	// other's tabs only: the preview tab it took over; test's fixed Services tab is not counted.
	if err := h.run(t, agent, "ui.tab.select", map[string]any{"index": 2}, nil); !errors.Is(err, engine.ErrUnknownTab) {
		t.Errorf("tab 2 of other's one: %v", err)
	}
	h.must(t, gui, "ui.setNavigatorScope", map[string]any{"scope": "section:local"}, nil)
	h.must(t, agent, "ui.tab.select", map[string]any{"index": 2}, &st)
	if st.Context != "test" || st.View != "services" || st.NavigatorScope != "cluster:test" {
		t.Errorf("tab 2 at the section: %q view %q scope %q", st.Context, st.View, st.NavigatorScope)
	}

	h.must(t, agent, "ui.setNavigatorScope", map[string]any{"scope": "cluster:broken"}, &st)
	if st.NavigatorScope != engine.NavScopeLocal || st.Connection.State != engine.ConnectError {
		t.Errorf("a failed connect: scope %q, %s", st.NavigatorScope, st.Connection.State)
	}
	if err := h.run(t, agent, "ui.setNavigatorScope", map[string]any{"scope": "section:teamwork"}, nil); !errors.Is(err, engine.ErrNavScope) {
		t.Errorf("an unknown section: %v", err)
	}
}
