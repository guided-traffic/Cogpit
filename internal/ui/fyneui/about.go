package fyneui

import (
	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/buildinfo"
)

// About Cogpit (decision A60, the owner's finding F2): Cogpit's own dialog, from the application
// menu on macOS (Cogpit › About Cogpit, appmenu_darwin.go) and from Help elsewhere (menus.go). It
// shows the logo, the name, the version the build reports (internal/buildinfo: the release's, or
// "Development build" and its commit), what Cogpit is in one line, the licence, and links to the
// repository and the licence. The version is the binary's own, never Fyne's app metadata: Cogpit
// packages a binary fyne did not build, so Fyne's metadata keeps its default "0.0.1" in every build
// (F2's cause). A window action, not an engine command, as the status bar's link (D-23): no agent
// opens a dialog or a browser for itself.

const (
	aboutLogoSize = 64 // the logo's side, in points
	aboutTagline  = "A native Kubernetes IDE"
	aboutLicence  = "Apache License 2.0"
	repoURL       = "https://github.com/guided-traffic/cogpit"
	licenceURL    = "https://github.com/guided-traffic/cogpit/blob/main/LICENSE"
)

// aboutVersion returns the dialog's version line and, for a development build, the line below it
// that names the commit.
func aboutVersion(info buildinfo.Info) (version, detail string) {
	if info.Release {
		return "Version " + info.Version, ""
	}
	if info.Commit == "" {
		return "Development build", ""
	}
	detail = "Commit " + info.Commit
	if info.Modified {
		detail += " (modified)"
	}
	return "Development build", detail
}

// showAbout shows About Cogpit for this binary's build.
func (sh *shell) showAbout() { sh.showAboutFor(buildinfo.Read()) }

// showAboutFor shows About Cogpit for a build. No title: the name is the dialog's heading.
func (sh *shell) showAboutFor(info buildinfo.Info) {
	dialog.NewCustom("", "Close", sh.aboutContent(info), sh.win).Show()
}

// aboutContent is the dialog's column, every line centred.
func (sh *shell) aboutContent(info buildinfo.Info) fyne.CanvasObject {
	line := func(text string, c fyne.ThemeColorName, size fyne.ThemeSizeName, style fyne.TextStyle) fyne.CanvasObject {
		t := newTextLine(text, c, size, style)
		t.align = fyne.TextAlignCenter
		return t
	}
	version, detail := aboutVersion(info)
	objs := []fyne.CanvasObject{
		container.NewCenter(newWelcomeLogo(aboutLogoSize)),
		newSpace(theme.SizeNameInnerPadding),
		line("Cogpit", theme.ColorNameForeground, theme.SizeNameHeadingText, fyne.TextStyle{Bold: true}),
		newSpace(theme.SizeNameLineSpacing),
		line(version, theme.ColorNameForeground, theme.SizeNameText, fyne.TextStyle{}),
	}
	if detail != "" {
		objs = append(objs, line(detail, colorMuted, theme.SizeNameCaptionText, fyne.TextStyle{Monospace: true}))
	}
	objs = append(objs,
		newSpace(theme.SizeNameInnerPadding),
		line(aboutTagline, colorMuted, theme.SizeNameText, fyne.TextStyle{}),
		newSpace(theme.SizeNameInnerPadding),
		line(aboutLicence, colorMuted, theme.SizeNameCaptionText, fyne.TextStyle{}),
		container.NewCenter(container.NewHBox(sh.aboutLink("Repository", repoURL), sh.aboutLink("Licence", licenceURL))),
		newSpace(theme.SizeNamePadding), // room above the dialog's Close
	)
	return sized(sizeAboutWidth, "", container.NewVBox(objs...))
}

// aboutLink is a link of the dialog: a click opens it in the user's browser, nothing else does.
func (sh *shell) aboutLink(text, target string) *widget.Hyperlink {
	u, _ := url.Parse(target)
	l := widget.NewHyperlink(text, u)
	l.OnTapped = func() {
		if err := openBrowser(sh.app, u); err != nil {
			sh.toast("error", "Could not open "+target+": "+err.Error())
		}
	}
	return l
}
