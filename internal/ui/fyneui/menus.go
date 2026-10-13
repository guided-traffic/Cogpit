package fyneui

import (
	"net/url"
	"runtime/debug"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// The window's Help and Window menus (ui-parity › Application menu; G13). Lens's own items are
// documented in part: Help › Welcome (with "Open Onboarding Wizard...", not a target: the welcome
// page offers its steps) and a Window menu with Minimize, on macOS Zoom, Bring All to Front and the
// list of windows, on Windows and Linux Close Window. Cogpit's, read in Fyne 2.8.1's and GLFW 3.4's
// source (not tried in a real window):
//   - Help: Welcome (ui.welcome.open), Documentation (Cogpit's README on GitHub, in the browser),
//     Copy MCP command while the window has an MCP server, About Cogpit (the build's version). An
//     item labelled exactly "About" would be moved by Fyne into macOS's application menu, taking over
//     GLFW's "About Cogpit" until the next menu rebuild drops it (menu_darwin.m), so the item is
//     "About Cogpit" and stays in Help. Documentation and About are window actions, not engine
//     commands, as the status bar's link (D-23): no agent opens a browser or a dialog for itself.
//   - Window on macOS: GLFW's own menu (cocoa_init.m, createMenuBar): Minimize ⌘M, Zoom, Bring All
//     to Front, Enter Full Screen ⌃⌘F, and AppKit's list of windows (NSApp's windowsMenu); Fyne keeps
//     it between Cogpit's menus and Help when it rebuilds the menu bar (resetDarwinMenu keeps items
//     it did not add). Cogpit adds none, which would be a second Window menu.
//   - Window on Windows and Linux: Close Window and Toggle Full Screen (Cogpit's). Minimize is
//     missing: Fyne 2.8.1's window has no minimise call and keeps GLFW's window private.
//   - Quit stays the platform's (GLFW's ⌘Q on macOS; Fyne adds Quit to the first menu elsewhere).

// goosMac is runtime.GOOS on macOS.
const goosMac = "darwin"

// helpMenu is the Help menu.
func (sh *shell) helpMenu() *fyne.Menu {
	var items []*fyne.MenuItem
	if c, ok := sh.eng.Registry().Get("ui.welcome.open"); ok {
		items = append(items, fyne.NewMenuItem(c.Label, func() { sh.dispatch(c.ID, struct{}{}, nil) }))
	}
	for _, item := range []*fyne.MenuItem{sh.auditMenuItem(), sh.diagnosticsMenuItem()} { // auditlog.go, diagnostics.go
		if item != nil {
			items = append(items, item)
		}
	}
	items = append(items, fyne.NewMenuItem("Documentation", sh.openDocumentation))
	if sh.hasMCP() {
		sh.mcpMenu = fyne.NewMenuItem("Copy MCP command", sh.copyMCPCommand)
		sh.mcpMenu.Disabled = !sh.mcp.on
		items = append(items, fyne.NewMenuItemSeparator(), sh.mcpMenu)
	}
	items = append(items, fyne.NewMenuItemSeparator(), fyne.NewMenuItem("About Cogpit", sh.showAbout))
	return fyne.NewMenu("Help", items...)
}

// windowMenu is Cogpit's Window menu on goos: nil on macOS, where GLFW's stays.
func (sh *shell) windowMenu(goos string) *fyne.Menu {
	if goos == goosMac {
		return nil
	}
	full := fyne.NewMenuItem("Toggle Full Screen", func() { sh.win.SetFullScreen(!sh.win.FullScreen()) })
	closeW := fyne.NewMenuItem("Close Window", sh.win.Close)
	return fyne.NewMenu("Window", full, fyne.NewMenuItemSeparator(), closeW)
}

// aboutInfo is what About Cogpit tells of the build.
type aboutInfo struct {
	Version  string // the packaged app's version, else the module's, else "development build"
	Revision string // the commit, shortened, "(modified)" when the tree had changes
	Time     string // the commit's time
	Go       string // the Go version it was built with
	Fyne     string // the Fyne module's version
}

// readAboutInfo takes the packaged version (fyne package --app-version) and the binary's build
// information (runtime/debug).
func readAboutInfo(packaged string, bi *debug.BuildInfo) aboutInfo {
	info := aboutInfo{Version: packaged}
	if bi == nil {
		if info.Version == "" {
			info.Version = "development build"
		}
		return info
	}
	if info.Version == "" && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		info.Version = bi.Main.Version
	}
	if info.Version == "" {
		info.Version = "development build"
	}
	info.Go = bi.GoVersion
	modified := false
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			info.Revision = s.Value
			if len(info.Revision) > 12 {
				info.Revision = info.Revision[:12]
			}
		case "vcs.time":
			info.Time = s.Value
		case "vcs.modified":
			modified = s.Value == "true"
		}
	}
	if info.Revision != "" && modified {
		info.Revision += " (modified)"
	}
	for _, d := range bi.Deps {
		if d.Path == "fyne.io/fyne/v2" {
			info.Fyne = d.Version
		}
	}
	return info
}

// lines are About's lines below the version, as "Label: value".
func (a aboutInfo) lines() []string {
	var out []string
	add := func(label, v string) {
		if v != "" {
			out = append(out, label+": "+v)
		}
	}
	add("Commit", a.Revision)
	add("Committed", a.Time)
	add("Go", a.Go)
	add("Fyne", a.Fyne)
	return append(out, "Licence: Apache License 2.0")
}

// showAbout shows About Cogpit: the logo, the version and the build, and the repository.
func (sh *shell) showAbout() {
	bi, _ := debug.ReadBuildInfo()
	info := readAboutInfo(sh.app.Metadata().Version, bi)
	title := newTextLine("Cogpit", theme.ColorNameForeground, theme.SizeNameHeadingText, fyne.TextStyle{Bold: true})
	version := newTextLine(info.Version, theme.ColorNameForeground, theme.SizeNameText, fyne.TextStyle{})
	details := newParagraph(strings.Join(info.lines(), "\n"), colorMuted, sh.app.Settings().Theme().Size(sizePickerWidth)*0.6)
	u, _ := url.Parse(readmeURL)
	repo := widget.NewHyperlink("github.com/guided-traffic/cogpit", u)
	repo.OnTapped = sh.openDocumentation
	head := container.NewHBox(newWelcomeLogo(56), newHGap(theme.SizeNameInnerPadding),
		container.NewVBox(layout.NewSpacer(), title, version, layout.NewSpacer()))
	content := container.NewVBox(head, newSpace(theme.SizeNameInnerPadding), details, repo)
	dialog.NewCustom("About Cogpit", "Close", content, sh.win).Show()
}
