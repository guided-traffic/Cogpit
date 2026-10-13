package engine

import (
	"context"
	"errors"
	"fmt"

	"github.com/guided-traffic/cogpit/internal/model"
)

// "Revert to original" of an Edit tab (G9; Lens's diff view's button, a paid feature there, not
// captured): the tab's text becomes the object as Cogpit's cache holds it now, and the tab is
// based on that version, as a freshly opened tab would be. Which original Lens means (the object as
// the tab opened it, or as the cluster holds it now) is not captured: Cogpit takes the cluster's,
// so that a revert never brings back a version the cluster has moved past (inferred). The text it
// replaced is kept, one step deep, so that editor.undoRevert puts it back with the version it was
// based on, until the text changes again.

// editUndo is what editor.undoRevert restores.
type editUndo struct {
	text                     string
	baseRV, baseUID, baseTxt string
	// afterRev and afterBaseRV are the text revision and base the revert left: an undo applies
	// only while the tab still holds them.
	afterRev    uint64
	afterBaseRV string
}

// RevertResult is what editor.revert and editor.undoRevert return.
type RevertResult struct {
	Ref model.ObjectRef `json:"ref"`
	// Changed is true when the text changed: it held unsaved changes (revert), or the reverted
	// text came back (undo).
	Changed             bool   `json:"changed"`
	Text                string `json:"yaml"`
	Revision            uint64 `json:"revision"`
	BaseResourceVersion string `json:"baseResourceVersion"`
	// CanUndo is true when editor.undoRevert would bring the replaced text back.
	CanUndo bool `json:"canUndo,omitempty"`
}

// ErrNothingToUndo is returned by UndoRevert when the tab's text changed since the revert, or
// nothing was reverted.
var ErrNothingToUndo = errors.New("nothing to undo: no revert, or the text changed since")

// canUndo reports whether the editor holds what its last revert left.
func (e *editor) canUndo() bool {
	return e.undo != nil && e.undo.afterRev == e.rev && e.undo.afterBaseRV == e.baseRV
}

// RevertEdit replaces an Edit tab's text with the object as the cache holds it now and bases the
// tab on that version. A Secret's tab: the GUI only (D-19).
func (s *Session) RevertEdit(ctx context.Context, id string) (RevertResult, error) {
	if err := s.GuardSecretEditor(ctx, id, ""); err != nil {
		return RevertResult{}, err
	}
	e, b, _, err := s.editTarget(id)
	if err != nil {
		return RevertResult{}, err
	}
	obj, err := cachedObject(ctx, b, e.target)
	if errors.Is(err, ErrNotFound) {
		return RevertResult{}, fmt.Errorf("%s no longer exists on the cluster; there is nothing to revert to: %w", e.target.Ref.String(), err)
	}
	if err != nil {
		return RevertResult{}, err
	}
	obj = withTypeMeta(obj, e.target)
	text, err := EditableYAML(obj)
	if err != nil {
		return RevertResult{}, err
	}
	rv, uid := obj.GetResourceVersion(), string(obj.GetUID())
	changed := false
	after, err := s.updateEditor(id, func(x *editor) {
		switch {
		case x.text != text: // the replaced text can come back
			x.undo = &editUndo{text: x.text, baseRV: x.baseRV, baseUID: x.baseUID, baseTxt: x.baseText}
			x.text = text
			x.rev++
			changed = true
		case !x.canUndo(): // nothing replaced; a second revert of an unchanged text keeps the first's undo
			x.undo = nil
		}
		x.baseRV, x.baseUID, x.baseText = rv, uid, text
		if x.undo != nil {
			x.undo.afterRev, x.undo.afterBaseRV = x.rev, x.baseRV
		}
	})
	if err != nil {
		return RevertResult{}, err
	}
	return RevertResult{Ref: e.target.Ref, Changed: changed, Text: after.text, Revision: after.rev,
		BaseResourceVersion: after.baseRV, CanUndo: after.canUndo()}, nil
}

// UndoRevert puts back the text an Edit tab's last revert replaced, based on the version it was
// based on, while the tab still holds what the revert left.
func (s *Session) UndoRevert(ctx context.Context, id string) (RevertResult, error) {
	if err := s.GuardSecretEditor(ctx, id, ""); err != nil {
		return RevertResult{}, err
	}
	e, err := s.editorOf(id)
	if err != nil {
		return RevertResult{}, err
	}
	if e.tab.Type != DockEdit {
		return RevertResult{}, fmt.Errorf("dock tab %q is a %s tab, not an Edit tab", id, e.tab.Type)
	}
	ok := false
	after, err := s.updateEditor(id, func(x *editor) {
		if !x.canUndo() {
			return
		}
		u := x.undo
		x.undo = nil
		x.text, x.baseRV, x.baseUID, x.baseText = u.text, u.baseRV, u.baseUID, u.baseTxt
		x.rev++
		ok = true
	})
	if err != nil {
		return RevertResult{}, err
	}
	if !ok {
		return RevertResult{}, ErrNothingToUndo
	}
	return RevertResult{Ref: e.target.Ref, Changed: true, Text: after.text, Revision: after.rev, BaseResourceVersion: after.baseRV}, nil
}
