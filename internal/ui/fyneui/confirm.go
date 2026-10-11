package fyneui

import (
	"context"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// guiConfirmer is the engine's Confirmer for the desktop app: a dialog that names the command,
// the objects, the namespace, the context and who asked. The engine calls Confirm on its own
// goroutines; the dialog is shown on the main goroutine and Confirm waits for the answer.
//
// One dialog is shown at a time, in the order the requests came: a second request (an agent's,
// while the user is about to confirm their own) would otherwise open on top, with its confirm
// button where the user's pointer already is. For the same reason the confirm button arms only
// armDelay after the dialog appears.
type guiConfirmer struct {
	sh       *shell
	slot     chan struct{}  // holds a token while a dialog is shown
	armDelay time.Duration  // how long the confirm button stays disabled
	current  *confirmDialog // the dialog on screen; main goroutine only
}

// defaultArmDelay keeps a click meant for the previous dialog, or a double click, from confirming.
const defaultArmDelay = 750 * time.Millisecond

func newGUIConfirmer(sh *shell) *guiConfirmer {
	return &guiConfirmer{sh: sh, slot: make(chan struct{}, 1), armDelay: defaultArmDelay}
}

var _ engine.Confirmer = (*guiConfirmer)(nil)

// confirmDialog is one shown confirmation.
type confirmDialog struct {
	d       *dialog.CustomDialog
	confirm *widget.Button
	cancel  *widget.Button
}

// Confirm implements engine.Confirmer. It returns the user's answer, or ctx's error when ctx ends
// first; the dialog is then taken down. It must not be called on the main goroutine.
func (c *guiConfirmer) Confirm(ctx context.Context, req engine.ConfirmRequest) (bool, error) {
	select {
	case c.slot <- struct{}{}:
	case <-ctx.Done():
		return false, ctx.Err()
	}
	defer func() { <-c.slot }()
	answer := make(chan bool, 1)
	var once sync.Once
	reply := func(ok bool) { once.Do(func() { answer <- ok }) }
	var dlg *confirmDialog
	c.sh.do(func() { dlg = c.show(req, reply) })
	select {
	case ok := <-answer:
		return ok, nil
	case <-ctx.Done():
		c.sh.do(func() {
			if dlg != nil {
				dlg.d.Hide()
			}
		})
		return false, ctx.Err()
	}
}

// show builds and shows the dialog; reply receives the answer once.
func (c *guiConfirmer) show(req engine.ConfirmRequest, reply func(bool)) *confirmDialog {
	th := c.sh.app.Settings().Theme()
	width := th.Size(sizeConfirmWidth)
	destructive := req.Danger == engine.DangerDestructive

	rows := box(kvLayout{})
	addRow := func(name, value string) {
		v := newParagraph(value, theme.ColorNameForeground, width-th.Size(sizeDrawerLabel)-2*th.Size(theme.SizeNamePadding))
		rows.Objects = append(rows.Objects, newTextLine(name, colorMuted, theme.SizeNameText, fyne.TextStyle{}), v)
	}
	objects := make([]string, 0, len(req.Objects))
	for _, o := range req.Objects {
		objects = append(objects, singular(c.sh.kindTitle(o.Kind))+" "+o.Name)
	}
	addRow("Objects", strings.Join(objects, "\n"))
	if len(req.Namespaces) > 0 {
		addRow("Namespace", strings.Join(req.Namespaces, ", "))
	}
	addRow("Context", req.Context)
	addRow("Requested by", originText(req.Origin))

	content := container.NewVBox(newParagraph(req.Message(), theme.ColorNameForeground, width), newSpace(theme.SizeNamePadding), rows)
	if destructive {
		content.Add(newSpace(theme.SizeNamePadding))
		content.Add(newParagraph("This cannot be undone.", colorStatusError, width))
	}

	label := req.Label
	if label == "" {
		label = "Confirm"
	}
	cd := &confirmDialog{}
	cd.confirm = widget.NewButton(label, nil)
	cd.confirm.Importance = widget.HighImportance
	if destructive {
		cd.confirm.Importance = widget.DangerImportance
		cd.confirm.SetIcon(theme.WarningIcon())
	}
	cd.cancel = widget.NewButton("Cancel", nil)
	title := label + "?"
	if req.Origin.IsAgent() {
		title = "An agent asks: " + title
	}
	cd.d = dialog.NewCustomWithoutButtons(title, content, c.sh.win)
	cd.d.SetButtons([]fyne.CanvasObject{cd.cancel, cd.confirm})
	cd.d.SetOnClosed(func() {
		reply(false) // closed any other way than Confirm: Escape, ctx ended
		if c.current == cd {
			c.current = nil
		}
	})
	cd.confirm.OnTapped = func() { reply(true); cd.d.Hide() }
	if c.armDelay > 0 {
		cd.confirm.Disable()
		time.AfterFunc(c.armDelay, func() { c.sh.do(cd.confirm.Enable) })
	}
	cd.cancel.OnTapped = func() { reply(false); cd.d.Hide() }
	c.current = cd
	cd.d.Show() // no button has the focus: Enter must not confirm a destructive command by accident
	return cd
}

// originText says who asked for a command, for the confirmation's "Requested by" row.
func originText(o engine.Origin) string {
	switch o {
	case engine.OriginGUI:
		return "You, in this window"
	case engine.OriginCLI:
		return "cogpit-cli"
	case engine.OriginExternalAgent:
		return "An external agent over MCP"
	case engine.OriginBuiltInAgent:
		return "Cogpit's built-in agent"
	}
	return string(o)
}
