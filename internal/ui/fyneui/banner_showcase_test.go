package fyneui

import (
	"image"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"

	logoart "github.com/guided-traffic/cogpit/internal/ui/logo"
)

// The README's banner (A111; showcase_test.go's pictures): Cogpit's logo, its name and one
// sentence, on the theme's background, in the dark and the light theme like every other picture.
// It is a render rather than an SVG: an SVG's text would be drawn in whatever font the reader's
// browser has, the render draws it in the app's own.
const (
	bannerWidth   = 1280 // points; the PNG is showcaseScale times as large
	bannerHeight  = 320
	bannerLogo    = 200 // the logo's side
	bannerGap     = 48  // between the logo and the text
	bannerName    = 112 // the name's text size
	bannerLine    = 34  // the sentence's text size
	bannerLeading = 4   // between the name and the sentence
	// bannerRise lifts the text: its boxes hold room above the name's capitals and below the
	// sentence's descenders, so that centring the boxes left the glyphs 17 points below the logo's
	// centre (measured on the render, 2026-10-15).
	bannerRise     = 17
	bannerSentence = "A fast, native Kubernetes IDE for your desktop."
)

// 0. The banner at the README's top.
func TestShowcaseBanner(t *testing.T) {
	eachTheme(t, func(t *testing.T, mode ThemeMode) {
		a := test.NewTempApp(t)
		th := newTheme(mode)
		a.Settings().SetTheme(th)
		w := test.NewTempWindow(t, showcaseBanner(th))
		w.SetPadded(false)
		size := fyne.NewSize(bannerWidth, bannerHeight)
		w.Resize(size)
		saveShowcaseSized(t, w, "banner", mode, size)
	})
}

// showcaseBanner lays the banner out: the logo and the two lines of text beside it, centred as a
// group. The logo is the welcome page's drawing (the app icon's line weights without its plate, in
// the theme's logo inks), rasterised at the size the capture shows it at; the name takes the
// logo's frame ink, the sentence the theme's text colour.
func showcaseBanner(th fyne.Theme) fyne.CanvasObject {
	bg := canvas.NewRectangle(themeColor(th, theme.ColorNameBackground))
	bg.Resize(fyne.NewSize(bannerWidth, bannerHeight))

	px := bannerLogo * showcaseScale
	img := image.NewNRGBA(image.Rect(0, 0, px, px))
	logoart.Draw(img, logoart.Icon(nil), logoart.Colors{Frame: themeColor(th, colorLogo), Accent: themeColor(th, colorLogoAccent)},
		float64(px)/logoart.Box, 0, 0)
	logo := canvas.NewImageFromImage(img)
	logo.FillMode, logo.ScaleMode = canvas.ImageFillContain, canvas.ImageScaleSmooth

	name := canvas.NewText("Cogpit", themeColor(th, colorLogo))
	name.TextSize, name.TextStyle = bannerName, fyne.TextStyle{Bold: true}
	line := canvas.NewText(bannerSentence, themeColor(th, theme.ColorNameForeground))
	line.TextSize = bannerLine

	ns, ls := name.MinSize(), line.MinSize()
	group := bannerLogo + bannerGap + max(ns.Width, ls.Width)
	x := (bannerWidth - group) / 2
	logo.Move(fyne.NewPos(x, (bannerHeight-bannerLogo)/2))
	logo.Resize(fyne.NewSize(bannerLogo, bannerLogo))
	tx := x + bannerLogo + bannerGap
	y := (bannerHeight-(ns.Height+bannerLeading+ls.Height))/2 - bannerRise
	name.Move(fyne.NewPos(tx, y))
	name.Resize(ns)
	line.Move(fyne.NewPos(tx, y+ns.Height+bannerLeading))
	line.Resize(ls)
	return container.NewWithoutLayout(bg, logo, name, line)
}
