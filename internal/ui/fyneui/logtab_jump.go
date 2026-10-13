package fyneui

import (
	"time"

	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// Jumping to a time in a logs tab (ui-parity › Logs, "Jump to a time and date"): a text field
// beside the selectors (Fyne has no date picker; the field and its forms are Cogpit's, inferred)
// takes a time in the user's zone and format, as the tab's timestamps show (D-31: "13.10.2026
// 01:20"), or "10m ago"; Enter dispatches logs.jumpTo, which reads it in the engine, and the tab
// follows the jump in the engine's state, whoever asked: it scrolls to the first line written at
// or after that time among the lines it holds, and when it holds none that old for a container it
// shows, it loads them first (OlderLogs with since).

// jumpState is what the tab knows about jumps.
type jumpState struct {
	seen    int       // the engine's jump this tab has carried out
	pending bool      // a jump that came before the stream opened
	at      time.Time // its time
}

// newJumpField returns the field a time to jump to is typed into.
func (lt *logTab) newJumpField() *widget.Entry {
	e := widget.NewEntry()
	// An example in the user's format, written as typed (no zone to convert from).
	example, tf := time.Date(2026, 10, 13, 1, 20, 0, 0, time.UTC), lt.sh.ui.TimeFormat.In(time.UTC)
	e.SetPlaceHolder("Jump to time: " + tf.FormatDate(example) + " " + tf.FormatClock(example) + ", 10m ago")
	e.SetIcon(theme.HistoryIcon())
	e.OnSubmitted = func(text string) {
		lt.sh.dispatch("logs.jumpTo", logJumpParams{ID: lt.tab.ID, Time: text}, func(_ any, err error) { lt.sh.showError(err) })
	}
	return e
}

// logJumpParams mirror logs.jumpTo's parameters.
type logJumpParams struct {
	ID   string `json:"id"`
	Time string `json:"time"`
}

// followJump carries out the jump the engine's state shows if the tab has not.
func (lt *logTab) followJump(t engine.DockTab) {
	if t.Logs.Jump.Seq == lt.jump.seen {
		return
	}
	lt.jump.seen = t.Logs.Jump.Seq
	at := t.Logs.Jump.Time
	lt.jumpField.SetText(lt.sh.timeFormat().Format(at)) // as the timestamps show it, and read back so
	lt.jumpTo(at)
}

// jumpTo scrolls to the first line written at or after at, loading older lines first when the
// tab holds none that old of a container it shows and does not hold whole.
func (lt *logTab) jumpTo(at time.Time) {
	if lt.failed {
		return
	}
	v := lt.view
	if v == nil {
		lt.jump.pending, lt.jump.at = true, at
		return
	}
	held := v.Held()
	older := len(held) == 0
	for _, h := range held {
		older = older || h.Oldest.After(at) && !lt.older.whole[h.Source]
	}
	if _, ok := lt.sh.eng.(olderLogsEngine); !ok || !older || lt.older.busy {
		lt.finishJump(at)
		return
	}
	lt.loadOlder(at, func() { lt.finishJump(at) })
}

// finishJump scrolls to the line, or says there is none.
func (lt *logTab) finishJump(at time.Time) {
	if lt.view.JumpTo(at) {
		return
	}
	lt.sh.toast("info", "No line written at or after "+lt.sh.timeFormat().Format(at)+" in this tab")
}
