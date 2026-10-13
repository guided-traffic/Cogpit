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

// pickerRowOf returns the picker's row of the context whose title is title.
func pickerRowOf(t *testing.T, sh *shell, title string) *pickerRow {
	t.Helper()
	for _, o := range sh.picker.list.Objects {
		if r, ok := o.(*pickerRow); ok && r.title.text == title {
			return r
		}
	}
	t.Fatalf("no picker row %q", title)
	return nil
}

// hasPill reports whether the row's badges hold a pill with this text.
func hasPill(r *pickerRow, text string) bool {
	for _, o := range r.badges.Objects {
		if p, ok := o.(*pill); ok && p.text == text {
			return true
		}
	}
	return false
}

// A context whose kubeconfig skips the certificate check carries a warning pill and a warning line
// that names the flag and the server its credential goes to; the other rows have neither.
func TestPickerMarksAnInsecureContext(t *testing.T) {
	f := insecureFake(t)
	sh, _, q := newTestShell(t, f, Options{})
	q.until(t, "the picker rows", func() bool { return sh.picker.visible() && len(sh.picker.list.Objects) == len(f.contexts) })

	row := pickerRowOf(t, sh, "dev-hfi")
	if !hasPill(row, "certificate not checked") {
		t.Error("the insecure context has no warning pill")
	}
	if row.note == nil || row.note.color != colorStatusWarning ||
		row.note.text != "insecure-skip-tls-verify: the credential goes to whoever answers at https://192.168.49.2:8443" {
		t.Errorf("the warning line: %+v", row.note)
	}
	if other := pickerRowOf(t, sh, "kind-cogpit"); hasPill(other, "certificate not checked") || other.note != nil {
		t.Error("a checked context is marked")
	}
	if got := insecureNote(engine.ContextInfo{Insecure: true}); !strings.HasSuffix(got, "its server") {
		t.Errorf("without a server: %q", got)
	}
}

func TestRenderPickerInsecureContext(t *testing.T) {
	for _, tc := range []struct {
		mode ThemeMode
		name string
	}{{ThemeDark, "picker_insecure_dark.png"}, {ThemeLight, "picker_insecure_light.png"}} {
		t.Run(tc.name, func(t *testing.T) {
			f := insecureFake(t)
			sh, w, q := newTestShell(t, f, Options{Theme: tc.mode})
			q.until(t, "the picker rows", func() bool { return sh.picker.visible() && len(sh.picker.list.Objects) == len(f.contexts) })
			savePNG(t, w, tc.name)
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
	q.until(t, "the picker", sh.picker.visible)
	sh.kc.showConnectLog("dev-hfi")
	q.until(t, "the log dialog", func() bool { return sh.kc.connectLog != nil })
	if n := len(sh.kc.connectLog.view.Lines); n != 3 {
		t.Errorf("%d lines", n)
	}
	savePNG(t, w, "connection_updates_insecure_dark.png")
}
