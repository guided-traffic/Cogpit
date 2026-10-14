package fyneui

import (
	"context"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// insecureOnly makes the pending group a context that waits only because its cluster skips the
// certificate check (kind insecure, A28): no program, no token file, the one context's server
// reached without the check.
func insecureOnly(group, plugin, ctx map[string]any) {
	group["kind"] = "insecure"
	group["found"], group["riskyEnv"] = "", nil
	for _, k := range []string{"apiVersion", "command", "args", "envNames"} {
		delete(plugin, k)
	}
	ctx["server"] = "https://192.168.49.2:8443"
	ctx["tls"] = map[string]any{"insecure": true}
}

// labContext is a source context whose cluster skips the certificate check and waits for the user.
var labContext = engine.ContextInfo{
	Name: "lab [lab.yaml]", Source: "lab.yaml", KubeContext: "lab", File: "/Users/me/lab.yaml",
	Server: "https://192.168.49.2:8443", Exec: engine.ExecRequired, ExecKind: "insecure", Insecure: true,
}

// openLabApproval connects labContext from the picker of a fake engine whose pending approval is
// that context alone, kind insecure, in state, and waits for the dialog.
func openLabApproval(t *testing.T, state string, mode ThemeMode) (*fakeEngine, *recorder, *shell, fyne.Window, *mainQueue) {
	t.Helper()
	f, rec := kubeconfigFake(t, state)
	lab := labContext
	lab.Exec = state
	f.contexts = append(f.contexts, lab)
	f.handlers["kubeconfig.pendingApprovals"] = func(_ context.Context, inv engine.Invocation) (any, error) {
		rec.add(inv)
		v := pendingFixture(state)
		g := group(v)
		ctx := g["contexts"].([]any)[0].(map[string]any)
		insecureOnly(g, g["plugin"].(map[string]any), ctx)
		g["file"], g["source"] = "/Users/me/lab.yaml", "lab.yaml"
		ctx["context"], ctx["kubeContext"] = labContext.Name, "lab"
		return v, nil
	}
	sh, w, q := newTestShell(t, f, Options{Theme: mode})
	selectAndConnect(t, sh, q, labContext.Name)
	q.until(t, "the approval dialog", func() bool { return sh.kc.approval != nil })
	return f, rec, sh, w, q
}

// TestApprovalDialogAsksForAnUncheckedConnection (A28, H-37): connecting such a context from the
// picker says what is reviewed and opens the dialog before anything connects; the dialog asks
// whether to connect without the certificate check, says what that hands to whoever answers and
// that it asks once, names the server and the kubeconfig, shows no program, and its button allows
// exactly that one context's fingerprint, then connects.
func TestApprovalDialogAsksForAnUncheckedConnection(t *testing.T) {
	f, rec, sh, w, q := openLabApproval(t, engine.ExecRequired, ThemeDark)
	if got := f.connects(); len(got) != 0 {
		t.Fatalf("connected before the user allowed it: %v", got)
	}
	if s := sh.picker.status.text; !strings.Contains(s, "whose certificate is not checked") {
		t.Errorf("picker status %q", s)
	}
	if !hasPill(pickerRowOf(t, sh, "lab"), "unchecked connection not allowed yet") {
		t.Error("the picker row does not say the unchecked connection waits")
	}
	texts := overlayTexts(w)
	for _, want := range []string{
		"Connect without checking the certificate?",
		"switches the check of the API server's certificate off (insecure-skip-tls-verify)",
		"it asks once for this context, and again if its server or proxy changes",
		"Certificate checks are off for this connection", "https://192.168.49.2:8443", "/Users/me/lab.yaml",
	} {
		if !hasText(texts, want) {
			t.Errorf("the dialog does not show %q; it shows %q", want, texts)
		}
	}
	for _, not := range []string{"Program", "Arguments", "Environment", "Allow this program"} {
		if hasText(texts, not) {
			t.Errorf("the dialog shows %q for a context that runs nothing", not)
		}
	}
	a := sh.kc.approval
	if a.allow.Text != "Allow and Connect" {
		t.Errorf("button %q", a.allow.Text)
	}
	test.Tap(a.allow)
	q.until(t, "the connect", func() bool { return len(f.connects()) == 1 })
	if p := rec.approvals(); len(p) != 1 || p[0].Context != labContext.Name || p[0].Fingerprint != "sha256:feed" {
		t.Errorf("approveExec %v", p)
	}
	if got := f.connects(); got[0] != labContext.Name {
		t.Errorf("connected %v", got)
	}
}

// The picker's row of such a context, beside a checked one, light and dark: the "certificate not
// checked" pill and line, and the pill saying the unchecked connection waits for the user.
func TestRenderPickerInsecureWaiting(t *testing.T) {
	for _, tc := range []struct {
		mode ThemeMode
		name string
	}{{ThemeDark, "picker_insecure_waiting_dark.png"}, {ThemeLight, "picker_insecure_waiting_light.png"}} {
		t.Run(tc.name, func(t *testing.T) {
			f, _ := kubeconfigFake(t, engine.ExecRequired)
			f.contexts = append([]engine.ContextInfo{labContext}, f.contexts...) // at the top, in view
			sh, w, q := newTestShell(t, f, Options{Theme: tc.mode})
			sh.picker.show()
			q.until(t, "the picker rows", func() bool { return sh.picker.visible() && len(sh.picker.list.Objects) == len(f.contexts) })
			if !hasPill(pickerRowOf(t, sh, "lab"), "certificate not checked") {
				t.Error("no certificate pill")
			}
			savePNG(t, w, tc.name)
		})
	}
}

func TestRenderInsecureApproval(t *testing.T) {
	for _, tc := range []struct {
		mode  ThemeMode
		state string
		name  string
	}{
		{ThemeDark, engine.ExecRequired, "insecure_approval_dark.png"},
		{ThemeLight, engine.ExecRequired, "insecure_approval_light.png"},
		{ThemeLight, engine.ExecChanged, "insecure_approval_changed_light.png"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, _, w, _ := openLabApproval(t, tc.state, tc.mode)
			savePNG(t, w, tc.name)
		})
	}
}
