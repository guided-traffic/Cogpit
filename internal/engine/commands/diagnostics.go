package commands

import (
	"context"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// The Diagnostics view (D-13; engine/diagnostics_view.go): the connected cluster's API requests by
// verb and resource, its watches and the objects per kind, from the counters the connection and the
// cache keep. Neither command sends a request.

func registerDiagnostics(g *registrar) {
	add(g, engine.Spec[struct{}]{
		ID: "diagnostics.get", Label: "Read Diagnostics", Danger: engine.DangerRead,
		Description: "Returns what Cogpit asked of the connected cluster and holds of it, as the Diagnostics tab shows it: " +
			"the API requests since connecting, by verb and by verb and resource (list, watch, get, ...; watches " +
			"included), each watch (informer) with its scope, state (open: a view or lookup holds it; warm: kept a while " +
			"after its last user; covered: served by another watch), objects, events and errors, the objects per kind, " +
			"and the table snapshots built. From Cogpit's own counters: it sends no request.",
		Run: func(_ context.Context, s *engine.Session, _ struct{}) (any, error) {
			return s.DiagnosticsReport(), nil
		},
	})

	add(g, engine.Spec[struct{}]{
		ID: "diagnostics.open", Label: "Diagnostics", MenuPath: "Help/Diagnostics", Danger: engine.DangerUI,
		Description: "Opens the Diagnostics dock tab, or activates it: the connected cluster's API requests by verb and " +
			"resource, its watches and the objects per kind, kept current while it is open, as Help › Diagnostics does. " +
			"Returns the tab.",
		Run: func(_ context.Context, s *engine.Session, _ struct{}) (any, error) {
			tab := engine.DockTab{ID: s.NewDockID(engine.DockDiagnostics), Type: engine.DockDiagnostics, Title: "Diagnostics"}
			created := true
			s.UpdateUI(func(u *engine.UIState) {
				for _, t := range u.Dock { // one Diagnostics tab
					if t.Type == engine.DockDiagnostics {
						tab, created = t, false
						break
					}
				}
				if created {
					u.Dock = append(u.Dock, tab)
				}
				u.ActiveDock = tab.ID
			})
			return DockResult{Tab: tab, Created: created}, nil
		},
	})
}
