package fyneui

import (
	"image/color"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"

	"github.com/guided-traffic/cogpit/internal/model"
)

var allStatuses = []model.Status{
	model.StatusOK, model.StatusSucceeded, model.StatusPending, model.StatusWarning,
	model.StatusError, model.StatusTerminating, model.StatusUnknown,
}

func TestThemeDefinesEveryColour(t *testing.T) {
	test.NewTempApp(t)
	for _, mode := range []ThemeMode{"", ThemeDark, ThemeLight, ThemeSystem} {
		test.AssertAllColorNamesDefined(t, newTheme(mode), string(mode))
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
