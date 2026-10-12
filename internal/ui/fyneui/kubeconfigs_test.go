package fyneui

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// sourceContexts are the picker's contexts with kubeconfig sources: a pasted kubeconfig, a file
// whose login program is not allowed yet, and a synced one whose program changed since.
func sourceContexts() []engine.ContextInfo {
	return append(fixtureContexts()[1:3],
		engine.ContextInfo{Name: "kind-cogpit [pasted 1]", KubeContext: "kind-cogpit", Source: "pasted 1", Cluster: "kind-cogpit",
			Server: "https://127.0.0.1:52341", File: "/Users/me/Library/Application Support/Cogpit/kubeconfigs/pasted-1f.yaml"},
		engine.ContextInfo{Name: "prod-eks [eks.yaml]", KubeContext: "prod-eks", Source: "eks.yaml", Cluster: "arn:aws:eks:eu-central-1:000000000000:cluster/prod",
			Server: "https://ABCDEF.gr7.eu-central-1.eks.amazonaws.com", File: "/Users/me/Downloads/eks.yaml", Exec: engine.ExecRequired},
		engine.ContextInfo{Name: "team [kubeconfigs/team.yaml]", KubeContext: "team", Source: "kubeconfigs/team.yaml", Cluster: "team",
			Server: "https://team.example:6443", Namespace: "team-a", File: "/Users/me/kubeconfigs/team.yaml", Exec: engine.ExecChanged},
	)
}

// pluginView is what kubeconfig.execPlugin returns for the EKS context of sourceContexts.
func pluginView(state string) map[string]any {
	return map[string]any{
		"context": "prod-eks [eks.yaml]", "source": "eks.yaml", "state": state, "fingerprint": "sha256:feed",
		"plugin": map[string]any{
			"file": "/Users/me/Downloads/eks.yaml", "context": "prod-eks", "user": "eks-user",
			"server": "https://ABCDEF.gr7.eu-central-1.eks.amazonaws.com", "apiVersion": "client.authentication.k8s.io/v1beta1",
			"command": "aws", "args": []string{"--region", "eu-central-1", "eks", "get-token", "--cluster-name", "prod"},
			"envNames": []string{"AWS_PROFILE", "PATH"},
		},
		"found": "/opt/homebrew/bin/aws", "riskyEnv": []string{"PATH"},
	}
}

// recorder records the kubeconfig commands a test's fake engine receives.
type recorder struct {
	mu    sync.Mutex
	calls []engine.Invocation
}

func (r *recorder) add(inv engine.Invocation) {
	r.mu.Lock()
	r.calls = append(r.calls, inv)
	r.mu.Unlock()
}

func (r *recorder) ids() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.calls))
	for i, c := range r.calls {
		out[i] = c.ID
	}
	return out
}

func (r *recorder) params(id string) map[string]string {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.calls {
		if c.ID == id {
			var p map[string]string
			_ = json.Unmarshal(c.Params, &p)
			return p
		}
	}
	return nil
}

// kubeconfigFake is a fake engine with the source contexts and handlers for the kubeconfig
// commands; cluster.connect is recorded too.
func kubeconfigFake(t *testing.T, state string) (*fakeEngine, *recorder) {
	f := newFakeEngine(t)
	f.contexts = sourceContexts()
	rec := &recorder{}
	handle := func(id string, res func(engine.Invocation) (any, error)) {
		f.handlers[id] = func(_ context.Context, inv engine.Invocation) (any, error) { rec.add(inv); return res(inv) }
	}
	handle("kubeconfig.execPlugin", func(engine.Invocation) (any, error) { return pluginView(state), nil })
	handle("kubeconfig.approveExec", func(engine.Invocation) (any, error) { return pluginView(engine.ExecApproved), nil })
	handle("kubeconfig.paste", func(inv engine.Invocation) (any, error) {
		var p map[string]string
		_ = json.Unmarshal(inv.Params, &p)
		if !strings.Contains(p["kubeconfig"], "contexts:") {
			return nil, errors.New("kubeconfig: the document defines no context")
		}
		return map[string]any{"source": map[string]any{"id": "pasted 2", "kind": "pasted"}, "contexts": []string{"x [pasted 2]"}}, nil
	})
	handle("kubeconfig.sources", func(engine.Invocation) (any, error) {
		return map[string]any{"sources": []map[string]any{
			{"id": "pasted 1", "kind": "pasted", "path": "/Users/me/Library/Application Support/Cogpit/kubeconfigs/pasted-1f.yaml", "files": []string{"pasted 1"}, "contexts": 1},
			{"id": "eks.yaml", "kind": "file", "path": "/Users/me/Downloads/eks.yaml", "files": []string{"eks.yaml"}, "contexts": 1},
			{"id": "kubeconfigs/", "kind": "folder", "path": "/Users/me/kubeconfigs", "files": []string{"kubeconfigs/team.yaml"}, "contexts": 1, "skipped": 2},
			{"id": "old.yaml", "kind": "file", "path": "/Users/me/old.yaml", "contexts": 0, "error": "kubeconfig file /Users/me/old.yaml does not exist"},
		}}, nil
	})
	handle("kubeconfig.remove", func(engine.Invocation) (any, error) { return map[string]any{"id": "eks.yaml"}, nil })
	handle("kubeconfig.addFile", func(engine.Invocation) (any, error) {
		return map[string]any{"source": map[string]any{"id": "dev.yaml", "kind": "file"}, "contexts": []string{"dev [dev.yaml]"}}, nil
	})
	return f, rec
}

func (f *fakeEngine) connects() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for _, c := range f.calls {
		if c.ID == "cluster.connect" {
			var p contextParams
			_ = json.Unmarshal(c.Params, &p)
			out = append(out, p.Context)
		}
	}
	return out
}

// findButton returns the visible button labelled label in the window's top overlay (a dialog), or
// nil.
func findButton(w fyne.Window, label string) *widget.Button {
	overlay := w.Canvas().Overlays().Top()
	if overlay == nil {
		return nil
	}
	var found *widget.Button
	var walk func(o fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		if found != nil || o == nil || !o.Visible() {
			return
		}
		switch x := o.(type) {
		case *widget.Button:
			if x.Text == label {
				found = x
			}
		case *fyne.Container:
			for _, c := range x.Objects {
				walk(c)
			}
		case fyne.Widget:
			for _, c := range test.WidgetRenderer(x).Objects() {
				walk(c)
			}
		}
	}
	walk(overlay)
	return found
}

func selectAndConnect(t *testing.T, sh *shell, q *mainQueue, name string) {
	t.Helper()
	q.until(t, "the picker's contexts", func() bool { return sh.picker.visible() && len(sh.picker.contexts) > 0 })
	sh.picker.selected = name
	sh.picker.render()
	test.Tap(sh.picker.connectB)
}

// TestPickerShowsTheLoginProgramBeforeItRuns: connecting to a context whose login program the user
// has not allowed opens the approval dialog first; nothing connects until the user allows it, and
// allowing sends the fingerprint the dialog showed, then connects.
func TestPickerShowsTheLoginProgramBeforeItRuns(t *testing.T) {
	f, rec := kubeconfigFake(t, engine.ExecRequired)
	sh, _, q := newTestShell(t, f, Options{})
	selectAndConnect(t, sh, q, "prod-eks [eks.yaml]")
	q.until(t, "the approval dialog", func() bool { return sh.kc.approval != nil })
	if got := f.connects(); len(got) != 0 {
		t.Fatalf("connected before the program was allowed: %v", got)
	}
	test.Tap(sh.kc.approval.allow)
	q.until(t, "the connection", func() bool { return len(f.connects()) == 1 })
	if p := rec.params("kubeconfig.approveExec"); p["context"] != "prod-eks [eks.yaml]" || p["fingerprint"] != "sha256:feed" {
		t.Errorf("approveExec %v", p)
	}
	if got := f.connects(); got[0] != "prod-eks [eks.yaml]" {
		t.Errorf("connected %v", got)
	}
}

// TestPickerCancelledApprovalDoesNotConnect: cancelling leaves the picker open, says why, and
// connects nothing.
func TestPickerCancelledApprovalDoesNotConnect(t *testing.T) {
	f, rec := kubeconfigFake(t, engine.ExecChanged)
	sh, _, q := newTestShell(t, f, Options{})
	selectAndConnect(t, sh, q, "prod-eks [eks.yaml]")
	q.until(t, "the approval dialog", func() bool { return sh.kc.approval != nil })
	test.Tap(sh.kc.approval.cancel)
	q.until(t, "the picker's answer", func() bool { return sh.picker.connecting == "" })
	if !strings.Contains(sh.picker.status.text, "was not allowed") || !sh.picker.visible() {
		t.Errorf("status %q", sh.picker.status.text)
	}
	if slices.Contains(rec.ids(), "kubeconfig.approveExec") || len(f.connects()) != 0 {
		t.Errorf("calls %v, connects %v", rec.ids(), f.connects())
	}
}

// TestPickerConnectsAllowedAndPastedContextsDirectly: no dialog for a context without a program to
// allow.
func TestPickerConnectsAllowedAndPastedContextsDirectly(t *testing.T) {
	f, rec := kubeconfigFake(t, engine.ExecRequired)
	sh, _, q := newTestShell(t, f, Options{})
	selectAndConnect(t, sh, q, "kind-cogpit [pasted 1]")
	q.until(t, "the connection", func() bool { return len(f.connects()) == 1 })
	if sh.kc.approval != nil || slices.Contains(rec.ids(), "kubeconfig.execPlugin") {
		t.Errorf("asked about a context without a program: %v", rec.ids())
	}
}

// TestPasteDialog: the text goes to kubeconfig.paste; an error stays in the dialog; success closes
// it and the widget keeps no copy of the text.
func TestPasteDialog(t *testing.T) {
	f, rec := kubeconfigFake(t, engine.ExecRequired)
	sh, _, q := newTestShell(t, f, Options{})
	q.until(t, "the picker", sh.picker.visible)
	sh.kc.showPaste()
	p := sh.kc.paste
	test.Tap(p.add)
	if !strings.Contains(p.status.text, "Paste a kubeconfig first") || slices.Contains(rec.ids(), "kubeconfig.paste") {
		t.Fatalf("empty paste: %q %v", p.status.text, rec.ids())
	}
	p.entry.SetText("apiVersion: v1\nkind: Config\n")
	test.Tap(p.add)
	q.until(t, "the error", func() bool { return strings.Contains(p.status.text, "no context") })
	if sh.kc.paste == nil {
		t.Fatal("an error closed the dialog")
	}
	text := "apiVersion: v1\nkind: Config\ncontexts:\n- name: x\n  context: {cluster: c, user: u}\n"
	p.entry.SetText(text)
	test.Tap(p.add)
	q.until(t, "the dialog to close", func() bool { return sh.kc.paste == nil })
	if got := rec.params("kubeconfig.paste"); got["kubeconfig"] == "" {
		t.Errorf("paste params %v", got)
	}
	if p.entry.Text != "" {
		t.Error("the closed dialog keeps the pasted text")
	}
}

// TestSourcesDialogRemovesAfterConfirming: Remove asks, then dispatches kubeconfig.remove.
func TestSourcesDialogRemovesAfterConfirming(t *testing.T) {
	f, rec := kubeconfigFake(t, engine.ExecRequired)
	sh, w, q := newTestShell(t, f, Options{})
	q.until(t, "the picker", sh.picker.visible)
	sh.kc.showSources()
	sd := sh.kc.sources
	q.until(t, "the sources", func() bool { return len(sd.rows) == 4 })
	test.Tap(sd.removeB["eks.yaml"])
	confirm := findButton(w, "Yes")
	if confirm == nil {
		t.Fatal("no confirmation before removing")
	}
	if slices.Contains(rec.ids(), "kubeconfig.remove") {
		t.Fatal("removed before the confirmation")
	}
	test.Tap(confirm)
	q.until(t, "kubeconfig.remove", func() bool { return slices.Contains(rec.ids(), "kubeconfig.remove") })
	if p := rec.params("kubeconfig.remove"); p["source"] != "eks.yaml" {
		t.Errorf("remove %v", p)
	}
}

// TestAddFileFromTheMenuAndEventsReloadThePicker: the Cluster menu offers the kubeconfig items; a
// picked file goes to kubeconfig.addFile; EventContexts makes the open picker list again.
func TestAddFileFromTheMenuAndEventsReloadThePicker(t *testing.T) {
	f, rec := kubeconfigFake(t, engine.ExecRequired)
	sh, _, q := newTestShell(t, f, Options{})
	q.until(t, "the picker", func() bool { return sh.picker.visible() && len(sh.picker.contexts) == len(f.contexts) })
	var labels []string
	for _, it := range sh.mainMenu().Items[0].Items {
		labels = append(labels, it.Label)
	}
	for _, want := range []string{"Add Kubeconfig File…", "Paste Kubeconfig…", "Add Sync Folder…", "Kubeconfig Sources…"} {
		if !slices.Contains(labels, want) {
			t.Errorf("Cluster menu %v lacks %q", labels, want)
		}
	}
	var asked []bool
	sh.kc.pick = func(folder bool, done func(string)) { asked = append(asked, folder); done("/Users/me/dev.yaml") }
	sh.kc.addFile()
	q.until(t, "kubeconfig.addFile", func() bool { return slices.Contains(rec.ids(), "kubeconfig.addFile") })
	if p := rec.params("kubeconfig.addFile"); p["path"] != "/Users/me/dev.yaml" || len(asked) != 1 || asked[0] {
		t.Errorf("addFile %v, asked %v", p, asked)
	}

	f.mu.Lock()
	f.contexts = append(f.contexts, engine.ContextInfo{Name: "dev [dev.yaml]", KubeContext: "dev", Source: "dev.yaml"})
	f.mu.Unlock()
	f.emit(engine.Event{Type: engine.EventContexts})
	q.until(t, "the picker to list the new context", func() bool { return len(sh.picker.contexts) == len(sourceContexts())+1 })
}

func TestRenderPickerWithSources(t *testing.T) {
	f, _ := kubeconfigFake(t, engine.ExecRequired)
	sh, w, q := newTestShell(t, f, Options{})
	q.until(t, "the picker", func() bool { return sh.picker.visible() && len(sh.picker.list.Objects) == len(f.contexts) })
	savePNG(t, w, "picker_sources_dark.png")
}

func renderApproval(t *testing.T, mode ThemeMode, state, name string) {
	f, _ := kubeconfigFake(t, state)
	sh, w, q := newTestShell(t, f, Options{Theme: mode})
	selectAndConnect(t, sh, q, "prod-eks [eks.yaml]")
	q.until(t, "the approval dialog", func() bool { return sh.kc.approval != nil })
	savePNG(t, w, name)
}

func TestRenderExecApprovalDark(t *testing.T) {
	renderApproval(t, ThemeDark, engine.ExecRequired, "exec_approval_dark.png")
}

func TestRenderExecApprovalLight(t *testing.T) {
	renderApproval(t, ThemeLight, engine.ExecChanged, "exec_approval_changed_light.png")
}

func TestRenderPasteKubeconfig(t *testing.T) {
	f, _ := kubeconfigFake(t, engine.ExecRequired)
	sh, w, q := newTestShell(t, f, Options{})
	q.until(t, "the picker", sh.picker.visible)
	sh.kc.showPaste()
	sh.kc.paste.entry.SetText("apiVersion: v1\nkind: Config\nclusters:\n- name: kind-cogpit\n  cluster:\n    server: https://127.0.0.1:52341\n" +
		"contexts:\n- name: kind-cogpit\n  context: {cluster: kind-cogpit, user: kind-cogpit}\n")
	savePNG(t, w, "paste_kubeconfig_dark.png")
}

func TestRenderKubeconfigSources(t *testing.T) {
	f, _ := kubeconfigFake(t, engine.ExecRequired)
	sh, w, q := newTestShell(t, f, Options{})
	q.until(t, "the picker", sh.picker.visible)
	sh.kc.showSources()
	q.until(t, "the sources", func() bool { return sh.kc.sources != nil && len(sh.kc.sources.rows) == 4 })
	savePNG(t, w, "kubeconfig_sources_dark.png")
}
