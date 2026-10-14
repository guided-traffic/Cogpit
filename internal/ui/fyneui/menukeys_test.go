package fyneui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

// ⌘F and ⌘⇧T are View menu items that show their keys (A55) while the keys stay the window's: Fyne's
// main-menu lookup never matches them (their shortcut's name is no typed key's), a click runs the
// item, and macOS's key equivalent does what the window does with the key: the focused widget that
// takes shortcuts keeps it (the editor's ⌘F), else the window's shortcut runs.
func TestWindowKeysInTheViewMenu(t *testing.T) {
	sh, w, q, f, _ := paletteShell(t, ThemeDark)
	f.mu.Lock()
	f.quietTerminals = true // the test driver's terminal race (fake_test.go): the terminals opened print nothing
	f.mu.Unlock()
	view := menuNamed(sh.mainMenu(), "View")
	search, term := menuItem(t, view, "Search"), menuItem(t, view, "New Terminal")
	cmdF := &desktop.CustomShortcut{KeyName: fyne.KeyF, Modifier: fyne.KeyModifierShortcutDefault}
	cmdShiftT := &desktop.CustomShortcut{KeyName: fyne.KeyT, Modifier: fyne.KeyModifierShortcutDefault | fyne.KeyModifierShift}
	for item, want := range map[*fyne.MenuItem]*desktop.CustomShortcut{search: cmdF, term: cmdShiftT} {
		sc, ok := item.Shortcut.(fyne.KeyboardShortcut)
		if !ok || sc.Key() != want.KeyName || sc.Mod() != want.Modifier {
			t.Errorf("%s shows %v, want %s", item.Label, item.Shortcut, want.ShortcutName())
		}
		if item.Shortcut.ShortcutName() == want.ShortcutName() {
			t.Errorf("%s's shortcut matches the typed key: the main menu would take it from the editor", item.Label)
		}
	}
	if menuItemWith(sh.mainMenu(), cmdF) != nil || menuItemWith(sh.mainMenu(), cmdShiftT) != nil {
		t.Error("a main-menu item takes ⌘F or ⌘⇧T")
	}

	// Clicks: the search takes the focus; a terminal opens.
	sh.menuKey = func() bool { return false }
	w.Canvas().Unfocus()
	search.Action()
	if w.Canvas().Focused() != sh.view.search {
		t.Errorf("Search: focus on %T", w.Canvas().Focused())
	}
	term.Action()
	q.until(t, "a terminal", func() bool { return len(f.callsOf("terminal.open")) == 1 })

	// The keys: a focused widget that takes shortcuts keeps them (the navigator's search here, the
	// editor's ⌘F alike); without one, the window's shortcut runs.
	sh.menuKey = func() bool { return true }
	sh.side.head.openSearch("")
	search.Action()
	term.Action()
	q.drain()
	if w.Canvas().Focused() != sh.side.head.entry || len(f.callsOf("terminal.open")) != 1 {
		t.Errorf("the keys in a text field: focus on %T, %d terminals", w.Canvas().Focused(), len(f.callsOf("terminal.open")))
	}
	w.Canvas().Unfocus()
	term.Action() // before ⌘F, whose search field then takes ⌘⇧T as a text field does
	search.Action()
	q.until(t, "the second terminal", func() bool { return len(f.callsOf("terminal.open")) == 2 })
	if w.Canvas().Focused() != sh.view.search {
		t.Errorf("⌘F without a focus: focus on %T", w.Canvas().Focused())
	}
}
