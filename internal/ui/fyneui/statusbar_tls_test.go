package fyneui

import (
	"slices"
	"testing"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// uncheckedFake is a fake engine connected to "lab", whose connection does not check the server's
// certificate (engine.Connection.TLSUnchecked).
func uncheckedFake(t *testing.T) *fakeEngine {
	f := connectedFake(t, 20)
	f.ui.Context = "lab [lab.yaml]"
	f.ui.Connection = engine.Connection{State: engine.Connected, TLSUnchecked: true}
	f.diag.Context, f.diag.Demo, f.diag.DemoStats = "lab [lab.yaml]", false, nil // a real cluster's bar
	return f
}

// TestStatusBarSaysTLSNotChecked (A29, H-37): while the connection does not check the server's
// certificate, the status bar shows "TLS not checked" in the warning colour right beside the
// context, from the engine's state; it goes when the state says the check is on, and it is not
// shown for a connection under way.
func TestStatusBarSaysTLSNotChecked(t *testing.T) {
	f := uncheckedFake(t)
	sh, w, q := newTestShell(t, f, Options{})
	q.until(t, "the kinds", func() bool { return len(sh.kinds) > 0 })
	settle(w)
	s := sh.status
	texts := visibleTexts(s.root)
	i := slices.Index(texts, "TLS not checked")
	if i < 0 || s.tls.color != colorStatusWarning {
		t.Fatalf("status bar %q, item colour %v", texts, s.tls.color)
	}
	if i == 0 || texts[i-1] != "lab [lab.yaml]" { // the context; its state is the dot alone (F13)
		t.Errorf("the item is not right beside the context: %q", texts)
	}
	if ctxEnd, tlsX := s.context.Position().X+s.context.Size().Width, s.tls.Position().X; tlsX < ctxEnd || tlsX > ctxEnd+12 {
		t.Errorf("the context ends at %v, the item starts at %v", ctxEnd, tlsX)
	}

	for _, c := range []engine.Connection{
		{State: engine.Connected},                      // checked
		{State: engine.Connecting, TLSUnchecked: true}, // not connected (yet)
		{State: engine.Disconnected},
	} {
		f.update(func(u *engine.UIState) { u.Connection = c })
		q.until(t, "the state", func() bool { return sh.ui.Connection == c })
		settle(w)
		if slices.Contains(visibleTexts(s.root), "TLS not checked") {
			t.Errorf("shown for %+v", c)
		}
	}
	f.update(func(u *engine.UIState) { u.Connection = engine.Connection{State: engine.Connected, TLSUnchecked: true} })
	q.until(t, "the unchecked state again", func() bool { return sh.ui.Connection.TLSUnchecked })
	settle(w)
	if !slices.Contains(visibleTexts(s.root), "TLS not checked") {
		t.Error("not shown again")
	}
}

func TestRenderStatusBarTLSNotChecked(t *testing.T) {
	for _, tc := range []struct {
		mode ThemeMode
		name string
	}{{ThemeDark, "statusbar_tls_unchecked_dark.png"}, {ThemeLight, "statusbar_tls_unchecked_light.png"}} {
		t.Run(tc.name, func(t *testing.T) {
			f := uncheckedFake(t)
			sh, w, q := newTestShell(t, f, Options{Theme: tc.mode})
			q.until(t, "the kinds", func() bool { return len(sh.kinds) > 0 })
			savePNG(t, w, tc.name)
		})
	}
}
