package fyneui

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/ui/fyneui/yamlview"
)

// actionSpec is one button of the drawer's toolbar. The toolbar is data: kinds map to actions,
// actions map to engine commands, and the engine decides whether each one is enabled.
type actionSpec struct {
	label     string
	icon      fyne.Resource
	command   string
	container bool   // the command takes a container; with several, the user picks one
	danger    bool   // styled as dangerous
	done      string // toast after success; %s is the object, "pod web-0"
}

// kindActions are the kind-specific actions, in Lens's order; every kind also gets Delete.
var kindActions = map[string][]actionSpec{
	kindPods: {
		{label: "Logs", icon: theme.DocumentIcon(), command: "pod.logs", container: true},
		{label: "Shell", icon: theme.ComputerIcon(), command: "pod.shell", container: true},
	},
	"deployments": {
		{label: "Restart", icon: theme.ViewRefreshIcon(), command: "deployment.restart", done: "Restarted %s"},
	},
}

var deleteAction = actionSpec{label: "Delete", icon: theme.DeleteIcon(), command: "object.delete", danger: true, done: "Deleted %s"}

func actionsFor(kind string) []actionSpec {
	return append(append([]actionSpec(nil), kindActions[kind]...), deleteAction)
}

// actionState is an action evaluated for one object, off the main goroutine.
type actionState struct {
	spec       actionSpec
	reason     string // why the action is disabled; empty when it is enabled
	hidden     bool   // this build has no such command
	containers []string
}

func (a actionState) params(ref model.ObjectRef, container string) any {
	if a.spec.container {
		return containerParams{Ref: ref, Container: container}
	}
	return refParams{Ref: ref}
}

// drawer is the detail drawer: the object's kind, name and status, an action toolbar, and the
// Details, Events and YAML tabs. It refreshes when the engine reports the object changed.
type drawer struct {
	sh *shell

	shown       *model.ObjectRef
	last        *engine.DetailModel
	busy, dirty bool

	kind     *textLine
	title    *textLine
	status   *pill
	missing  *textLine
	actions  *fyne.Container
	reasons  *textLine
	tabs     *container.AppTabs
	events   *container.TabItem
	details  *fyne.Container
	eventBox *fyne.Container
	yaml     *yamlview.View
	yamlText string
	root     fyne.CanvasObject
}

func newDrawer(sh *shell) *drawer {
	d := &drawer{sh: sh}
	d.kind = newTextLine("", colorMuted, theme.SizeNameCaptionText, fyne.TextStyle{})
	d.title = newTextLine("", theme.ColorNameForeground, theme.SizeNameSubHeadingText, fyne.TextStyle{Bold: true})
	d.title.truncate = true
	d.status = newPill("", colorMuted, true)
	d.status.Hide()
	d.missing = newTextLine("This object is no longer in the cluster.", colorStatusWarning, theme.SizeNameCaptionText, fyne.TextStyle{})
	d.missing.Hide()
	d.actions = container.NewHBox()
	d.reasons = newTextLine("", colorMuted, theme.SizeNameCaptionText, fyne.TextStyle{})
	d.reasons.truncate = true

	closeB := widget.NewButtonWithIcon("", theme.CancelIcon(), sh.closeDrawer)
	closeB.Importance = widget.LowImportance

	d.details = container.NewVBox()
	d.eventBox = container.NewVBox()
	d.yaml = yamlview.New()
	d.yaml.SetReadOnly(true)
	d.events = container.NewTabItem("Events", container.NewVScroll(inset(theme.SizeNameInnerPadding, theme.SizeNameInnerPadding, d.eventBox)))
	d.tabs = container.NewAppTabs(
		container.NewTabItem("Details", container.NewVScroll(inset(theme.SizeNameInnerPadding, theme.SizeNameInnerPadding, d.details))),
		d.events,
		container.NewTabItem("YAML", d.yaml),
	)

	titleRow := container.NewBorder(nil, nil, nil, container.NewHBox(container.NewCenter(d.status), closeB), d.title)
	toolbar := container.NewBorder(nil, nil, d.actions, nil, d.reasons)
	header := container.NewStack(newSurface(colorDrawerHeader),
		inset(theme.SizeNameInnerPadding, theme.SizeNameInnerPadding,
			container.NewVBox(d.kind, titleRow, toolbar, d.missing)))
	d.root = container.NewStack(newSurface(colorDrawer),
		container.NewBorder(container.NewVBox(header, newDivider(false)), nil, nil, nil, d.tabs))
	return d
}

// request fetches the drawer object's detail off the main goroutine and shows it. Requests that
// arrive while one runs collapse into one more fetch afterwards.
func (d *drawer) request() {
	ref := d.sh.ui.Drawer
	if ref == nil {
		return
	}
	if d.busy {
		d.dirty = true
		return
	}
	d.busy = true
	r := *ref
	if d.shown == nil || *d.shown != r {
		d.showLoading(r)
	}
	go func() {
		var (
			m    *engine.DetailModel
			err  error
			acts []actionState
		)
		func() {
			defer func() {
				if p := recover(); p != nil {
					err = fmt.Errorf("internal error: %v", p)
				}
			}()
			m, err = d.sh.eng.Detail(r)
			acts = d.evaluate(r, m)
		}()
		d.sh.do(func() {
			d.busy = false
			if cur := d.sh.ui.Drawer; cur != nil && *cur == r {
				d.show(r, m, err, acts)
			}
			if d.dirty {
				d.dirty = false
				d.request()
			}
		})
	}()
}

// evaluate asks the engine which actions are enabled for the object. Off the main goroutine.
func (d *drawer) evaluate(ref model.ObjectRef, m *engine.DetailModel) []actionState {
	var containers []string
	if m != nil {
		containers = m.Containers
	}
	specs := actionsFor(ref.Kind)
	out := make([]actionState, 0, len(specs))
	for _, spec := range specs {
		a := actionState{spec: spec, containers: containers}
		first := ""
		if len(containers) > 0 {
			first = containers[0]
		}
		if err := d.sh.commandEnabled(spec.command, a.params(ref, first)); err != nil {
			if err == errUnavailable {
				a.hidden = true
			} else {
				a.reason = err.Error()
			}
		}
		out = append(out, a)
	}
	return out
}

// showLoading retargets the drawer at a new object while its detail loads.
func (d *drawer) showLoading(ref model.ObjectRef) {
	r := ref
	d.shown = &r
	d.last = nil
	d.kind.SetText(singular(d.sh.kindTitle(ref.Kind)))
	d.title.SetText(ref.Name)
	d.status.Hide()
	d.missing.Hide()
	d.actions.Objects = nil
	d.actions.Refresh()
	d.reasons.SetText("")
	d.details.Objects = []fyne.CanvasObject{newTextLine("Loading…", colorMuted, theme.SizeNameText, fyne.TextStyle{})}
	d.details.Refresh()
	d.eventBox.Objects = nil
	d.eventBox.Refresh()
	d.setYAML("")
	d.tabs.SelectIndex(0)
}

// show renders a detail model. Unchanged sections are not rebuilt, so that a churning object does
// not flicker or lose the scroll position.
func (d *drawer) show(ref model.ObjectRef, m *engine.DetailModel, err error, acts []actionState) {
	if d.shown == nil || *d.shown != ref {
		d.showLoading(ref)
	}
	if err != nil || m == nil {
		msg := "No details."
		if err != nil {
			msg = "Could not load the details: " + err.Error()
		}
		d.details.Objects = []fyne.CanvasObject{newTextLine(msg, colorStatusError, theme.SizeNameText, fyne.TextStyle{})}
		d.details.Refresh()
		return
	}
	kind, name := splitTitle(m.Title)
	if kind == "" {
		kind = singular(d.sh.kindTitle(ref.Kind))
	}
	d.kind.SetText(kind)
	d.title.SetText(name)
	if m.Status != model.StatusNone {
		d.status.Set(statusLabel(m), statusColorName(m.Status))
		d.status.Show()
	} else {
		d.status.Hide()
	}
	if m.Missing {
		d.missing.Show()
	} else {
		d.missing.Hide()
	}
	d.showActions(ref, acts)

	if d.last == nil || !reflect.DeepEqual(d.last.Sections, m.Sections) {
		d.details.Objects = d.buildDetails(m.Sections)
		d.details.Refresh()
	}
	if d.last == nil || !reflect.DeepEqual(d.last.Events, m.Events) {
		d.eventBox.Objects = d.buildEvents(m.Events)
		d.eventBox.Refresh()
		d.events.Text = "Events"
		if n := len(m.Events); n > 0 {
			d.events.Text = fmt.Sprintf("Events (%d)", n)
		}
		d.tabs.Refresh()
	}
	d.setYAML(m.YAML)
	d.last = m
}

// clear forgets the shown object when the drawer closes.
func (d *drawer) clear() {
	d.shown, d.last, d.dirty = nil, nil, false
}

// setYAML hands the YAML to the viewer only when it changed: SetText scrolls to the top.
func (d *drawer) setYAML(text string) {
	if text == d.yamlText {
		return
	}
	d.yamlText = text
	d.yaml.SetText(text)
	d.yaml.Refresh()
}

// showActions rebuilds the toolbar: enabled buttons dispatch, disabled ones say why below.
func (d *drawer) showActions(ref model.ObjectRef, acts []actionState) {
	d.actions.Objects = d.actions.Objects[:0]
	var reasons []string
	byReason := map[string][]string{}
	for _, a := range acts {
		if a.hidden {
			continue
		}
		b := widget.NewButtonWithIcon(a.spec.label, a.spec.icon, nil)
		b.OnTapped = func() { d.run(ref, a, b) }
		if a.spec.danger {
			b.Importance = widget.DangerImportance
		}
		if a.reason != "" {
			b.Disable()
			if _, seen := byReason[a.reason]; !seen {
				reasons = append(reasons, a.reason)
			}
			byReason[a.reason] = append(byReason[a.reason], a.spec.label)
		}
		d.actions.Add(b)
	}
	d.actions.Refresh()
	parts := make([]string, 0, len(reasons))
	for _, r := range reasons {
		parts = append(parts, strings.Join(byReason[r], ", ")+": "+r)
	}
	d.reasons.SetText(strings.Join(parts, " · "))
}

// run dispatches an action. A container command on a pod with several containers asks which.
func (d *drawer) run(ref model.ObjectRef, a actionState, from fyne.CanvasObject) {
	dispatch := func(container string) {
		d.sh.dispatch(a.spec.command, a.params(ref, container), func(_ any, err error) {
			if err != nil {
				d.sh.showError(err)
				return
			}
			if a.spec.done != "" {
				d.sh.toast("info", fmt.Sprintf(a.spec.done, strings.ToLower(singular(d.sh.kindTitle(ref.Kind)))+" "+ref.Name))
			}
		})
	}
	if !a.spec.container || len(a.containers) <= 1 {
		first := ""
		if len(a.containers) == 1 {
			first = a.containers[0]
		}
		dispatch(first)
		return
	}
	items := make([]*fyne.MenuItem, 0, len(a.containers))
	for _, c := range a.containers {
		items = append(items, fyne.NewMenuItem(c, func() { dispatch(c) }))
	}
	if cnv := fyne.CurrentApp().Driver().CanvasForObject(from); cnv != nil {
		widget.ShowPopUpMenuAtRelativePosition(fyne.NewMenu("Container", items...), cnv,
			fyne.NewPos(0, from.Size().Height), from)
	}
}

// buildDetails renders the sections: a title, then name and value pairs.
func (d *drawer) buildDetails(sections []model.DetailSection) []fyne.CanvasObject {
	out := make([]fyne.CanvasObject, 0, 2*len(sections))
	for i, sec := range sections {
		if i > 0 {
			out = append(out, newSpace(theme.SizeNameInnerPadding))
		}
		out = append(out, newTextLine(sec.Title, theme.ColorNameForeground, theme.SizeNameText, fyne.TextStyle{Bold: true}), newDivider(false))
		grid := box(kvLayout{})
		for _, it := range sec.Items {
			name := newTextLine(it.Name, colorMuted, theme.SizeNameText, fyne.TextStyle{})
			grid.Objects = append(grid.Objects, name, detailValue(it))
		}
		out = append(out, grid)
	}
	return out
}

// detailValue renders one value: a status pill, chips for labels, one line per line otherwise.
func detailValue(it model.DetailItem) fyne.CanvasObject {
	if it.Status != model.StatusNone {
		return container.NewHBox(newPill(it.Value, statusColorName(it.Status), true))
	}
	lines := strings.Split(it.Value, "\n")
	chips := it.Name == "Labels"
	if len(lines) == 1 && !chips {
		t := newTextLine(it.Value, theme.ColorNameForeground, theme.SizeNameText, fyne.TextStyle{})
		t.truncate = true
		return t
	}
	v := container.NewVBox()
	for _, l := range lines {
		if chips {
			v.Add(container.NewHBox(newPill(l, theme.ColorNameForeground, false)))
			continue
		}
		t := newTextLine(l, theme.ColorNameForeground, theme.SizeNameText, fyne.TextStyle{})
		t.truncate = true
		v.Add(t)
	}
	return v
}

// buildEvents renders the object's events, newest first, as Lens's drawer does.
func (d *drawer) buildEvents(events []engine.EventItem) []fyne.CanvasObject {
	if len(events) == 0 {
		return []fyne.CanvasObject{newTextLine("No events.", colorMuted, theme.SizeNameText, fyne.TextStyle{})}
	}
	now := d.sh.now()
	out := make([]fyne.CanvasObject, 0, 2*len(events))
	for _, e := range events {
		st := e.Status
		if st == model.StatusNone {
			st = model.StatusOK
			if e.Type == "Warning" {
				st = model.StatusWarning
			}
		}
		var meta []string
		if e.Source != "" {
			meta = append(meta, e.Source)
		}
		if e.Count > 1 {
			meta = append(meta, fmt.Sprintf("×%d", e.Count))
		}
		if e.LastSeen > 0 {
			meta = append(meta, model.FormatAge(now.Sub(time.Unix(e.LastSeen, 0)))+" ago")
		}
		head := container.NewBorder(nil, nil,
			container.NewHBox(newPill(e.Type, statusColorName(st), true),
				newTextLine(e.Reason, theme.ColorNameForeground, theme.SizeNameText, fyne.TextStyle{Bold: true})),
			newTextLine(strings.Join(meta, " · "), colorMuted, theme.SizeNameCaptionText, fyne.TextStyle{}))
		msg := widget.NewLabel(e.Message)
		msg.Wrapping = fyne.TextWrapWord
		out = append(out, container.NewVBox(head, msg), newDivider(false))
	}
	return out
}

// splitTitle splits "Pod: web-0" into "Pod" and "web-0".
func splitTitle(t string) (kind, name string) {
	if k, n, ok := strings.Cut(t, ": "); ok {
		return k, n
	}
	return "", t
}

// statusLabel is the drawer's status pill text: the Status item of the first section that has
// one, else the token itself.
func statusLabel(m *engine.DetailModel) string {
	for _, s := range m.Sections {
		for _, it := range s.Items {
			if it.Name == "Status" && it.Value != "" {
				return it.Value
			}
		}
	}
	return string(m.Status)
}

// singular turns a view title into an object noun: "Pods" → "Pod", "Ingresses" → "Ingress".
func singular(title string) string {
	switch {
	case strings.HasSuffix(title, "sses"):
		return strings.TrimSuffix(title, "es")
	case strings.HasSuffix(title, "ies"):
		return strings.TrimSuffix(title, "ies") + "y"
	case strings.HasSuffix(title, "s"):
		return strings.TrimSuffix(title, "s")
	}
	return title
}

// kvLayout lays out name and value pairs in two columns: names in the theme's label width,
// values in the rest. Objects alternate name, value.
type kvLayout struct{}

func (kvLayout) dims() (nameW, gap float32) {
	th := fyne.CurrentApp().Settings().Theme()
	return th.Size(sizeDrawerLabel), th.Size(theme.SizeNamePadding) * 2
}

func (l kvLayout) MinSize(objs []fyne.CanvasObject) fyne.Size {
	nameW, gap := l.dims()
	var h, valueW float32
	for i := 0; i+1 < len(objs); i += 2 {
		n, v := objs[i].MinSize(), objs[i+1].MinSize()
		h += fyne.Max(n.Height, v.Height) + gap/2
		valueW = fyne.Max(valueW, v.Width)
	}
	return fyne.NewSize(nameW+gap+valueW, h)
}

func (l kvLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	nameW, gap := l.dims()
	y := float32(0)
	for i := 0; i+1 < len(objs); i += 2 {
		n, v := objs[i].MinSize(), objs[i+1].MinSize()
		rowH := fyne.Max(n.Height, v.Height)
		objs[i].Move(fyne.NewPos(0, y))
		objs[i].Resize(fyne.NewSize(nameW, n.Height))
		objs[i+1].Move(fyne.NewPos(nameW+gap, y))
		objs[i+1].Resize(fyne.NewSize(fyne.Max(size.Width-nameW-gap, 0), v.Height))
		y += rowH + gap/2
	}
}

// newSpace is vertical empty space of a theme size.
func newSpace(size fyne.ThemeSizeName) fyne.CanvasObject {
	return sized("", size)
}

// newHGap is horizontal empty space of a theme size.
func newHGap(size fyne.ThemeSizeName) fyne.CanvasObject {
	return sized(size, "")
}
