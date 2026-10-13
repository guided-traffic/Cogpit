package fyneui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// A cluster's Settings › Node Shell (G13, D-37; engine/nodeshell.go), as Lens's page holds it (its
// capture: the image, Lens's default shown as the field's placeholder, and an optional image pull
// secret in kube-system): the image of the privileged pod a node shell runs, with Cogpit's default,
// pinned by digest, as the placeholder, and the image pull secret. A line under the image says
// whether it is pinned by digest or a floating tag, as the node shell's confirmation does. Only the
// user sets them (cluster.settings.setNodeShell refuses agents anything but the defaults). Its place
// after Metrics follows Lens's order without the pages Cogpit lacks (Lens Metrics, Cost).

const (
	nodeShellSectionLabel      = "Node Shell"
	nodeShellImageSetting      = "nodeShellImage"
	nodeShellPullSecretSetting = "nodeShellPullSecret"
	nodeShellNotePinned        = "Pinned by digest: every node shell runs exactly this image."
	nodeShellNoteFloating      = "A floating tag, not pinned by digest: what it runs can change without Cogpit knowing. The node shell's confirmation says so."
	nodeShellNoteDemo          = "The demo cluster is generated in memory: it has no nodes to open a shell on."
)

func init() { clusterSectionLabels[engine.SectionNodeShell] = nodeShellSectionLabel }

// clusterNodeShellSection builds a cluster's Settings › Node Shell.
func (p *settingsPage) clusterNodeShellSection() fyne.CanvasObject {
	image := widget.NewEntry()
	image.SetPlaceHolder(engine.DefaultNodeShellImage)
	image.OnChanged = func(string) {
		p.typed(nodeShellImageSetting, func() { p.setNodeShell("image", nodeShellImageSetting, strings.TrimSpace(image.Text)) })
	}
	secret := widget.NewEntry()
	secret.SetPlaceHolder("none")
	secret.OnChanged = func(string) {
		p.typed(nodeShellPullSecretSetting, func() {
			p.setNodeShell("imagePullSecret", nodeShellPullSecretSetting, strings.TrimSpace(secret.Text))
		})
	}
	note := newTextLine("", colorMuted, theme.SizeNameText, fyne.TextStyle{})
	note.truncate = true
	p.controls["nodeShellNote"] = note
	p.sync = func() {
		v := p.clusterView()
		p.setEntry(image, image, nodeShellImageSetting, v.NodeShellImage)
		p.setEntry(secret, secret, nodeShellPullSecretSetting, v.NodeShellPullSecret)
		shown := strings.TrimSpace(image.Text)
		if shown == "" {
			shown = engine.DefaultNodeShellImage
		}
		switch {
		case p.shown.Context == engine.DemoContext:
			image.Disable()
			secret.Disable()
			note.SetText(nodeShellNoteDemo)
		case engine.ImagePinned(shown):
			note.SetText(nodeShellNotePinned)
		default:
			note.SetText(nodeShellNoteFloating)
		}
	}
	return container.NewVBox(
		p.setting(nodeShellImageSetting, "Image",
			"The image of the privileged pod a node shell runs in kube-system; empty: Cogpit's default, Alpine 3.19 pinned by digest.",
			container.NewVBox(image, note)),
		p.setting(nodeShellPullSecretSetting, "Image pull secret",
			"A Secret in kube-system that pulls that image; empty: none.", secret),
	)
}

// setNodeShell dispatches the user's node shell image or pull secret; a refusal shows under it.
func (p *settingsPage) setNodeShell(param, setting, value string) {
	if p.syncing {
		return
	}
	p.sh.dispatch("cluster.settings.setNodeShell", map[string]any{contextParam: p.shown.Context, param: value},
		func(_ any, err error) { p.showResult(setting, err) })
}
