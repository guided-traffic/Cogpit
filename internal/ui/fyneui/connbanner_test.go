package fyneui

import (
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
)

// interruptedAt is the fixed time of the banner tests' interruption.
var interruptedAt = time.Date(2026, 10, 15, 14, 3, 0, 0, time.UTC)

// interrupt puts the fake's connection into state at interruptedAt, the next attempt 8 s after now.
func interrupt(f *fakeEngine, state engine.ConnectionState, now time.Time) {
	f.update(func(u *engine.UIState) {
		u.Connection = engine.Connection{State: state, Demo: u.Connection.Demo, Since: interruptedAt,
			Reason: "no answer from the API server"}
		if state == engine.Interrupted {
			u.Connection.NextAttempt = now.Add(8 * time.Second)
		} else {
			u.Connection.Reason = "the API server rejects the login (401 Unauthorized)"
		}
	})
}

// TestConnectionBannerFollowsTheState is S3: interrupted, the top bar's dot is amber and reads
// "Reconnecting…", the banner over the view says what is interrupted, since when and the next
// attempt, and Reconnect now dispatches cluster.reconnect; an expired login reads "Login expired"
// and offers Log in again (cluster.login); connected again, the banner goes.
func TestConnectionBannerFollowsTheState(t *testing.T) {
	f := connectedFake(t, 20)
	sh, _, q := newTestShell(t, f, Options{})
	q.until(t, "kinds", func() bool { return len(sh.kinds) > 0 })
	if sh.conn.root.Visible() {
		t.Fatal("a banner while connected")
	}
	now := interruptedAt.Add(4 * time.Minute)
	sh.now = func() time.Time { return now }
	interrupt(f, engine.Interrupted, now)
	q.until(t, "the banner", func() bool { return sh.conn.root.Visible() })
	if sh.top.state.text != "Reconnecting…" || sh.top.dot.color != statusColorName(model.StatusWarning) {
		t.Errorf("top bar %q %v", sh.top.state.text, sh.top.dot.color)
	}
	if got := sh.conn.title.text; !strings.HasPrefix(got, "The connection to demo is interrupted · since ") {
		t.Errorf("title %q", got)
	}
	if got := sh.conn.detail.text; !strings.Contains(got, "No answer from the API server") || !strings.Contains(got, "next attempt in 8 s") ||
		!strings.Contains(got, "changes are disabled") {
		t.Errorf("detail %q", got)
	}
	if !strings.Contains(sh.status.context.text, "Reconnecting…") {
		t.Errorf("status bar %q", sh.status.context.text)
	}
	if sh.view.root.Visible() == false || len(sh.kinds) == 0 {
		t.Error("the view or the navigator went with the interruption")
	}
	test.Tap(sh.conn.action)
	q.until(t, "cluster.reconnect", func() bool { return len(f.callsOf("cluster.reconnect")) == 1 })

	interrupt(f, engine.LoginExpired, now)
	q.until(t, "the login banner", func() bool { return sh.conn.action.Text == "Log in again" })
	if sh.top.state.text != "Login expired" || sh.top.dot.color != statusColorName(model.StatusError) {
		t.Errorf("top bar %q %v", sh.top.state.text, sh.top.dot.color)
	}
	if !strings.Contains(sh.conn.detail.text, "may open your browser") {
		t.Errorf("detail %q", sh.conn.detail.text)
	}
	test.Tap(sh.conn.action)
	q.until(t, "cluster.login", func() bool { return len(f.callsOf("cluster.login")) == 1 })

	f.update(func(u *engine.UIState) { u.Connection = engine.Connection{State: engine.Connected, Demo: true} })
	q.until(t, "the banner to go", func() bool { return !sh.conn.root.Visible() })
	if sh.top.state.text != "Connected" {
		t.Errorf("top bar %q", sh.top.state.text)
	}
}

// TestAShellTabOffersReconnect is S7's shells in the window: a shell tab the interruption ended
// shows a row with Reconnect, disabled until the connection is back; its click dispatches
// dock.reconnect, and a new session count opens a new session in the same tab.
func TestAShellTabOffersReconnect(t *testing.T) {
	skipTerminalUnderRace(t)
	f := connectedFake(t, 20)
	f.quietTerminals = true
	pod := f.tables["pods"].Rows[0].Ref
	f.ui.Dock = []engine.DockTab{{ID: "shell-1", Type: engine.DockShell, Title: "Shell: " + pod.Name, Context: "demo", Ref: &pod, Container: "api"}}
	f.ui.ActiveDock = "shell-1"
	sh, _, q := newTestShell(t, f, Options{})
	q.until(t, "the shell", func() bool { c := sh.dock.contents["shell-1"]; return c != nil && c.term != nil })
	c := sh.dock.contents["shell-1"]
	if c.reconnect == nil || c.reconnect.root.Visible() {
		t.Fatal("a Reconnect row on a working shell")
	}
	now := interruptedAt.Add(time.Minute)
	interrupt(f, engine.Interrupted, now)
	f.update(func(u *engine.UIState) { u.Dock[0].Interrupted = interruptedAt })
	q.until(t, "the row", func() bool { return c.reconnect.root.Visible() })
	if !c.reconnect.button.Disabled() || !strings.Contains(c.reconnect.text.text, "once the connection is back") {
		t.Errorf("while interrupted: disabled %v, %q", c.reconnect.button.Disabled(), c.reconnect.text.text)
	}
	f.update(func(u *engine.UIState) { u.Connection = engine.Connection{State: engine.Connected, Demo: true} })
	q.until(t, "Reconnect enabled", func() bool { return !c.reconnect.button.Disabled() })
	test.Tap(c.reconnect.button)
	q.until(t, "dock.reconnect", func() bool { return len(f.callsOf("dock.reconnect")) == 1 })
	f.update(func(u *engine.UIState) { u.Dock[0].Interrupted, u.Dock[0].Session = time.Time{}, 1 })
	q.until(t, "a new session", func() bool {
		f.mu.Lock()
		defer f.mu.Unlock()
		return f.opened["shell-1"] == 2
	})
	q.until(t, "the new terminal", func() bool { return c.term != nil && !c.reconnect.root.Visible() })
}

// connectionScene is the shell over the pods view with the drawer open, the connection interrupted
// (or the login expired) four minutes ago.
func connectionScene(t *testing.T, mode ThemeMode, state engine.ConnectionState) fyne.Window {
	f := connectedFake(t, 200)
	row := f.tables["pods"].Rows[3]
	ref := row.Ref
	f.details[ref.String()] = fixturePodDetail(row)
	f.ui.Selected = []model.ObjectRef{ref}
	f.ui.Drawer = &ref
	f.ui.Context = "prod-eu [config]"
	f.ui.Connection.Demo = false
	sh, w, q := newTestShell(t, f, Options{Theme: mode})
	q.until(t, "the shell to settle", func() bool { return len(sh.kinds) > 0 && sh.drawer.last != nil && sh.diag.total > 0 })
	now := interruptedAt.Add(4 * time.Minute)
	sh.now = func() time.Time { return now }
	interrupt(f, state, now)
	q.until(t, "the banner", func() bool { return sh.conn.root.Visible() })
	return w
}

func TestRenderConnectionInterruptedDark(t *testing.T) {
	savePNG(t, connectionScene(t, ThemeDark, engine.Interrupted), "connection_interrupted_dark.png")
}

func TestRenderConnectionInterruptedLight(t *testing.T) {
	savePNG(t, connectionScene(t, ThemeLight, engine.Interrupted), "connection_interrupted_light.png")
}

func TestRenderConnectionLoginExpiredDark(t *testing.T) {
	savePNG(t, connectionScene(t, ThemeDark, engine.LoginExpired), "connection_login_expired_dark.png")
}
