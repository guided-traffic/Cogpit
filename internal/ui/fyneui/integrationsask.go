package fyneui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// The question of a new cluster's integrations (D-76; engine/integrationsask.go): while the UI
// state carries it (UIState.IntegrationsQuestion), a dialog over the window asks which
// integrations Cogpit uses for the shown cluster. It holds up nothing: the connect has finished
// and the views load behind it. The same rows as Cluster Settings › Integrations (a switch with the
// label, the status line in the status colours, live, and what it adds), prefilled from what
// applies now (the default for new clusters), the checkbox "Save as the default for new clusters"
// (the owner, 2026-10-17: "einen haken ..., der diese aktuellen Settings als default für zukünftige
// Abfragen anderer Server speichern soll"; off until ticked), and Later and Apply. Apply dispatches
// cluster.integrations.answer, Later cluster.integrations.later; the dialog closes when the engine
// drops the question, whoever answered it (an agent too). The switches are the dialog's own until
// Apply: nothing changes before.

const (
	integrationsAskSave = "Save as the default for new clusters"
	integrationsAskText = "Cogpit reads these tools where the cluster runs them, with your identity, and never writes to " +
		"them. Choose what it uses for this cluster; Cluster Settings › Integrations changes it later."
)

// integrationsQuestion is the window's dialog of the question.
type integrationsQuestion struct {
	sh  *shell
	ctx string // the context the open dialog asks about; "" while none is open

	d       *dialog.CustomDialog
	rows    []*integrationRow
	save    *widget.Check
	apply   *widget.Button
	later   *widget.Button
	errLine *textLine
	seq     int  // the latest status request; older answers are dropped
	closing bool // the dialog is being closed because the question ended: no Later
}

func newIntegrationsQuestion(sh *shell) *integrationsQuestion { return &integrationsQuestion{sh: sh} }

// stateChanged follows UIState.IntegrationsQuestion: opens the dialog, closes it, or asks about
// another cluster. Main goroutine.
func (q *integrationsQuestion) stateChanged() {
	want := q.sh.ui.IntegrationsQuestion
	if want == q.ctx {
		return
	}
	q.close()
	if want != "" {
		q.open(want)
	}
}

// close closes the dialog without answering.
func (q *integrationsQuestion) close() {
	if q.d != nil {
		q.closing = true
		q.d.Hide()
		q.closing = false
	}
	q.d, q.ctx, q.rows = nil, "", nil
}

// open shows the question for ctx.
func (q *integrationsQuestion) open(ctx string) {
	sh := q.sh
	q.ctx = ctx
	q.rows = nil
	objs := make([]fyne.CanvasObject, 0, 6+len(engine.Integrations))
	objs = append(objs, newParagraph(integrationsAskText, theme.ColorNameForeground, sh.app.Settings().Theme().Size(sizePickerWidth)),
		newSpace(theme.SizeNameInnerPadding))
	for _, in := range engine.Integrations {
		r := newIntegrationRow(in, nil) // the dialog's own switch until Apply
		r.check.SetChecked(sh.ui.IntegrationOn(ctx, in.Name))
		r.check.OnChanged = func(bool) { showAsked(r, r.last) }
		showAsked(r, engine.IntegrationStatus{Name: in.Name, State: engine.IntegrationChecking})
		q.rows = append(q.rows, r)
		objs = append(objs, r.box)
	}
	q.save = widget.NewCheck(integrationsAskSave, nil)
	q.errLine = newTextLine("", colorStatusError, theme.SizeNameCaptionText, fyne.TextStyle{})
	q.errLine.truncate = true
	q.errLine.Hide()
	objs = append(objs, newSpace(theme.SizeNameInnerPadding), newDivider(false), q.save, q.errLine)
	content := sized(sizePickerWidth, "", container.NewVBox(objs...))

	q.apply = widget.NewButton("Apply", q.answer)
	q.apply.Importance = widget.HighImportance
	q.later = widget.NewButton("Later", q.putOff)
	q.d = dialog.NewCustomWithoutButtons("Integrations for "+sh.identity(ctx).name, content, sh.win)
	q.d.SetButtons([]fyne.CanvasObject{q.later, q.apply})
	d := q.d
	q.d.SetOnClosed(func() {
		if !q.closing && q.d == d { // closed otherwise than by the question's end: Later
			q.putOff()
		}
	})
	q.d.Show()
	q.request()
}

// request reads the integrations' statuses of the asked cluster (cluster.settings.get).
func (q *integrationsQuestion) request() {
	if q.ctx == "" {
		return
	}
	q.seq++
	seq, ctx := q.seq, q.ctx
	q.sh.dispatch("cluster.settings.get", clusterSettingsContext{Context: ctx}, func(res any, err error) {
		var v engine.ClusterSettingsView
		if seq != q.seq || ctx != q.ctx || err != nil || decode(res, &v) != nil {
			return
		}
		for _, st := range v.Integrations {
			for _, r := range q.rows {
				if r.name == st.Name {
					showAsked(r, st)
				}
			}
		}
	})
}

// integrationsAskOff is the status of an integration off now that the dialog's switch turns on:
// nothing was looked for while it was off.
const integrationsAskOff = "Looked for once you apply"

// showAsked shows a status in the question's row as its own switch makes it: an integration off
// now and switched on in the dialog is looked for once the answer applies.
func showAsked(r *integrationRow, st engine.IntegrationStatus) {
	r.setStatus(st)
	if r.check.Checked && st.State == engine.IntegrationOff {
		r.status.SetText(integrationsAskOff)
		r.status.SetColor(colorMuted)
	}
}

// statusesChanged reads the statuses again when they may have changed for the asked cluster ("":
// any cluster's).
func (q *integrationsQuestion) statusesChanged(ctx string) {
	if q.ctx != "" && (ctx == "" || ctx == q.ctx) {
		q.request()
	}
}

// answer dispatches the switches as the cluster's choice (Apply).
func (q *integrationsQuestion) answer() {
	if q.ctx == "" {
		return
	}
	switches := map[string]bool{}
	for _, r := range q.rows {
		switches[string(r.name)] = r.check.Checked
	}
	ctx := q.ctx
	q.sh.dispatch("cluster.integrations.answer", map[string]any{contextParam: ctx, "integrations": switches,
		"saveAsDefault": q.save.Checked}, func(_ any, err error) { q.failed(ctx, err) })
}

// putOff dispatches Later.
func (q *integrationsQuestion) putOff() {
	if q.ctx == "" {
		return
	}
	ctx := q.ctx
	q.sh.dispatch("cluster.integrations.later", map[string]any{contextParam: ctx}, func(_ any, err error) { q.failed(ctx, err) })
}

// failed shows why an answer was refused under the switches; the dialog stays.
func (q *integrationsQuestion) failed(ctx string, err error) {
	if err == nil || ctx != q.ctx || q.errLine == nil {
		return
	}
	q.errLine.SetText(err.Error())
	q.errLine.Show()
	if q.d != nil && q.sh.ui.IntegrationsQuestion == ctx {
		q.d.Show() // Later by closing it: shown again while the question stands
	}
}
