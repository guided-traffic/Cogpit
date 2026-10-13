package fyneui

import (
	"fmt"

	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// "Revert to original" in an Edit tab (G9): a button in the editor's footer and in the review a
// Save shows, which puts the object as the cluster holds it now into the editor (editor.revert)
// and bases the edit on that version. When the revert replaced unsaved text, Undo Revert beside it
// brings the text back (editor.undoRevert) until the user types or saves: the revert asks nothing
// first and can be undone instead. Lens's diff view has the button (a paid feature there, not
// captured); its look, its place in the editor's footer and the undo are Cogpit's (inferred).

func (e *editorTab) newRevertButtons() {
	e.revertBtn = widget.NewButtonWithIcon("Revert to Original", theme.ContentUndoIcon(), e.revert)
	e.revertBtn.Importance = widget.LowImportance
	e.reviewRevert = widget.NewButtonWithIcon("Revert to Original", theme.ContentUndoIcon(), e.revert)
	e.reviewRevert.Importance = widget.LowImportance
	e.undoBtn = widget.NewButtonWithIcon("Undo Revert", theme.ContentRedoIcon(), e.undoRevert)
	e.undoBtn.Hide()
	e.pane.onEdit = func() { e.showUndo(false) }
}

// showUndo shows or hides Undo Revert.
func (e *editorTab) showUndo(on bool) {
	if on {
		e.undoBtn.Show()
	} else {
		e.undoBtn.Hide()
	}
}

// revert sends what the user typed, then reverts the tab to the object as the cache holds it.
func (e *editorTab) revert() {
	if e.busy {
		return
	}
	e.setBusy(true)
	e.setStatus("Reverting to "+e.tab.Title+" as the cluster holds it…", colorMuted)
	e.pane.flush(func() {
		e.sh.dispatch("editor.revert", dockParams{ID: e.tab.ID}, func(res any, err error) {
			e.setBusy(false)
			var rr engine.RevertResult
			if err == nil {
				err = decode(res, &rr)
			}
			if err != nil {
				e.setStatus("Not reverted: "+errText(err), colorStatusError)
				return
			}
			e.pane.replace(rr.Text, rr.Revision)
			if e.reviewing {
				e.backToEditor()
			}
			if rr.Changed {
				e.setStatus(fmt.Sprintf("Reverted to %s · version %s, as the cluster holds it. Undo Revert brings your text back.", e.tab.Title, rr.BaseResourceVersion), colorMuted)
			} else {
				e.setStatus(fmt.Sprintf("Nothing to revert: the text is %s · version %s, as the cluster holds it.", e.tab.Title, rr.BaseResourceVersion), colorMuted)
			}
			e.showUndo(rr.CanUndo)
		})
	})
}

// undoRevert puts the text the last revert replaced back.
func (e *editorTab) undoRevert() {
	if e.busy {
		return
	}
	e.setBusy(true)
	e.sh.dispatch("editor.undoRevert", dockParams{ID: e.tab.ID}, func(res any, err error) {
		e.setBusy(false)
		e.showUndo(false)
		var rr engine.RevertResult
		if err == nil {
			err = decode(res, &rr)
		}
		if err != nil {
			e.setStatus("Not undone: "+errText(err), colorStatusError)
			return
		}
		e.pane.replace(rr.Text, rr.Revision)
		e.setStatus(fmt.Sprintf("Your text is back, based on version %s.", rr.BaseResourceVersion), colorMuted)
	})
}

// flush sends what the user typed since the last send, then calls then; with nothing to send it
// calls then at once.
func (p *editorPane) flush(then func()) {
	if p.closed {
		return
	}
	if !p.dirty {
		then()
		return
	}
	if p.timer != nil {
		p.timer.Stop()
	}
	p.dirty = false
	p.sh.dispatch("editor.setText", editorTextParams{ID: p.id, YAML: p.view.Text()}, func(res any, err error) {
		var tr struct {
			Revision uint64 `json:"revision"`
		}
		if err == nil && decode(res, &tr) == nil && tr.Revision > p.pushed {
			p.pushed = tr.Revision
		}
		if !p.closed {
			then()
		}
	})
}
