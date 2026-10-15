package fyneui

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// insecureFake lists the fixture's contexts with dev-hfi's cluster skipping the certificate check.
func insecureFake(t *testing.T) *fakeEngine {
	f := newFakeEngine(t)
	f.contexts = fixtureContexts()
	for i := range f.contexts {
		if f.contexts[i].Name == "dev-hfi" {
			f.contexts[i].Insecure = true
		}
	}
	return f
}

// welcomeRowOf returns the welcome page's row of the context whose title is title.
func welcomeRowOf(t *testing.T, sh *shell, title string) *clusterRow {
	t.Helper()
	for _, r := range sh.tabs.welcome.rows {
		if r.title.text == title {
			return r
		}
	}
	t.Fatalf("no welcome row %q", title)
	return nil
}

// hasPill reports whether the row's badges hold a pill with this text.
func hasPill(r *clusterRow, text string) bool {
	for _, o := range r.badges.Objects {
		if p, ok := o.(*pill); ok && p.text == text {
			return true
		}
	}
	return false
}

// welcomeOnFake shows the welcome page over a fake engine and waits for every context's row.
func welcomeOnFake(t *testing.T, f *fakeEngine, mode ThemeMode) (*shell, *mainQueue) {
	t.Helper()
	f.ui.Welcome = true // the engine's state without a cluster (engine/welcome.go)
	sh, _, q := newTestShell(t, f, Options{Theme: mode})
	w := welcomeShown(t, sh, q)
	q.until(t, "every row", func() bool { return len(w.rows) == len(f.contexts) })
	return sh, q
}

// A context whose kubeconfig skips the certificate check carries a warning pill and a warning line
// that names the flag and the server its credential goes to on the welcome page, and the line in
// its row's tooltip in the navigator's section (the context picker's rows had the pill and the
// line, D-57); the other rows have neither.
func TestClusterRowsMarkAnInsecureContext(t *testing.T) {
	f := insecureFake(t)
	sh, q := welcomeOnFake(t, f, ThemeDark)

	row := welcomeRowOf(t, sh, "dev-hfi")
	if !hasPill(row, "certificate not checked") {
		t.Error("the insecure context has no warning pill")
	}
	note := "insecure-skip-tls-verify: the credential goes to whoever answers at https://192.168.49.2:8443"
	if row.note == nil || row.note.color != colorStatusWarning || row.note.text != note {
		t.Errorf("the warning line: %+v", row.note)
	}
	if other := welcomeRowOf(t, sh, "kind-cogpit"); hasPill(other, "certificate not checked") || other.note != nil {
		t.Error("a checked context is marked")
	}
	n := sh.side.sec
	q.until(t, "the section's rows", func() bool { return n.loaded && n.rows["dev-hfi"] != nil })
	if tip := n.rows["dev-hfi"].tipText(); !strings.HasSuffix(tip, "\n"+note) {
		t.Errorf("the section's tooltip %q", tip)
	}
	if tip := n.rows["kind-cogpit"].tipText(); strings.Contains(tip, "insecure") {
		t.Errorf("a checked context's tooltip %q", tip)
	}
	if got := insecureNote(engine.ContextInfo{Insecure: true}); !strings.HasSuffix(got, "its server") {
		t.Errorf("without a server: %q", got)
	}
}

// The welcome page's rows of such contexts, light and dark: the "certificate not checked" pill and
// line, beside a checked one, and the pill saying an unchecked connection waits for the user.
func TestRenderClusterRowsInsecure(t *testing.T) {
	for _, tc := range []struct {
		mode ThemeMode
		name string
	}{{ThemeDark, "welcome_insecure_dark.png"}, {ThemeLight, "welcome_insecure_light.png"}} {
		t.Run(tc.name, func(t *testing.T) {
			f := insecureFake(t)
			f.contexts = append([]engine.ContextInfo{labContext}, f.contexts...)
			sh, q := welcomeOnFake(t, f, tc.mode)
			if !hasPill(welcomeRowOf(t, sh, "lab"), "unchecked connection not allowed yet") {
				t.Error("the row does not say the unchecked connection waits")
			}
			q.drain()
			savePNG(t, sh.win, tc.name)
		})
	}
}

// The connect log shows the engine's WARN line in the warning colour.
func TestRenderConnectionUpdatesInsecure(t *testing.T) {
	f := insecureFake(t)
	at := time.Date(2026, 10, 13, 2, 3, 9, 0, time.UTC)
	f.handlers["cluster.connectLog"] = func(context.Context, engine.Invocation) (any, error) {
		return map[string]any{"context": "dev-hfi", "lines": []map[string]any{
			{"time": at, "level": engine.LogInfo, "text": "Connecting to dev-hfi"},
			{"time": at.Add(time.Second), "level": engine.LogWarn, "text": "Certificate checks are off for this context " +
				"(insecure-skip-tls-verify in its kubeconfig): Cogpit cannot tell the cluster at https://192.168.49.2:8443 " +
				"from anyone else who answers there, and sends the context's credential to whoever does"},
			{"time": at.Add(time.Second), "level": engine.LogInfo, "text": "Connected to dev-hfi"},
		}}, nil
	}
	sh, w, q := newTestShell(t, f, Options{})
	q.until(t, "the section's rows", func() bool { return sh.side.sec.loaded && len(sh.side.sec.rows) == len(f.contexts) })
	sh.kc.showConnectLog("dev-hfi")
	q.until(t, "the log dialog", func() bool { return sh.kc.connectLog != nil })
	if n := len(sh.kc.connectLog.view.Lines); n != 3 {
		t.Errorf("%d lines", n)
	}
	savePNG(t, w, "connection_updates_insecure_dark.png")
}
