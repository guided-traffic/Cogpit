package fyneui

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/kubeconfig"
)

// The log of a connect (G2): the picker marks a context whose last connect failed, or whose
// connection Cogpit closed because its kubeconfig file went away, with a warning icon; the icon
// opens "Connection Updates: <context>", INFO and ERROR lines as Lens shows them (captured
// 2026-10-12). The engine keeps the lines (cluster.connectLog); what a credential plugin writes to
// its standard error is not among them (gap G2).

// logTimeZone is the zone the log's times are shown in; tests fix it for stable images.
var logTimeZone = time.Local

type connectLogView struct {
	Context string `json:"context"`
	Lines   []struct {
		Time  time.Time `json:"time"`
		Level string    `json:"level"`
		Text  string    `json:"text"`
	} `json:"lines"`
}

// connectLogDialog is the open "Connection Updates" dialog.
type connectLogDialog struct {
	d     *dialog.CustomDialog
	close *widget.Button
	view  connectLogView
}

// showConnectLog reads the log of contextID and shows it.
func (k *kubeconfigUI) showConnectLog(contextID string) {
	k.sh.dispatch("cluster.connectLog", map[string]string{"context": contextID}, func(res any, err error) {
		if err != nil {
			k.sh.showError(err)
			return
		}
		var v connectLogView
		if err := decode(res, &v); err != nil {
			k.sh.toast("error", "Unexpected connection log: "+err.Error())
			return
		}
		k.openConnectLog(contextID, v)
	})
}

func (k *kubeconfigUI) openConnectLog(contextID string, v connectLogView) {
	if k.connectLog != nil {
		k.connectLog.d.Hide()
	}
	th := k.sh.app.Settings().Theme()
	width := th.Size(sizePickerWidth)
	pad := th.Size(theme.SizeNamePadding)
	rows := container.NewVBox()
	for _, l := range v.Lines {
		fg := colorMuted
		switch l.Level {
		case engine.LogError:
			fg = colorStatusError
		case engine.LogWarn:
			fg = colorStatusWarning
		}
		prefix := newTextLine(l.Time.In(logTimeZone).Format("15:04:05")+"  "+padRight(l.Level, 5)+"  ", fg,
			theme.SizeNameText, fyne.TextStyle{Monospace: true})
		textW := width - prefix.MinSize().Width - 8*pad
		rows.Add(container.NewBorder(nil, nil, container.NewVBox(prefix), nil,
			newParagraph(l.Text, theme.ColorNameForeground, textW)))
	}
	if len(v.Lines) == 0 {
		rows.Add(newTextLine("No connection was attempted this session.", colorMuted, theme.SizeNameText, fyne.TextStyle{}))
	}
	bg := newSurface(theme.ColorNameInputBackground)
	bg.radius, bg.stroke = theme.SizeNameInputRadius, theme.ColorNameInputBorder
	content := sized(sizePickerWidth, sizePickerHeight, container.NewStack(bg, inset(theme.SizeNamePadding, theme.SizeNamePadding, container.NewVScroll(rows))))

	cl := &connectLogDialog{view: v}
	cl.d = dialog.NewCustomWithoutButtons("Connection Updates: "+kubeconfig.DisplayText(contextID, maxShownArg), content, k.sh.win)
	cl.close = widget.NewButton("Close", cl.d.Hide)
	cl.d.SetButtons([]fyne.CanvasObject{cl.close})
	cl.d.SetOnClosed(func() {
		if k.connectLog == cl {
			k.connectLog = nil
		}
	})
	k.connectLog = cl
	cl.d.Show()
}

func padRight(s string, n int) string {
	for len(s) < n {
		s += " "
	}
	return s
}
