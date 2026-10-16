package commands_test

import (
	"testing"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// shownClusterOnly are the commands that read or act on a cluster and work on the cluster the window
// shows by design, with the reason: what they read is the window's.
var shownClusterOnly = map[string]string{
	"table.exportCsv": "exports the rows of a view in the active tab, as the window shows them",
}

// TestEveryClusterCommandNamesItsCluster: agents work on any connected cluster (D-69 item 6), so
// every command that reads or acts on a cluster, other than the window's own ui commands, says which
// one through its parameters (engine/target.go): a context parameter (engine.OnCluster), an
// object reference (ref, refs), a dock tab (id, tab), or all of them (all: the journals of every
// context). A command added without one would work on the shown cluster only, unseen by an agent
// that names another; it fails here until it gets one or joins shownClusterOnly with its reason.
func TestEveryClusterCommandNamesItsCluster(t *testing.T) {
	h := newHarness(t, seed(), engine.Options{})
	targets := []string{"context", engine.ParamRef, "refs", "id", "tab", "all"}
	seen := map[string]bool{}
	for _, cmd := range h.s.Registry().Commands() {
		if cmd.GUIOnly || cmd.Danger == engine.DangerUI || len(cmd.Verbs) == 0 {
			continue
		}
		if _, ok := shownClusterOnly[cmd.ID]; ok {
			seen[cmd.ID] = true
			continue
		}
		named := false
		if cmd.Schema != nil {
			for _, p := range targets {
				if _, ok := cmd.Schema.Properties[p]; ok {
					named = true
					break
				}
			}
		}
		if !named {
			t.Errorf("%s reads or acts on a cluster (verbs %v) without a context, ref, refs, dock tab or all parameter", cmd.ID, cmd.Verbs)
		}
	}
	for id := range shownClusterOnly {
		if !seen[id] {
			t.Errorf("shownClusterOnly lists %s, which is no longer a cluster command without a target", id)
		}
	}
}
