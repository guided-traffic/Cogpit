package fyneui

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/terminal"
	"github.com/guided-traffic/cogpit/internal/ui/fyneui/termview"
)

// attachEngine is the fake engine with what an attach session to a REPL shows.
type attachEngine struct{ *fakeEngine }

func (a attachEngine) OpenTerminal(ctx context.Context, tabID string) (terminal.Session, error) {
	if !strings.HasPrefix(tabID, "attach-") {
		return a.fakeEngine.OpenTerminal(ctx, tabID)
	}
	a.mu.Lock()
	a.opened[tabID]++
	a.mu.Unlock()
	out := "[cogpit] attached to orders-repl-0/repl; if no prompt shows, press Enter\r\n" +
		"orders-repl v2.3.1 (python 3.12.4)\r\n>>> orders.count(status=\"open\")\r\n1284\r\n>>> "
	pr, pw := io.Pipe()
	go func() { _, _ = io.WriteString(pw, out) }() // then the session stays open, as an attached REPL does
	return &openStream{fakeStream: fakeStream{Reader: pr, f: a.fakeEngine, id: tabID, done: make(chan struct{})}, pw: pw}, nil
}

// openStream is a session that keeps running until it is closed.
type openStream struct {
	fakeStream
	pw *io.PipeWriter
}

func (s *openStream) Close() error {
	_ = s.pw.Close()
	return s.fakeStream.Close()
}

// attachScene is the pods view with an attach tab open in the dock.
func attachScene(t *testing.T, mode ThemeMode) (*shell, fyne.Window, *mainQueue, *fakeEngine) {
	t.Helper()
	f := connectedFake(t, 200)
	ref := f.tables["pods"].Rows[3].Ref
	f.ui.Dock = []engine.DockTab{
		{ID: "logs-1", Type: engine.DockLogs, Title: "Logs: " + ref.Name + "/api", Context: "demo", Ref: &ref, Container: "api"},
		{ID: "attach-2", Type: engine.DockAttach, Title: "Attach: orders-repl-0/repl", Context: "demo", Ref: &ref, Container: "repl"},
	}
	f.ui.ActiveDock = "attach-2"

	a := test.NewTempApp(t)
	a.Settings().SetTheme(newTheme(mode))
	w := test.NewTempWindow(t, nil)
	w.SetPadded(false)
	q := &mainQueue{}
	sh := newShell(attachEngine{f}, a, w, Options{Theme: mode}, q.do)
	sh.now = func() time.Time { return fixedNow }
	f.SetConfirmer(sh.confirmer)
	w.SetContent(sh.root)
	w.Resize(fyne.NewSize(1440, 900))
	sh.start()
	t.Cleanup(func() {
		sh.stop()
		q.drain()
	})
	q.until(t, "the attach session", func() bool {
		f.mu.Lock()
		defer f.mu.Unlock()
		return len(sh.kinds) > 0 && f.opened["attach-2"] == 1
	})
	return sh, w, q, f
}

// TestAttachTabShowsATerminal: an attach tab is a terminal on Session.OpenTerminal, with the
// attachment icon, and closing it closes the session.
func TestAttachTabShowsATerminal(t *testing.T) {
	sh, _, q, f := attachScene(t, ThemeDark)
	c := sh.dock.contents["attach-2"]
	var term *termview.Terminal
	q.until(t, "the terminal widget", func() bool {
		if len(c.obj.Objects) == 1 {
			term, _ = c.obj.Objects[0].(*termview.Terminal)
		}
		return term != nil
	})
	q.until(t, "the REPL's prompt", func() bool { return strings.Contains(term.Text(), ">>> orders.count") })
	if !strings.Contains(term.Text(), "[cogpit] attached to orders-repl-0/repl") {
		t.Errorf("terminal %q", term.Text())
	}
	tabs := sh.dock.tabRow.Objects
	if b := tabs[1].(*tabButton); b.title != "Attach: orders-repl-0/repl" || b.icon.Name() != "mail-attachment.svg" && !strings.Contains(b.icon.Name(), "attachment") {
		t.Errorf("tab %q icon %q", b.title, b.icon.Name())
	}
	go func() {
		_, _ = f.Dispatch(context.Background(), engine.Invocation{ID: "dock.close", Origin: engine.OriginGUI, Params: mustJSON(t, dockParams{ID: "attach-2"})})
	}()
	q.until(t, "the session to close", func() bool {
		f.mu.Lock()
		defer f.mu.Unlock()
		return f.closed["attach-2"] >= 1 // the widget closes it, and its reader once more
	})
}

func TestRenderAttachDark(t *testing.T) {
	sh, w, q, _ := attachScene(t, ThemeDark)
	waitAttachPrompt(t, sh, q)
	savePNG(t, w, "attach_dark.png")
}

func TestRenderAttachLight(t *testing.T) {
	sh, w, q, _ := attachScene(t, ThemeLight)
	waitAttachPrompt(t, sh, q)
	savePNG(t, w, "attach_light.png")
}

func waitAttachPrompt(t *testing.T, sh *shell, q *mainQueue) {
	t.Helper()
	q.until(t, "the REPL's prompt", func() bool {
		c := sh.dock.contents["attach-2"]
		if len(c.obj.Objects) != 1 {
			return false
		}
		term, ok := c.obj.Objects[0].(*termview.Terminal)
		return ok && strings.Contains(term.Text(), ">>>")
	})
}
