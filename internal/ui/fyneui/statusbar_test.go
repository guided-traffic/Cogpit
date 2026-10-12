package fyneui

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
)

// statusSizes lays the status bar out at width and returns the link's and the items' rectangles.
func statusSizes(sh *shell, width float32) (link, left fyne.Position, linkW float32, linkShown bool) {
	s := sh.status
	s.root.Resize(fyne.NewSize(width, s.root.MinSize().Height))
	return s.link.Position(), s.layout.left.Position(), s.link.Size().Width, s.link.Visible()
}

// TestStatusBarOwnerLink: D-23. The status bar's left end shows "Need a Kubernetes Expert? Visit:
// hans-fischer.com", also before a connection; a click opens https://hans-fischer.com in the
// browser, and nothing opens it before.
func TestStatusBarOwnerLink(t *testing.T) {
	for name, connected := range map[string]bool{"before connecting": false, "connected": true} {
		t.Run(name, func(t *testing.T) {
			f := connectedFake(t, 20)
			if !connected {
				f.ui.Connection = engine.Connection{State: engine.Disconnected}
				f.ui.Context = ""
			}
			browser := recordBrowser(t)
			sh, w, q := newTestShell(t, f, Options{})
			q.until(t, "the kinds", func() bool { return len(sh.kinds) > 0 || !connected })
			settle(w)

			l := sh.status.link
			if !l.Visible() || l.text != "Need a Kubernetes Expert? Visit: hans-fischer.com" {
				t.Fatalf("link visible %v, text %q", l.Visible(), l.text)
			}
			if p := l.Position(); p.X != 0 {
				t.Errorf("the link is at x=%v, want the left end", p.X)
			}
			if len(browser.opened()) != 0 {
				t.Fatalf("the browser was opened before a click: %v", browser.opened())
			}
			test.Tap(l)
			if got := browser.opened(); len(got) != 1 || got[0] != "https://hans-fischer.com" {
				t.Fatalf("opened %v, want https://hans-fischer.com once", got)
			}
		})
	}
}

// TestStatusBarOwnerLinkIsNoCommand: the link is a GUI link, not an engine command (D-23): no
// command of the registry opens a URL, so no agent can.
func TestStatusBarOwnerLinkIsNoCommand(t *testing.T) {
	r := engine.NewRegistry()
	if err := commands.RegisterAll(r); err != nil {
		t.Fatal(err)
	}
	for _, c := range r.Commands() {
		text := strings.ToLower(c.ID + " " + c.Label + " " + c.Description)
		if strings.Contains(text, "hans-fischer") || strings.Contains(text, "openurl") {
			t.Errorf("command %q mentions the owner's page: %q", c.ID, text)
		}
	}
}

// TestStatusBarOwnerLinkGivesWayFirst: when the bar is too narrow the link is shortened and then
// hidden, and the status items keep their widths and their room, at the same place right of where
// the link would end.
func TestStatusBarOwnerLinkGivesWayFirst(t *testing.T) {
	f := connectedFake(t, 20)
	sh, w, q := newTestShell(t, f, Options{MCPCommand: "claude mcp add cogpit -- cogpit-cli mcp"})
	q.until(t, "the diagnostics", func() bool { return sh.diag.total > 0 })
	settle(w)

	s := sh.status
	th := fyne.CurrentApp().Settings().Theme()
	items := s.layout.left.MinSize().Width + s.layout.right.MinSize().Width
	full := fyne.MeasureText(ownerLinkText, th.Size(theme.SizeNameCaptionText), fyne.TextStyle{}).Width
	gaps := 2*th.Size(theme.SizeNameInnerPadding) + 1
	pad := 2 * th.Size(theme.SizeNameInnerPadding) // the bar's own inset

	// Wide: the whole link, then the items.
	_, left, linkW, shown := statusSizes(sh, items+pad+full+gaps+40)
	if !shown || linkW < full {
		t.Fatalf("wide: link shown %v, %v wide, its text needs %v", shown, linkW, full)
	}
	if left.X < linkW {
		t.Errorf("wide: the items start at %v, inside the link (%v wide)", left.X, linkW)
	}
	itemsW := s.layout.left.Size().Width
	if itemsW != s.layout.left.MinSize().Width {
		t.Errorf("wide: the items are %v wide, want their minimum %v", itemsW, s.layout.left.MinSize().Width)
	}

	// Half the link's room: shortened, the items unharmed.
	_, left, linkW, shown = statusSizes(sh, items+pad+gaps+full/2)
	if !shown || linkW >= full || linkW <= 0 {
		t.Errorf("narrow: link shown %v, %v wide, want shortened below %v", shown, linkW, full)
	}
	if got := s.layout.left.Size().Width; got != itemsW {
		t.Errorf("narrow: the items are %v wide, want %v as when wide", got, itemsW)
	}
	if left.X < linkW {
		t.Errorf("narrow: the items start at %v, inside the link (%v wide)", left.X, linkW)
	}
	if got := s.link.Size().Width; fyne.MeasureText(shortened(s), th.Size(theme.SizeNameCaptionText), fyne.TextStyle{}).Width > got {
		t.Errorf("narrow: the link's text overflows its %v", got)
	}

	// No room beyond the items: the link and its divider are hidden, the items keep everything.
	_, left, _, shown = statusSizes(sh, items+pad)
	if shown || s.layout.sep.Visible() {
		t.Errorf("no room: link shown %v, divider shown %v", shown, s.layout.sep.Visible())
	}
	if left.X != 0 || s.layout.left.Size().Width != itemsW {
		t.Errorf("no room: items at x=%v, %v wide; want 0 and %v", left.X, s.layout.left.Size().Width, itemsW)
	}
	if got := s.layout.right.Position().X + s.layout.right.Size().Width; got != items {
		t.Errorf("no room: the right items end at %v, want the bar's inner width %v", got, items)
	}

	// The bar's minimum width is the items' alone: the link never asks for room.
	if got := s.root.MinSize().Width; got != items+pad {
		t.Errorf("minimum width %v, want the items' %v", got, items+pad)
	}

	// Widening the bar brings the link back.
	_, _, _, shown = statusSizes(sh, items+pad+full+gaps+40)
	if !shown {
		t.Error("the link did not come back when the bar widened")
	}
}

// shortened is the text the link shows now (the renderer's ellipsized text).
func shortened(s *statusBar) string {
	th := fyne.CurrentApp().Settings().Theme()
	return ellipsize(s.link.text, s.link.Size().Width, th.Size(theme.SizeNameCaptionText), fyne.TextStyle{})
}
