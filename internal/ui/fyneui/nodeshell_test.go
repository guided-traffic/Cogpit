package fyneui

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/terminal"
	"github.com/guided-traffic/cogpit/internal/ui/fyneui/termview"
)

// The node shell in the window (G7, G8; D-37): its confirmation, which names the privileged pod,
// the namespace, the node and the image; its dock tab, a terminal on Session.OpenTerminal; and
// Cluster Settings › Node Shell. The row menus and the drawer's toolbar: TestRowMenuNodes,
// TestDrawerToolbarFromRegistry, TestRegistryMenusAsLens. PNGs:
// testdata/node_shell_confirm_{dark,light}.png, node_shell_{dark,light}.png and
// cluster_settings_node_shell_{dark,light}.png.

const nodeShellPod = "cogpit-node-shell-k3v9q2mx"

// nodeShellOutput is what the engine's node shell session shows: its notice, then the node's
// shell (the shell's lines are the node's, as on a kind node).
var nodeShellOutput = "[cogpit] Node shell on " + busyNode + ": privileged pod kube-system/" + nodeShellPod + ", image " +
	engine.DefaultNodeShellImage + "; the pod ends by itself after " + engine.NodeShellDeadlineText + ".\r\n" +
	"[cogpit] Waiting for the pod to run…\r\n" +
	"root@" + busyNode + ":/# hostname\r\n" + busyNode + "\r\nroot@" + busyNode + ":/# crictl ps --name kube-proxy -q\r\n" +
	"5f2c0e8a9d14\r\nroot@" + busyNode + ":/# "

// nodeShellEngine is the fake engine whose node shell tabs stream nodeShellOutput.
type nodeShellEngine struct {
	*fakeEngine
	last *atomic.Pointer[openStream]
}

func (n nodeShellEngine) OpenTerminal(ctx context.Context, tabID string) (terminal.Session, error) {
	if !strings.HasPrefix(tabID, string(engine.DockNodeShell)+"-") {
		return n.fakeEngine.OpenTerminal(ctx, tabID)
	}
	n.mu.Lock()
	n.opened[tabID]++
	n.mu.Unlock()
	s := newGatedStream(n.fakeEngine, tabID, nodeShellOutput)
	n.last.Store(s)
	return s, nil
}

// nodeShellScene is the Nodes view with a node shell tab open in the dock, its output shown.
func nodeShellScene(t *testing.T, mode ThemeMode) (*shell, fyne.Window) {
	t.Helper()
	f := kindsFake(t, "nodes")
	ref := model.ObjectRef{Context: "demo", Kind: "nodes", Name: busyNode}
	f.ui.Dock = []engine.DockTab{{ID: "nodeShell-1", Type: engine.DockNodeShell, Title: "Node shell: " + busyNode, Context: "demo",
		Ref: &ref, Container: engine.NodeShellContainer, NodeShell: &engine.NodeShellTab{Node: busyNode, Namespace: engine.NodeShellNamespace,
			Pod: nodeShellPod, Image: engine.DefaultNodeShellImage, Pinned: true, DeadlineSeconds: 7200}}}
	f.ui.ActiveDock = "nodeShell-1"
	eng := nodeShellEngine{fakeEngine: f, last: &atomic.Pointer[openStream]{}}
	sh, w, q := startShell(t, eng, f, Options{Theme: mode})
	// As attachScene: the output flows once the shell has settled and the widget waits in Read.
	q.until(t, "the terminal to read", func() bool {
		s := eng.last.Load()
		return len(sh.kinds) > 0 && sh.view.snap != nil && s != nil && s.reading.Load()
	})
	q.drain()
	s := eng.last.Load()
	close(s.gate)
	deadline := time.Now().Add(5 * time.Second)
	for !s.idle.Load() {
		if time.Now().After(deadline) {
			t.Fatal("the terminal did not take the node shell's output")
		}
		time.Sleep(time.Millisecond)
	}
	term := sh.dock.contents["nodeShell-1"].obj.Objects[0].(*termview.Terminal)
	t.Cleanup(func() {
		_ = term.Close()
		select {
		case <-term.Done():
		case <-time.After(10 * time.Second):
			t.Error("the node shell's widget did not end")
		}
	})
	return sh, w
}

// TestNodeShellTabShowsATerminal: a node shell tab is a terminal on Session.OpenTerminal, titled
// "Node shell: <node>".
func TestNodeShellTabShowsATerminal(t *testing.T) {
	sh, _ := nodeShellScene(t, ThemeDark)
	term, ok := sh.dock.contents["nodeShell-1"].obj.Objects[0].(*termview.Terminal)
	if !ok {
		t.Fatalf("the node shell tab shows %T", sh.dock.contents["nodeShell-1"].obj.Objects[0])
	}
	if text := term.Text(); !strings.Contains(text, "privileged pod kube-system/"+nodeShellPod) || !strings.Contains(text, "root@"+busyNode) {
		t.Errorf("terminal %q", text)
	}
	if b := sh.dock.tabRow.Objects[0].(*tabButton); b.title != "Node shell: "+busyNode {
		t.Errorf("tab %q", b.title)
	}
}

func TestRenderNodeShellDark(t *testing.T) {
	_, w := nodeShellScene(t, ThemeDark)
	savePNG(t, w, "node_shell_dark.png")
}

func TestRenderNodeShellLight(t *testing.T) {
	_, w := nodeShellScene(t, ThemeLight)
	savePNG(t, w, "node_shell_light.png")
}

// nodeShellRequest is the confirmation the engine asks for a node shell on busyNode (as Dispatch
// builds it from node.shell: its label, the node, the context, the detail).
func nodeShellRequest(origin engine.Origin) engine.ConfirmRequest {
	ref := model.ObjectRef{Context: "demo", Kind: "nodes", Name: busyNode}
	return engine.ConfirmRequest{Command: engine.NodeShellCommand, Label: "Shell", Danger: engine.DangerDestructive, Origin: origin,
		Context: "demo", Objects: []model.ObjectRef{ref}, Detail: engine.NodeShellDetail(busyNode, engine.DefaultNodeShellImage, "")}
}

// confirmNodeShell shows the node shell's confirmation and returns the dialog's texts; answer is
// tapped afterwards.
func confirmNodeShell(t *testing.T, mode ThemeMode, origin engine.Origin, png string) []string {
	t.Helper()
	f := kindsFake(t, "nodes")
	sh, w, q := newTestShell(t, f, Options{Theme: mode})
	q.until(t, "the nodes", func() bool { return len(sh.kinds) > 0 && sh.view.snap != nil })
	done := make(chan bool, 1)
	go func() {
		ok, _ := sh.confirmer.Confirm(context.Background(), nodeShellRequest(origin))
		done <- ok
	}()
	q.until(t, "the dialog", func() bool { return sh.confirmer.current != nil })
	q.drain()
	texts := overlayTexts(w)
	if png != "" {
		savePNG(t, w, png)
	}
	test.Tap(sh.confirmer.current.cancel)
	if <-done {
		t.Error("Cancel confirmed the node shell")
	}
	return texts
}

// TestNodeShellConfirmationNamesWhatItCreates: the dialog says it creates a privileged pod with
// root access, names the namespace, the node, the image and the context, and who asked.
func TestNodeShellConfirmationNamesWhatItCreates(t *testing.T) {
	texts := strings.Join(confirmNodeShell(t, ThemeDark, engine.OriginExternalAgent, ""), "\n")
	for _, want := range []string{"An agent asks: Shell?", "privileged pod in namespace kube-system on node " + busyNode,
		"root access to the node", "Image: docker.io/library/alpine:3.19, pinned by digest sha256:6baf43584bcb", "demo",
		"An external agent over MCP"} {
		if !strings.Contains(texts, want) {
			t.Errorf("the confirmation lacks %q:\n%s", want, texts)
		}
	}
}

func TestRenderNodeShellConfirmDark(t *testing.T) {
	confirmNodeShell(t, ThemeDark, engine.OriginGUI, "node_shell_confirm_dark.png")
}

func TestRenderNodeShellConfirmLight(t *testing.T) {
	confirmNodeShell(t, ThemeLight, engine.OriginGUI, "node_shell_confirm_light.png")
}

// nodeShellField is a shown Node Shell section's field.
func nodeShellField(sh *shell, setting string) *widget.Entry {
	switch c := sh.settings.controls[setting].(type) {
	case *widget.Entry:
		return c
	case *fyne.Container:
		e, _ := c.Objects[0].(*widget.Entry)
		return e
	}
	return nil
}

// TestClusterSettingsNodeShellSection: the image field shows Cogpit's default as its placeholder
// and says it is pinned; an image typed is kept, and a tag says it floats; a pull secret typed is
// kept; what cannot be an image is refused under the field.
func TestClusterSettingsNodeShellSection(t *testing.T) {
	s, _ := settingsEngine(t)
	sh, w, q := newEngineShell(t, s, Options{})
	mustDispatch(t, s, "cluster.settings.open", map[string]any{"context": "kind-dev", "section": engine.SectionNodeShell})
	q.until(t, "the Node Shell section", func() bool {
		return sh.settings.visible() && nodeShellField(sh, nodeShellImageSetting) != nil && sh.settings.view != nil
	})
	image, note := nodeShellField(sh, nodeShellImageSetting), sh.settings.controls["nodeShellNote"].(*textLine)
	if image.PlaceHolder != engine.DefaultNodeShellImage || note.text != nodeShellNotePinned {
		t.Errorf("placeholder %q, note %q", image.PlaceHolder, note.text)
	}

	w.Canvas().Focus(image)
	test.Type(image, "registry.example.com/tools/shell:1.2")
	q.until(t, "the image kept", func() bool {
		cs, _ := s.ClusterSettingsOf("kind-dev")
		return cs.NodeShellImage == "registry.example.com/tools/shell:1.2"
	})
	w.Canvas().Unfocus()
	q.until(t, "the floating note", func() bool { return note.text == nodeShellNoteFloating })

	secret := nodeShellField(sh, nodeShellPullSecretSetting)
	w.Canvas().Focus(secret)
	test.Type(secret, "registry-pull")
	q.until(t, "the pull secret kept", func() bool {
		cs, _ := s.ClusterSettingsOf("kind-dev")
		return cs.NodeShellPullSecret == "registry-pull"
	})

	w.Canvas().Focus(image)
	image.SetText("alpine 3.19")
	q.until(t, "the refusal", func() bool { return sh.settings.errs[nodeShellImageSetting].Visible() })
	if text := sh.settings.errs[nodeShellImageSetting].text; !strings.Contains(text, "without spaces") {
		t.Errorf("refusal %q", text)
	}
}

func TestRenderClusterSettingsNodeShellDark(t *testing.T) {
	renderClusterSettingsNodeShell(t, ThemeDark, "cluster_settings_node_shell_dark.png")
}

func TestRenderClusterSettingsNodeShellLight(t *testing.T) {
	renderClusterSettingsNodeShell(t, ThemeLight, "cluster_settings_node_shell_light.png")
}

func renderClusterSettingsNodeShell(t *testing.T, mode ThemeMode, name string) {
	s, _ := settingsEngine(t)
	mustDispatch(t, s, "cluster.settings.set", map[string]any{"context": "kind-dev", "name": "Dev cluster", "iconColor": "violet"})
	mustDispatch(t, s, "cluster.settings.setNodeShell", map[string]any{"context": "kind-dev", "imagePullSecret": "registry-pull"})
	sh, w, q := newEngineShell(t, s, Options{Theme: mode})
	mustDispatch(t, s, "cluster.settings.open", map[string]any{"context": "kind-dev", "section": engine.SectionNodeShell})
	q.until(t, "the Node Shell section", func() bool {
		e := nodeShellField(sh, nodeShellPullSecretSetting)
		return sh.settings.visible() && sh.settings.view != nil && e != nil && e.Text == "registry-pull"
	})
	q.drain()
	savePNG(t, w, name)
}
