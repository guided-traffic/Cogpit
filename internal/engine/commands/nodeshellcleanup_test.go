package commands_test

import (
	"slices"
	"testing"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
)

// The requests of the cleanup of node shell pods a crash left (H-71, engine/nodeshellsweep.go) are
// among node.shell's declared verbs: the access reviews and the privilege footprint that cover the
// node shell cover its cleanup too.
func TestNodeShellCleanupVerbsAreTheNodeShells(t *testing.T) {
	r := engine.NewRegistry()
	if err := commands.RegisterAll(r); err != nil {
		t.Fatal(err)
	}
	c, ok := r.Get(engine.NodeShellCommand)
	if !ok {
		t.Fatal("no node.shell")
	}
	if len(engine.NodeShellCleanupVerbs) == 0 {
		t.Fatal("the cleanup declares no verbs")
	}
	for _, v := range engine.NodeShellCleanupVerbs {
		if v.Namespace != engine.NodeShellNamespace || !slices.Contains(c.Verbs, v) {
			t.Errorf("the cleanup's %+v is not among node.shell's %+v", v, c.Verbs)
		}
	}
}
