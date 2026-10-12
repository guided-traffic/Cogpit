package fyneui

import (
	"fmt"
	"image/color"
	"math"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"

	"github.com/guided-traffic/cogpit/internal/model"
)

var allStatuses = []model.Status{
	model.StatusOK, model.StatusSucceeded, model.StatusPending, model.StatusWarning,
	model.StatusError, model.StatusTerminating, model.StatusUnknown, model.StatusStopped,
}

func TestThemeDefinesEveryColour(t *testing.T) {
	test.NewTempApp(t)
	for _, mode := range []ThemeMode{"", ThemeDark, ThemeLight, ThemeSystem} {
		test.AssertAllColorNamesDefined(t, newTheme(mode), string(mode))
	}
}

// TestForeignThemeWithoutCogpitColours: under a theme that lacks Cogpit's own colour names (Fyne's
// test theme returns nil for them), themeColor answers with Cogpit's palette, and a tinted pill
// renders instead of panicking in withAlpha. A test's cleanup crashed this way: a stopped shell's
// queued UI state was applied after Fyne's NewTempApp cleanup had installed a fresh test app.
func TestForeignThemeWithoutCogpitColours(t *testing.T) {
	a := test.NewTempApp(t)
	a.Settings().SetTheme(test.Theme())
	th, v := a.Settings().Theme(), variant()
	if th.Color(colorChip, v) != nil {
		t.Fatalf("precondition: the test theme defines %s", colorChip)
	}
	for _, n := range []fyne.ThemeColorName{colorChip, colorStatusWarning, colorMuted} {
		if got, want := themeColor(th, n), paletteFor(v).fyne[n]; got == nil || got != want {
			t.Errorf("%s: %v, want Cogpit's %v", n, got, want)
		}
	}
	if got, want := themeColor(th, "cogpit.no-such-colour"), th.Color(theme.ColorNameForeground, v); got != want {
		t.Errorf("an unknown name: %v, want the theme's foreground %v", got, want)
	}
	if got := withAlpha(nil, 0x2c); got != color.Transparent {
		t.Errorf("withAlpha(nil) = %v, want transparent", got)
	}

	p := newPill("Cluster role backup-operator not found", colorStatusWarning, true)
	w := test.NewTempWindow(t, p)
	w.Resize(fyne.NewSize(300, 40))
	p.Refresh()
	r, ok := test.WidgetRenderer(p).(*pillRenderer)
	if !ok {
		t.Fatalf("pill renderer %T", test.WidgetRenderer(p))
	}
	if want := withAlpha(paletteFor(v).fyne[colorStatusWarning], 0x2c); r.bg.FillColor != want {
		t.Errorf("tinted pill background %v, want %v", r.bg.FillColor, want)
	}
}

func TestStatusColoursAreDistinct(t *testing.T) {
	for _, v := range []fyne.ThemeVariant{theme.VariantDark, theme.VariantLight} {
		seen := map[color.Color]model.Status{}
		for _, s := range allStatuses {
			c := StatusColor(s, v)
			if other, dup := seen[c]; dup {
				t.Errorf("variant %d: %s and %s share colour %v", v, s, other, c)
			}
			seen[c] = s
			if got := newTheme(modeFor(v)).Color(statusColorName(s), v); got != c {
				t.Errorf("variant %d: theme colour of %s is %v, StatusColor says %v", v, s, got, c)
			}
		}
		if fg := StatusColor(model.StatusNone, v); fg != paletteFor(v).fyne[theme.ColorNameForeground] {
			t.Errorf("variant %d: StatusNone is %v, want the foreground", v, fg)
		}
	}
}

func modeFor(v fyne.ThemeVariant) ThemeMode {
	if v == theme.VariantLight {
		return ThemeLight
	}
	return ThemeDark
}

func TestThemeModeIgnoresSystemVariantUnlessSystem(t *testing.T) {
	dark := newTheme("") // the zero mode is dark, whatever the system says
	if got, want := dark.Color(theme.ColorNameBackground, theme.VariantLight), darkPalette.fyne[theme.ColorNameBackground]; got != want {
		t.Errorf("default mode on a light system: background %v, want dark %v", got, want)
	}
	sys := newTheme(ThemeSystem)
	if got, want := sys.Color(theme.ColorNameBackground, theme.VariantLight), lightPalette.fyne[theme.ColorNameBackground]; got != want {
		t.Errorf("system mode on a light system: background %v, want light %v", got, want)
	}
}

func TestPerfStats(t *testing.T) {
	samples := make([]perfSample, 0, 100)
	for i := range 100 {
		samples = append(samples, perfSample{at: time.Duration(i) * perfFrame, late: time.Duration(i+1) * time.Millisecond / 2}) // 0.5 … 50 ms
	}
	s := perfStats(samples, time.Now().Add(-2*time.Second))
	if s.Samples != 100 || s.MaxMs != 50 || s.P50Ms != 25.5 || s.P99Ms != 50 {
		t.Errorf("stats %+v", s)
	}
	if s.Over16Ms != 68 { // 16.5 ms … 50 ms
		t.Errorf("over 16 ms: %d, want 68", s.Over16Ms)
	}
	// One run of late samples is one stall, reported at its first sample.
	if len(s.Stalls) != 1 || s.Stalls[0].Ms != 16.5 {
		t.Errorf("stalls %+v, want one starting with the 16.5 ms sample", s.Stalls)
	}
	if s.DurationS < 2 {
		t.Errorf("duration %v", s.DurationS)
	}
	if empty := perfStats(nil, time.Time{}); empty.Samples != 0 || empty.MaxMs != 0 {
		t.Errorf("empty stats %+v", empty)
	}
}

func TestFormatCountAndEllipsize(t *testing.T) {
	for n, want := range map[int64]string{0: "0", 999: "999", 1000: "1,000", 10000: "10,000", 1234567: "1,234,567", -4200: "-4,200"} {
		if got := formatCount(n); got != want {
			t.Errorf("formatCount(%d) = %q, want %q", n, got, want)
		}
	}
	test.NewTempApp(t)
	long := "checkout-api-7d9f8b6c5-x2k4p"
	w := fyne.MeasureText(long, 13, fyne.TextStyle{}).Width
	if got := ellipsize(long, w, 13, fyne.TextStyle{}); got != long {
		t.Errorf("text that fits was shortened to %q", got)
	}
	short := ellipsize(long, w/2, 13, fyne.TextStyle{})
	if len(short) >= len(long) || short[len(short)-len("…"):] != "…" || fyne.MeasureText(short, 13, fyne.TextStyle{}).Width > w/2 {
		t.Errorf("ellipsize to half width gave %q", short)
	}
}

// lum and contrastRatio are WCAG 2's relative luminance and contrast ratio.
func lum(c color.Color) float64 {
	n := color.NRGBAModel.Convert(c).(color.NRGBA)
	lin := func(b uint8) float64 {
		v := float64(b) / 255
		if v <= 0.03928 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(n.R) + 0.7152*lin(n.G) + 0.0722*lin(n.B)
}

func contrastRatio(a, b color.Color) float64 {
	la, lb := lum(a), lum(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// TestPaletteContrast: D-22 copies Solarized's colours, which are low in contrast by design; the
// tokens that carry text stay readable on the surfaces they are drawn on (WCAG: 4.5:1 for body
// text, 3:1 for de-emphasised text and for status colours, 2.5:1 for a status colour on a selected
// row, which is transient).
func TestPaletteContrast(t *testing.T) {
	for name, p := range map[string]*palette{"dark": &darkPalette, "light": &lightPalette} {
		c := func(n fyne.ThemeColorName) color.Color { return p.fyne[n] }
		// Minimum contrast of the foreground per surface: Solarized's own floor is about 4.4:1 on its
		// panels, so the large panels ask 4.3 and the small, transient ones 3.5.
		surfaces := map[fyne.ThemeColorName]float64{
			theme.ColorNameBackground: 4.5, colorSidebar: 4.3, colorDrawer: 4.3, colorStatusBar: 4.3,
			colorTopBar: 4.3, theme.ColorNameMenuBackground: 4.3,
			theme.ColorNameHeaderBackground: 3.5, colorRowHover: 3.5, theme.ColorNameInputBackground: 3.5,
			colorToast: 3.5, colorTooltip: 3.5,
		}
		check := func(what string, fg color.Color, bg fyne.ThemeColorName, floor float64) {
			t.Helper()
			if got := contrastRatio(fg, c(bg)); got < floor {
				t.Errorf("%s: %s on %s is %.2f:1, want %.1f:1", name, what, bg, got, floor)
			}
		}
		for bg, floor := range surfaces {
			check("foreground", c(theme.ColorNameForeground), bg, floor)
			check("muted", c(colorMuted), bg, 2.9)
			check("link", c(colorLink), bg, 3.0)
			for s := range p.status {
				if s == model.StatusUnknown || s == model.StatusStopped {
					continue // grey on purpose: a stopped container's square, an unknown state
				}
				check(fmt.Sprintf("status %s", s), p.status[s], bg, 3.0)
			}
		}
		for s, col := range p.status {
			check(fmt.Sprintf("status %s", s), col, colorRowSelected, 2.5)
		}
		check("selected row text", c(colorRowSelectedText), colorRowSelected, 4.5)
		check("running container", c(colorContainerRunning), colorRowSelected, 2.5)
		check("running container", c(colorContainerRunning), theme.ColorNameBackground, 3.0)
		check("active tab", c(colorDockTabText), colorDockTab, 4.5)
		check("inactive tab", c(colorDockTabDim), colorDock, 3.5)
		check("text on primary", c(theme.ColorNameForegroundOnPrimary), theme.ColorNamePrimary, 4.4)
		check("text on error", c(theme.ColorNameForegroundOnError), theme.ColorNameError, 3.5)
		check("text on success", c(theme.ColorNameForegroundOnSuccess), theme.ColorNameSuccess, 3.5)
		check("text on warning", c(theme.ColorNameForegroundOnWarning), theme.ColorNameWarning, 3.5)
		check("placeholder", c(theme.ColorNamePlaceHolder), theme.ColorNameInputBackground, 3.0)
	}
}
