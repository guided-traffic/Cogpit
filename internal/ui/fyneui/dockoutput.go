package fyneui

import (
	"context"
	"fmt"
	"time"

	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/ui/fyneui/termview"
)

// What the dock's logs and terminal tabs show, for agents (G17, D-44; engine/dockoutput.go): the
// window is the engine's DockOutputReader, which pages a tab's lines on the main goroutine, where
// the widgets hold them. A terminal tab's strip offers the switch that lets agents read it,
// "Agents may read", off when a tab opens; only this window sets it (dock.allowAgentRead is
// GUI-only), and it follows the engine's state of the active tab (DockTab.AgentRead).

// agentReadLabel is the switch's label in the dock's strip.
const agentReadLabel = "Agents may read"

// dockOutputReader is the window's engine.DockOutputReader.
type dockOutputReader struct{ sh *shell }

// ReadDockOutput pages what the tab shows on the main goroutine and waits for it, or for ctx.
func (r dockOutputReader) ReadDockOutput(ctx context.Context, tab engine.DockTab, q engine.DockOutputQuery) (engine.DockOutput, error) {
	done := make(chan engine.DockOutput, 1)
	r.sh.do(func() { done <- r.sh.dock.output(tab, q) })
	select {
	case out := <-done:
		return out, nil
	case <-ctx.Done():
		return engine.DockOutput{}, fmt.Errorf("the window did not answer in time: %w", ctx.Err())
	}
}

// output pages what a tab shows now. A tab the window has not opened yet, or whose stream or
// session has not started, is opening, without lines. Main goroutine.
func (d *dock) output(tab engine.DockTab, q engine.DockOutputQuery) engine.DockOutput {
	out := engine.DockOutput{Tab: tab, DockPage: engine.DockPage{Lines: []string{}}}
	c, ok := d.contents[tab.ID]
	switch {
	case !ok || c.closed:
		out.Opening = true
	case c.logs != nil && c.logs.view != nil:
		v := c.logs.view
		src := v.Output()
		out.DockPage = engine.PageDockLines(src, q)
		out.Shown, out.Held, out.Dropped = src.Len(), v.HeldLines(), v.DroppedLines()
		out.Status, out.Ended = v.StatusText(), v.Ended()
	case c.term != nil:
		src := termview.ScreenOutput(c.term.ScreenLines())
		out.DockPage = engine.PageDockLines(src, q)
		out.Shown = src.Len()
		select {
		case <-c.term.Done():
			out.Ended = true
		default:
		}
	default:
		out.Opening = true
	}
	return out
}

// newAgentReadSwitch makes the strip's switch; it shows while the active tab is a terminal tab.
// Turning it on is refused for the confirmations' arm delay after the switch began to show another
// tab: an agent can make another terminal tab the active one (dock_activate, a UI command) just
// before the user's click lands, which would let agents read a tab the user did not mean.
func (d *dock) newAgentReadSwitch() *widget.Check {
	c := widget.NewCheck(agentReadLabel, func(on bool) {
		if d.syncingAgentRead {
			return
		}
		if on && time.Since(d.agentReadSince) < d.sh.confirmer.armDelay {
			d.updateAgentRead() // the click was meant for the tab shown before
			return
		}
		id := d.active
		d.sh.dispatch("dock.allowAgentRead", allowAgentReadParams{ID: id, Allowed: on}, func(_ any, err error) {
			d.sh.showError(err)
			// What the engine holds: the state its answer brought (its event comes first), else
			// the state before the click, on an error or when a later change already applied.
			d.updateAgentRead()
		})
	})
	c.Hide()
	return c
}

// allowAgentReadParams mirror dock.allowAgentRead's parameters.
type allowAgentReadParams struct {
	ID      string `json:"id"`
	Allowed bool   `json:"allowed"`
}

// updateAgentRead shows the switch for the active tab, if it is a terminal tab, as the engine holds
// it. Main goroutine.
func (d *dock) updateAgentRead() {
	var active *engine.DockTab
	for i := range d.tabs {
		if d.tabs[i].ID == d.active {
			active = &d.tabs[i]
		}
	}
	if active == nil || !engine.IsTerminalTab(active.Type) || !d.open() {
		d.agentRead.Hide()
		return
	}
	if active.ID != d.agentReadTab {
		d.agentReadTab, d.agentReadSince = active.ID, time.Now()
	}
	d.syncingAgentRead = true
	d.agentRead.SetChecked(active.AgentRead)
	d.syncingAgentRead = false
	d.agentRead.Show()
}
