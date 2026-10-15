package fyneui

import (
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// The top bar's right end (F7, the owner's finding of 2026-10-15, after VS Code's title bar): an
// Update button while the app's updater holds a newer release for the user (UIState.Update), then
// three layout toggles in VS Code's order: the navigator (its primary side bar), the dock (its
// panel) and the agent chat's column (its secondary side bar). Each toggle dispatches its engine
// command (ui.toggleNavigator ⌘B, ui.toggleDock ⌘J, chat.toggle ⌘I), so agents see and drive the
// same state, and its icon shows whether its part is open. The bell stays at the bar's right end.
// The View menu has the three with their shortcuts (layoutMenuItems).

// layoutIcon is one of the toggles' icons, Cogpit's own drawing (no VS Code asset or codicon,
// D-12): a window's outline and, filled while it is open or as a line where it sits while it is
// closed, the part the toggle shows and hides. Drawn on the inline icon's 16-point grid, so that
// its one-point lines stay sharp on a 1× screen; themed: in the foreground colour, greyed when the
// button is disabled.
func layoutIcon(name, part string) fyne.Resource {
	return theme.NewThemedResource(fyne.NewStaticResource("cogpit-layout-"+name+".svg", []byte(
		`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16">`+
			`<path d="M2.5 2h11A1.5 1.5 0 0 1 15 3.5v9a1.5 1.5 0 0 1-1.5 1.5h-11A1.5 1.5 0 0 1 1 12.5v-9A1.5 1.5 0 0 1 2.5 2z`+
			`M2.5 3a.5.5 0 0 0-.5.5v9a.5.5 0 0 0 .5.5h11a.5.5 0 0 0 .5-.5v-9a.5.5 0 0 0-.5-.5z"/>`+
			`<path d="`+part+`"/></svg>`)))
}

// The toggles' icons: the navigator's column at the left, the dock's band at the bottom, the chat's
// column at the right; open filled, closed as the line of the part's inner edge.
var (
	iconNavigatorShown  = layoutIcon("navigator-shown", "M2 3h4v10H2z")
	iconNavigatorHidden = layoutIcon("navigator-hidden", "M5 3h1v10H5z")
	iconDockOpen        = layoutIcon("dock-open", "M2 9h12v4H2z")
	iconDockClosed      = layoutIcon("dock-closed", "M2 9h12v1H2z")
	iconChatShown       = layoutIcon("chat-shown", "M10 3h4v10h-4z")
	iconChatHidden      = layoutIcon("chat-hidden", "M10 3h1v10h-1z")
)

// layoutToggleParams mirrors ui.toggleNavigator's and ui.toggleDock's parameters
// (commands/layout.go).
type layoutToggleParams struct {
	Open *bool `json:"open,omitempty"`
}

// layoutBar is the Update button and the three toggles.
type layoutBar struct {
	sh                    *shell
	update                *tipButton
	updateBox             *fyne.Container // the Update button as a pill in the bar's middle (compactLayout)
	navigator, dock, chat *tipButton
	root                  *fyne.Container
}

func newLayoutBar(sh *shell) *layoutBar {
	l := &layoutBar{sh: sh}
	l.update = newTipButton(sh, nil, "", func() { sh.dispatch("app.checkForUpdates", struct{}{}, nil) })
	l.update.SetText("Update")
	l.update.Importance = widget.HighImportance
	l.updateBox = box(compactLayout{}, l.update)
	l.showUpdate(false)
	l.navigator = newTipButton(sh, iconNavigatorShown, "", func() {
		sh.dispatch("ui.toggleNavigator", layoutToggleParams{}, nil)
	})
	l.dock = newTipButton(sh, iconDockClosed, "", func() { sh.dispatch("ui.toggleDock", layoutToggleParams{}, nil) })
	// ⌘I's: the input takes the focus (chat.go). The column is made after the top bar.
	l.chat = newTipButton(sh, iconChatHidden, "", func() { sh.chat.toggle() })
	for _, b := range []struct {
		b  *tipButton
		id string
	}{{l.navigator, "ui.toggleNavigator"}, {l.dock, "ui.toggleDock"}, {l.chat, "chat.toggle"}} {
		if _, ok := sh.eng.Registry().Get(b.id); !ok {
			b.b.Hide() // a build without the command (a test fake)
		}
	}
	l.root = container.NewHBox(l.updateBox, l.navigator, l.dock, l.chat)
	return l
}

// stateChanged follows the UI state: each toggle's icon and tooltip, the dock's enabled state, and
// the Update button.
func (l *layoutBar) stateChanged() {
	ui := l.sh.ui
	set := func(b *tipButton, open bool, on, off fyne.Resource, verbOpen, verbClosed, label, id string) {
		icon, verb := off, verbClosed
		if open {
			icon, verb = on, verbOpen
		}
		if b.Icon != icon {
			b.SetIcon(icon)
		}
		b.tip = verb + " " + label + l.shortcut(id)
	}
	set(l.navigator, !ui.NavigatorHidden, iconNavigatorShown, iconNavigatorHidden, "Hide", "Show", "Navigator", "ui.toggleNavigator")
	set(l.dock, l.sh.dock.open(), iconDockOpen, iconDockClosed, "Collapse", "Open", "Dock", "ui.toggleDock")
	set(l.chat, ui.Chat, iconChatShown, iconChatHidden, "Hide", "Show", "Agent Chat", "chat.toggle")
	if len(ui.Dock) == 0 {
		l.dock.tip = "Dock: no tabs to show" + l.shortcut("ui.toggleDock")
		l.dock.Disable()
	} else {
		l.dock.Enable()
	}
	if !l.sh.chat.available() {
		l.chat.Hide() // a window without the agent's chat
	}
	l.updateChanged(ui.Update)
}

// updateChanged shows the Update button while the updater holds a newer release, its version in
// the tooltip.
func (l *layoutBar) updateChanged(u engine.UpdateState) {
	l.update.tip = "A newer Cogpit is available: show it in the update window"
	if u.Version != "" {
		l.update.tip = "Cogpit " + u.Version + " is available: show it in the update window"
	}
	l.showUpdate(u.Available)
}

// showUpdate shows or hides the Update button with its place, so that a hidden one leaves no gap.
func (l *layoutBar) showUpdate(on bool) {
	if on == l.updateBox.Visible() && on == l.update.Visible() {
		return
	}
	if on {
		l.update.Show()
		l.updateBox.Show()
	} else {
		l.update.Hide()
		l.updateBox.Hide()
	}
}

// compactLayout gives its object its least width and the height of a line of bold text with the
// theme's padding above and below, centred: the Update button as a pill, not a block as tall as
// the bar (VS Code's is compact too).
type compactLayout struct{}

func (compactLayout) height() float32 {
	th := fyne.CurrentApp().Settings().Theme()
	return fyne.MeasureText("Mg", th.Size(theme.SizeNameText), fyne.TextStyle{Bold: true}).Height + 2*th.Size(theme.SizeNamePadding)
}

func (l compactLayout) MinSize(objs []fyne.CanvasObject) fyne.Size {
	var w float32
	for _, o := range objs {
		if o.Visible() {
			w = max(w, o.MinSize().Width)
		}
	}
	return fyne.NewSize(w, l.height())
}

func (l compactLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	h := min(size.Height, l.height())
	for _, o := range objs {
		o.Resize(fyne.NewSize(size.Width, h))
		o.Move(fyne.NewPos(0, (size.Height-h)/2))
	}
}

// shortcut is " (⌘B)" for a command's shortcut, "" without one.
func (l *layoutBar) shortcut(id string) string {
	if c, ok := l.sh.eng.Registry().Get(id); ok {
		if sc := shortcutText(c.Shortcut, runtime.GOOS == goosMac); sc != "" {
			return " (" + sc + ")"
		}
	}
	return ""
}

// layoutChanged follows the layout's state: the navigator shown or hidden, the dock open or
// collapsed, and the top bar's toggles. applyUIState runs it after the dock followed its tabs.
func (sh *shell) layoutChanged(old engine.UIState, first bool) {
	st := sh.ui
	if first || st.NavigatorHidden != old.NavigatorHidden {
		sh.showNavigator(!st.NavigatorHidden)
	}
	if !first && st.DockOpen != old.DockOpen {
		sh.layoutBody()
		sh.dock.rebuildStrip() // the active tab's mark and the strip's arrow
	}
	sh.top.layout.stateChanged()
}

// showNavigator shows or hides the navigator's column with its edge: the views take its room
// (panelLayout), the hotbar stays. Hidden, it gives the keyboard back if it had it.
func (sh *shell) showNavigator(on bool) {
	panel, handle, _ := sh.sideLay.parts(sh.mainRow.Objects)
	if on {
		panel.Show()
		handle.Show()
	} else {
		if sh.navigatorFocused() {
			sh.win.Canvas().Unfocus()
		}
		panel.Hide()
		handle.Hide()
	}
	sh.mainRow.Refresh()
}

// navigatorFocused reports whether the keyboard is in the navigator: its tree, its section level
// or its search.
func (sh *shell) navigatorFocused() bool {
	f := sh.win.Canvas().Focused()
	if f == nil {
		return false
	}
	s := sh.side
	return f == fyne.Focusable(s.tree) || f == fyne.Focusable(s.sec) || (s.head != nil && f == fyne.Focusable(s.head.entry))
}

// layoutMenuChanged reports whether the View menu's layout items must follow the state: their
// checkmarks, and the Dock item's enabled state with the dock's tabs.
func layoutMenuChanged(st, old engine.UIState) bool {
	return st.Chat != old.Chat || st.NavigatorHidden != old.NavigatorHidden || st.DockOpen != old.DockOpen ||
		(len(st.Dock) == 0) != (len(old.Dock) == 0)
}

// layoutMenuItems are the View menu's Navigator (⌘B) and Dock (⌘J), checked while their part is
// open, before Agent Chat (⌘I, chatMenuItems); main-menu items, so their shortcuts work whatever
// has the focus. None in a build without the commands.
func (sh *shell) layoutMenuItems() []*fyne.MenuItem {
	var items []*fyne.MenuItem
	if c, ok := sh.eng.Registry().Get("ui.toggleNavigator"); ok {
		item := sh.withShortcut(fyne.NewMenuItem(c.Label, func() {
			sh.dispatch(c.ID, layoutToggleParams{}, nil)
		}), c.ID)
		item.Checked = !sh.ui.NavigatorHidden
		items = append(items, item)
	}
	if c, ok := sh.eng.Registry().Get("ui.toggleDock"); ok {
		item := sh.withShortcut(fyne.NewMenuItem(c.Label, func() {
			sh.dispatch(c.ID, layoutToggleParams{}, nil)
		}), c.ID)
		item.Checked = sh.dock.open()
		item.Disabled = len(sh.ui.Dock) == 0
		items = append(items, item)
	}
	return items
}
