package fyneui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// The top bar's right part, left to right (the owner, 2026-10-16 night): the Update button's
// place, the alerts pill, the bell, a divider, then the three layout toggles at the bar's right
// end.
func TestTopBarRightOrder(t *testing.T) {
	s := chatEngine(t)
	sh, w, _ := newEngineShell(t, s, Options{})
	settle(w)
	top := sh.top
	got := top.right.Objects
	if len(got) != 7 || got[0] != top.layout.updateBox || got[1] != top.alerts || got[2] != top.notes.bell ||
		got[6] != top.layout.root { // gaps of the inner padding around the divider
		t.Fatalf("the bar's right part holds %v", got)
	}
	div, ok := got[4].(*fyne.Container)
	if !ok || len(div.Objects) != 1 {
		t.Fatalf("no divider between the bell and the toggles: %T", got[4])
	}
	if _, ok := div.Objects[0].(*surface); !ok || div.Objects[0].Size().Height >= top.right.Size().Height {
		t.Errorf("the divider %T is %v high in a row of %v: want a short line", div.Objects[0], div.Objects[0].Size().Height, top.right.Size().Height)
	}
	_, chatEnd := absRect(top.layout.chat)
	_, rightEnd := absRect(top.right)
	near(t, "the chat toggle at the bar's right end", chatEnd.X, rightEnd.X)
	bell, _ := absRect(top.notes.bell)
	_, alertsEnd := absRect(top.alerts)
	if alertsEnd.X > bell.X {
		t.Errorf("the alerts pill ends at %v, right of the bell at %v", alertsEnd.X, bell.X)
	}
}

// The alerts pill: 0 in the bell's colour without alerts, the warning colour with warnings only,
// the error colour once one is critical; the number counts every alert; prepared without a
// source, its tooltip says so.
func TestAlertsPill(t *testing.T) {
	s := chatEngine(t)
	sh, w, _ := newEngineShell(t, s, Options{})
	settle(w)
	a := sh.top.alerts
	for _, c := range []struct {
		critical, warning int
		colour            fyne.ThemeColorName
		text              string
	}{
		{0, 0, theme.ColorNameForeground, "0"},
		{0, 3, colorStatusWarning, "3"},
		{2, 3, colorStatusError, "5"},
		{100, 0, colorStatusError, "99+"},
		{-1, -1, theme.ColorNameForeground, "0"},
	} {
		a.set(c.critical, c.warning)
		if a.colour() != c.colour || a.countText() != c.text {
			t.Errorf("%d critical, %d warnings: colour %v, text %q; want %v, %q", c.critical, c.warning, a.colour(), a.countText(), c.colour, c.text)
		}
	}
	a.set(0, 0)
	if tip := a.tipText(); tip != "Alerts: 0 critical, 0 warnings (no source of alerts yet)" {
		t.Errorf("tooltip %q", tip)
	}
	if bell := sh.top.notes.bell; a.MinSize().Height > bell.MinSize().Height {
		t.Errorf("the pill (%v) is taller than the bell (%v)", a.MinSize(), bell.MinSize())
	}
}

// The top bar's right part with the alerts pill in its three colours, for review.
func TestRenderTopBarAlertsDark(t *testing.T) {
	renderTopBarAlerts(t, ThemeDark, "topbar_alerts_dark.png")
}
func TestRenderTopBarAlertsLight(t *testing.T) {
	renderTopBarAlerts(t, ThemeLight, "topbar_alerts_light.png")
}

func renderTopBarAlerts(t *testing.T, mode ThemeMode, name string) {
	s := chatEngine(t)
	sh, w, _ := newEngineShell(t, s, Options{Theme: mode})
	sh.top.alerts.set(2, 3)
	savePNG(t, w, name)
}
