package fyneui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// A logs tab's pod and container selectors (ui-parity › Logs, "Pod and container selectors"):
// two buttons above the toolbar naming what the tab shows. Tapped, a selector asks the engine
// what the tab can show (logs.sources, from the cache) and offers it in a menu; a choice dispatches
// logs.setSource, and the buttons follow the tab as the engine reports it. Lens's log tab has a
// pod and a container drop-down (its documentation's "View logs"); their place, the "All pods"
// and "All containers" entries and the init containers under a heading of their own are Cogpit's
// (inferred).

// Labels of the selectors' "every pod" and "every container" entries.
const (
	allPodsLabel       = "All pods"
	allContainersLabel = "All containers"
	initHeadingLabel   = "Init containers"
)

// maxSourceItems bounds the pods a selector's menu lists (Cogpit's default): a Daemon Set on a
// large cluster has a pod per node. logs.setSource takes any of them.
const maxSourceItems = 200

// logSourceParams mirror logs.setSource's parameters; nil keeps a part.
type logSourceParams struct {
	ID        string  `json:"id"`
	Pod       *string `json:"pod,omitempty"`
	Container *string `json:"container,omitempty"`
}

// newPicker returns the pod (pod) or container selector.
func (lt *logTab) newPicker(pod bool) *widget.Button {
	b := widget.NewButtonWithIcon("", theme.MenuDropDownIcon(), nil)
	b.IconPlacement, b.Importance = widget.ButtonIconTrailingText, widget.LowImportance
	b.OnTapped = func() { lt.showSources(b, pod) }
	return b
}

// podLabel names the pod a tab shows.
func podLabel(t engine.DockTab) string {
	switch {
	case t.Ref == nil:
		return ""
	case t.Ref.Kind == kindPods:
		return t.Ref.Name
	case t.Logs.Pod == "":
		return allPodsLabel
	}
	return t.Logs.Pod
}

// containerLabel names the container a tab shows.
func containerLabel(t engine.DockTab) string {
	if t.Container == "" && (t.Ref == nil || t.Ref.Kind != kindPods) {
		return allContainersLabel
	}
	return t.Container
}

// showSources asks the engine what the tab can show and offers it in a menu under the selector.
func (lt *logTab) showSources(from *widget.Button, pod bool) {
	sh := lt.sh
	sh.dispatch("logs.sources", dockParams{ID: lt.tab.ID}, func(res any, err error) {
		var src engine.LogSources
		if err == nil {
			err = decode(res, &src)
		}
		if err != nil {
			sh.showError(err)
			return
		}
		menu := lt.sourceMenu(src, pod)
		if c := fyne.CurrentApp().Driver().CanvasForObject(from); c != nil && len(menu.Items) > 0 {
			m := widget.NewPopUpMenu(menu, c)
			m.ShowAtRelativePosition(fyne.NewPos(0, from.Size().Height), from)
		}
	})
}

// sourceMenu is a selector's menu: "All pods" or "All containers" for a controlling resource's
// tab, then the pods, or the containers with the init containers apart; what the tab shows is
// checked.
func (lt *logTab) sourceMenu(src engine.LogSources, pod bool) *fyne.Menu {
	t := lt.tab
	var items []*fyne.MenuItem
	choice := func(label, value string, checked bool) *fyne.MenuItem {
		it := fyne.NewMenuItem(label, func() { lt.setSource(pod, value) })
		it.Checked = checked
		return it
	}
	if pod {
		cur := t.Logs.Pod
		if t.Ref.Kind == kindPods {
			cur = t.Ref.Name
		}
		if src.AllPods {
			items = append(items, choice(allPodsLabel, "", cur == ""), fyne.NewMenuItemSeparator())
		}
		for i, p := range src.Pods {
			if i == maxSourceItems {
				more := fyne.NewMenuItem(fmt.Sprintf("… %d more pods", len(src.Pods)-i), nil)
				more.Disabled = true
				items = append(items, more)
				break
			}
			items = append(items, choice(p, p, p == cur))
		}
		return fyne.NewMenu("", items...)
	}
	cur := t.Container
	if src.AllPods {
		items = append(items, choice(allContainersLabel, "", cur == ""), fyne.NewMenuItemSeparator())
	}
	for _, c := range src.Containers {
		items = append(items, choice(c, c, c == cur))
	}
	if len(src.InitContainers) > 0 {
		heading := fyne.NewMenuItem(initHeadingLabel, nil)
		heading.Disabled = true
		items = append(items, fyne.NewMenuItemSeparator(), heading)
		for _, c := range src.InitContainers {
			items = append(items, choice(c, c, c == cur))
		}
	}
	return fyne.NewMenu("", items...)
}

// setSource dispatches a selector's choice; the buttons follow the engine's answer.
func (lt *logTab) setSource(pod bool, value string) {
	p := logSourceParams{ID: lt.tab.ID}
	if pod {
		p.Pod = &value
	} else {
		p.Container = &value
	}
	lt.sh.dispatch("logs.setSource", p, func(_ any, err error) { lt.sh.showError(err) })
}
