package fyneui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// A job tab (engine/jobs.go; A91): a background job's progress in the dock, as the engine's state
// holds it (DockTab.Job), so that the window shows what agents read with ui_state. A header with the
// job's state and outcome, then a row per step: its state's icon, what it acts on, and its detail
// ("Waiting for its controller: Reconciling: …", "Ready: stored artifact …"), in the state's colour.
// Closing the tab stops the job (dock.close); the tab has no controls of its own. The look is
// Cogpit's own: Lens has no such tab.

// jobStateStyles are a job's and a step's states: their label, colour and icon (by name, resolved
// when drawn: a theme icon asked for at package initialisation asks Fyne for an app none has
// started yet, TestStartUpLogsNoFyneError).
var jobStateStyles = map[engine.JobState]struct {
	label string
	color fyne.ThemeColorName
	icon  fyne.ThemeIconName
}{
	engine.JobPending:   {"Waiting", colorMuted, theme.IconNameRadioButton},
	engine.JobRunning:   {"Running", colorAccent, theme.IconNameViewRefresh},
	engine.JobDone:      {"Done", colorStatusOK, theme.IconNameConfirm},
	engine.JobFailed:    {jobFailedLabel, colorStatusError, theme.IconNameError},
	engine.JobCancelled: {"Stopped", colorMuted, theme.IconNameMediaStop},
}

// jobFailedLabel is a failed job's or step's state.
const jobFailedLabel = "Failed"

// jobStyle returns a state's style; an unknown state shows as running, with its own name.
func jobStyle(st engine.JobState) (label string, c fyne.ThemeColorName, icon fyne.Resource) {
	s, ok := jobStateStyles[st]
	if !ok {
		s = jobStateStyles[engine.JobRunning]
		s.label = string(st)
	}
	return s.label, s.color, themedIcon(theme.Icon(s.icon), s.color)
}

// jobTab is a job tab's content.
type jobTab struct {
	sh      *shell
	root    *fyne.Container
	icon    *widget.Icon
	state   *textLine
	times   *textLine
	outcome *textLine
	steps   *fyne.Container
	rows    []*jobStepRow
}

// jobStepRow is one step's row.
type jobStepRow struct {
	root   *fyne.Container
	icon   *widget.Icon
	text   *textLine
	detail *textLine
}

func newJobTab(sh *shell, t engine.DockTab) *jobTab {
	j := &jobTab{sh: sh, icon: widget.NewIcon(nil)}
	j.state = newTextLine("", colorAccent, theme.SizeNameText, fyne.TextStyle{Bold: true})
	j.times = newTextLine("", colorMuted, theme.SizeNameText, fyne.TextStyle{})
	j.outcome = newTextLine("", colorMuted, theme.SizeNameText, fyne.TextStyle{})
	j.outcome.truncate = true
	j.steps = container.NewVBox()
	head := container.NewBorder(nil, nil, container.NewHBox(j.icon, j.state, j.times), nil, j.outcome)
	j.root = container.NewBorder(
		container.NewVBox(inset(theme.SizeNameInnerPadding, theme.SizeNamePadding, head), newDivider(false)),
		nil, nil, nil, container.NewVScroll(inset(theme.SizeNameInnerPadding, theme.SizeNamePadding, j.steps)))
	j.update(t)
	return j
}

// update shows the job as the engine's state holds it.
func (j *jobTab) update(t engine.DockTab) {
	job := t.Job
	if job == nil {
		job = &engine.JobTab{State: engine.JobRunning}
	}
	label, c, icon := jobStyle(job.State)
	j.icon.SetResource(icon)
	j.state.SetText(label)
	j.state.SetColor(c)
	tf := j.sh.timeFormat()
	times := ""
	if !job.Started.IsZero() {
		times = "started " + tf.Format(job.Started)
	}
	if !job.Ended.IsZero() {
		times += ", ended " + tf.FormatClock(job.Ended)
	}
	j.times.SetText(times)
	j.outcome.SetText(job.Outcome)
	if job.State == engine.JobFailed {
		j.outcome.SetColor(colorStatusError)
	} else {
		j.outcome.SetColor(colorMuted)
	}
	for len(j.rows) < len(job.Steps) {
		r := &jobStepRow{icon: widget.NewIcon(nil)}
		r.text = newTextLine("", theme.ColorNameForeground, theme.SizeNameText, fyne.TextStyle{})
		r.detail = newTextLine("", colorMuted, theme.SizeNameText, fyne.TextStyle{})
		r.detail.truncate = true
		r.root = container.NewBorder(nil, nil, container.NewHBox(r.icon, r.text), nil, r.detail)
		j.rows = append(j.rows, r)
		j.steps.Add(r.root)
	}
	for i, r := range j.rows {
		if i >= len(job.Steps) {
			r.root.Hide()
			continue
		}
		st := job.Steps[i]
		_, c, icon := jobStyle(st.State)
		r.icon.SetResource(icon)
		r.text.SetText(st.Text)
		r.detail.SetText(st.Detail)
		r.detail.SetColor(c)
		if st.State == engine.JobPending || st.State == engine.JobDone {
			r.detail.SetColor(colorMuted)
		}
		r.root.Show()
	}
}

// remote and close make the job tab a dockEditor; it has nothing to follow or stop.
func (j *jobTab) remote(uint64) {}
func (j *jobTab) close()        {}
