package yamlview

import (
	"image/color"
	"time"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// The logic tests drive the widget without painting it, through a minimal fake app: Fyne's default
// theme pinned to one variant, an in-memory clipboard, and text measured as a fixed monospace grid.
// It exists because every offscreen painter Fyne exports (fyne.io/fyne/v2/test and
// fyne.io/fyne/v2/driver/software) imports github.com/anthonynsimon/bild, and the test package also
// github.com/stretchr/testify, neither of which go.mod lists. The tests that paint, and the S4
// benchmarks, use fyne.io/fyne/v2/test behind the fynetest build tag (render_test.go,
// bench_test.go).

// newLogicView returns a view of the given size under a fresh fake app with the dark theme.
func newLogicView(w, h float32) (*View, *fakeApp) {
	app := newFakeApp(theme.VariantDark)
	fyne.SetCurrentApp(app)
	v := New()
	v.Resize(fyne.NewSize(w, h))
	return v, app
}

// forcedVariant pins a theme to one variant whatever the settings say.
type forcedVariant struct {
	fyne.Theme
	variant fyne.ThemeVariant
}

func (f forcedVariant) Color(n fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	return f.Theme.Color(n, f.variant)
}

type fakeApp struct {
	fyne.App // methods the fake does not provide are nil and panic if Fyne ever calls them

	drv      *fakeDriver
	settings *fakeSettings
	clip     *fakeClipboard
}

func newFakeApp(variant fyne.ThemeVariant) *fakeApp {
	return &fakeApp{
		drv:      &fakeDriver{},
		settings: &fakeSettings{th: forcedVariant{Theme: theme.DefaultTheme(), variant: variant}, variant: variant},
		clip:     &fakeClipboard{},
	}
}

func (a *fakeApp) Driver() fyne.Driver       { return a.drv }
func (a *fakeApp) Settings() fyne.Settings   { return a.settings }
func (a *fakeApp) Clipboard() fyne.Clipboard { return a.clip }

type fakeSettings struct {
	fyne.Settings

	th      fyne.Theme
	variant fyne.ThemeVariant
}

func (s *fakeSettings) Theme() fyne.Theme               { return s.th }
func (s *fakeSettings) SetTheme(th fyne.Theme)          { s.th = th }
func (s *fakeSettings) ThemeVariant() fyne.ThemeVariant { return s.variant }
func (s *fakeSettings) Scale() float32                  { return 1 }
func (s *fakeSettings) PrimaryColor() string            { return theme.ColorBlue }

type fakeClipboard struct{ content string }

func (c *fakeClipboard) Content() string     { return c.content }
func (c *fakeClipboard) SetContent(s string) { c.content = s }

// fakeDriver has no canvas: canvas.Refresh finds none and does nothing.
type fakeDriver struct {
	fyne.Driver
}

func (d *fakeDriver) CanvasForObject(fyne.CanvasObject) fyne.Canvas { return nil }

func (d *fakeDriver) DoubleTapDelay() time.Duration { return 300 * time.Millisecond }

// RenderedTextSize measures every rune as 0.6 em wide and lines as 1.25 em high, close to Fyne's
// monospace font; the logic tests only need the grid to be consistent.
func (d *fakeDriver) RenderedTextSize(text string, size float32, _ fyne.TextStyle, _ fyne.Resource) (fyne.Size, float32) {
	return fyne.NewSize(float32(utf8.RuneCountInString(text))*size*0.6, size*1.25), size
}
