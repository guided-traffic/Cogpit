package commands

import (
	"context"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// An Edit tab's "Revert to original" and its undo (G9, engine/editrevert.go). Neither writes to a
// cluster.

func registerRevert(g *registrar) {
	add(g, engine.Spec[editorParams]{
		ID: "editor.revert", Label: "Revert to Original", Danger: engine.DangerUI, Verbs: cacheReads,
		Description: "Replaces an Edit tab's text with the object as Cogpit's cache holds it now and bases the edit on that " +
			"version, as a freshly opened tab is. changed says whether the text held unsaved changes; editor_undoRevert puts " +
			"them back while the text is unchanged since. Writes nothing to the cluster. A Secret's tab is the user's only (D-19).",
		Enabled: func(s *engine.Session, p editorParams) error {
			_, err := writableOn(s, tabContext(s, p.ID))
			return err
		},
		Run: func(ctx context.Context, s *engine.Session, p editorParams) (any, error) {
			return s.RevertEdit(ctx, p.ID)
		},
	})

	add(g, engine.Spec[editorParams]{
		ID: "editor.undoRevert", Label: "Undo Revert", Danger: engine.DangerUI,
		Description: "Puts back the text an Edit tab's last editor_revert replaced, based on the version it was based on, while " +
			"the tab still holds what the revert left (editor_get's canUndoRevert). Writes nothing to the cluster.",
		Enabled: func(s *engine.Session, p editorParams) error {
			_, err := connectedOn(s, tabContext(s, p.ID))
			return err
		},
		Run: func(ctx context.Context, s *engine.Session, p editorParams) (any, error) {
			return s.UndoRevert(ctx, p.ID)
		},
	})
}
