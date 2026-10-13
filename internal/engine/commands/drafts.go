package commands

import (
	"context"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
)

// Drafts of unsaved editor text (G9), kept for a crash or a quit (D-11, engine/drafts.go): listed,
// restored and discarded. None of them writes to a cluster.

type draftsParams struct {
	All bool `json:"all,omitempty" jsonschema:"list the drafts of every context, not only the connected one's"`
}

type draftParams struct {
	ID string `json:"id" jsonschema:"draft id as editor_drafts lists it: 32 hex digits"`
}

// keepsDrafts is the Enabled check of the draft commands.
func keepsDrafts(s *engine.Session) error {
	if s.Drafts() == nil {
		return engine.ErrNoDrafts
	}
	return nil
}

func registerDrafts(g *registrar) {
	add(g, engine.Spec[draftsParams]{
		ID: "editor.drafts", Label: "Unsaved Editor Drafts", Danger: engine.DangerRead,
		Description: "Lists the drafts of unsaved editor text Cogpit kept from Edit and Create resource tabs that a crash or a quit " +
			"ended (the connected context's; all: every context's), oldest first: the tab's type, title, object and the " +
			"version its text is based on, when it was kept, its size; never the text. editor_restoreDraft opens the tab " +
			"with the text again (editor_get then reads it), editor_discardDraft deletes it. Cogpit keeps no draft of a " +
			"Secret or of Helm values.",
		Enabled: func(s *engine.Session, _ draftsParams) error { return keepsDrafts(s) },
		Run: func(_ context.Context, s *engine.Session, p draftsParams) (any, error) {
			return s.EditorDrafts(p.All)
		},
	})

	add(g, engine.Spec[draftParams]{
		ID: "editor.restoreDraft", Label: "Restore Draft", Danger: engine.DangerUI, Verbs: cacheReads,
		Description: "Opens the tab of a draft from editor_drafts again with its text, on the draft's context, which must be " +
			"connected: a Create resource tab, or an Edit tab on the object based on the version the draft was based on, so " +
			"that an apply after a change on the cluster is refused with a conflict first. An Edit tab already open on the " +
			"object takes the draft unless it has unsaved changes of its own. The tab keeps the draft from then on. Writes " +
			"nothing to the cluster. Returns the tab's state as editor_get does.",
		Enabled: func(s *engine.Session, _ draftParams) error {
			if err := keepsDrafts(s); err != nil {
				return err
			}
			_, err := writable(s)
			return err
		},
		Run: func(ctx context.Context, s *engine.Session, p draftParams) (any, error) {
			st, err := s.RestoreDraft(ctx, p.ID)
			if err == nil && engine.OriginOf(ctx) != engine.OriginGUI && s.EditorHoldsSecret(st.Tab.ID) {
				st.Text = model.Withheld // never kept for a Secret; withheld as editor.get would (D-19)
			}
			return st, err
		},
	})

	add(g, engine.Spec[draftParams]{
		ID: "editor.discardDraft", Label: "Discard Draft", Danger: engine.DangerUI,
		Description: "Deletes a draft from editor_drafts: its unsaved text is gone for good. A draft of a tab open in this or " +
			"another Cogpit window is refused; closing that tab discards it.",
		Enabled: func(s *engine.Session, _ draftParams) error { return keepsDrafts(s) },
		Run: func(_ context.Context, s *engine.Session, p draftParams) (any, error) {
			return nil, s.DiscardDraft(p.ID)
		},
	})
}
