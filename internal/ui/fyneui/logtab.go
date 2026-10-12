package fyneui

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/ui/fyneui/termview"
)

// logTab is a logs dock tab: termview.Logs on Session.OpenLogs, with the tab's options in its
// toolbar and its pod and container selectors above it (ui-parity › Logs). The options and the
// source are engine state (DockTab.Logs, DockTab.Container): a toggle dispatches logs.setOptions,
// a selector logs.setSource, and the widgets follow the tab as the engine reports it, whoever
// changed it. A change of the stream's request (another pod's or container's tab, previous)
// opens the stream again; the other options, and a controlling resource's selection, change how
// the view shows the lines it holds (the engine narrows the merged stream itself).
type logTab struct {
	sh  *shell
	c   *dockContent
	ctx context.Context

	tab    engine.DockTab
	key    logStreamKey
	view   *termview.Logs // nil until the stream opened
	seq    int            // the newest open; an older one that arrives late is closed
	failed bool           // the view shows why the stream did not open
	older  olderState     // older lines (logtab_older.go)

	timestamps, previous, names, wrap, filter *widget.Check
	search                                    *widget.Entry
	controller, download                      *tipButton
	controls                                  []fyne.CanvasObject
	podPick, containerPick                    *widget.Button // the source selectors (logtab_sources.go)
	header                                    []fyne.CanvasObject
	syncing                                   bool // set while the widgets follow engine state
	searchTimer                               *time.Timer
	searchBusy                                int // search texts typed but not acknowledged yet
}

// logStreamKey is what a logs tab's stream depends on: when it changes, the stream opens again.
type logStreamKey struct {
	ref            model.ObjectRef
	container, pod string
	previous       bool
}

// streamKeyOf: a pod's tab's stream is its container's; a controlling resource's merged stream
// follows the tab's selection itself while it follows, and opens again for it with Previous, whose
// stream ends after the previous instances. Timestamps come with every stream.
func streamKeyOf(t engine.DockTab) logStreamKey {
	k := logStreamKey{previous: t.Logs.Previous}
	if t.Ref != nil {
		k.ref = *t.Ref
	}
	if k.ref.Kind == kindPods || t.Logs.Previous {
		k.container, k.pod = t.Container, t.Logs.Pod
	}
	return k
}

// logOptionsParams mirror logs.setOptions' parameters; options left nil keep their value.
type logOptionsParams struct {
	ID            string  `json:"id"`
	Previous      *bool   `json:"previous,omitempty"`
	Timestamps    *bool   `json:"timestamps,omitempty"`
	ResourceNames *bool   `json:"resourceNames,omitempty"`
	Wrap          *bool   `json:"wrap,omitempty"`
	Search        *string `json:"search,omitempty"`
	Filter        *bool   `json:"filter,omitempty"`
}

func newLogTab(ctx context.Context, sh *shell, c *dockContent, t engine.DockTab) *logTab {
	lt := &logTab{sh: sh, c: c, ctx: ctx, tab: t, key: streamKeyOf(t)}
	lt.older.seen = t.Logs.LoadOlder // requests before the tab was shown are not carried out
	lt.resetOlder()
	check := func(label string, set func(p *logOptionsParams, on bool)) *widget.Check {
		return widget.NewCheck(label, func(on bool) {
			if lt.syncing {
				return
			}
			p := logOptionsParams{ID: lt.tab.ID}
			set(&p, on)
			lt.setOptions(p)
		})
	}
	lt.timestamps = check("Timestamps", func(p *logOptionsParams, on bool) { p.Timestamps = &on })
	lt.previous = check("Previous container", func(p *logOptionsParams, on bool) { p.Previous = &on })
	lt.names = check("Resource names", func(p *logOptionsParams, on bool) { p.ResourceNames = &on })
	lt.wrap = check("Wrap", func(p *logOptionsParams, on bool) { p.Wrap = &on })
	lt.filter = check("Filter", func(p *logOptionsParams, on bool) { p.Filter = &on })
	lt.search = widget.NewEntry()
	lt.search.SetPlaceHolder("Search logs… ( / )")
	lt.search.SetIcon(theme.SearchIcon())
	lt.search.OnChanged = lt.searchChanged
	lt.controller = newTipButton(sh, theme.MoveUpIcon(), "Logs of every pod of the pod's controller", func() {
		sh.dispatch("logs.controller", dockParams{ID: lt.tab.ID}, nil)
	})
	lt.download = newTipButton(sh, theme.DownloadIcon(), "Download the lines this tab holds", lt.saveDialog)
	lt.controls = []fyne.CanvasObject{
		lt.controller, lt.timestamps, lt.previous, lt.names, lt.wrap,
		sized(sizeSearchWidth, "", lt.search), lt.filter, lt.download,
	}
	lt.podPick, lt.containerPick = lt.newPicker(true), lt.newPicker(false)
	lt.header = []fyne.CanvasObject{lt.podPick, lt.containerPick}
	lt.follow(t)
	return lt
}

// follow makes the widgets show the tab's options, without dispatching them back.
func (lt *logTab) follow(t engine.DockTab) {
	lt.tab = t
	lt.syncing = true
	lt.timestamps.SetChecked(t.Logs.Timestamps)
	lt.previous.SetChecked(t.Logs.Previous)
	lt.names.SetChecked(t.Logs.ResourceNames)
	lt.wrap.SetChecked(t.Logs.Wrap)
	lt.filter.SetChecked(t.Logs.Filter)
	if lt.searchBusy == 0 && lt.search.Text != t.Logs.Search {
		lt.search.SetText(t.Logs.Search)
	}
	lt.podPick.SetText("Pod: " + podLabel(t))
	lt.containerPick.SetText("Container: " + containerLabel(t))
	lt.syncing = false
	if workload := t.Ref != nil && t.Ref.Kind != kindPods; workload {
		lt.names.Show()
		lt.controller.Hide()
	} else {
		lt.names.Hide()
		lt.controller.Show()
		lt.checkController()
	}
	if lt.view != nil {
		lt.view.SetView(viewOf(t))
	}
}

// viewOf is how a tab's lines show: every stream carries the server's timestamps, shown or not;
// a controlling resource's tab shows the lines of the pods and containers it selects.
func viewOf(t engine.DockTab) termview.LogView {
	v := termview.LogView{
		Prefixed: t.Ref != nil && t.Ref.Kind != kindPods, Names: t.Logs.ResourceNames, Stamped: true, Timestamps: t.Logs.Timestamps,
		Wrap: t.Logs.Wrap, Search: t.Logs.Search, Filter: t.Logs.Filter,
	}
	if v.Prefixed {
		v.Pod, v.Container = t.Logs.Pod, t.Container
	}
	return v
}

// update follows a change of the tab in the engine's state: the stream opens again when its
// request changed; otherwise the view shows the lines it holds the new way.
func (lt *logTab) update(t engine.DockTab) {
	key := streamKeyOf(t)
	narrowed := selectionOf(lt.tab) != selectionOf(t)
	lt.follow(t)
	switch {
	case key != lt.key:
		lt.key = key
		lt.open()
	case narrowed && lt.older.complete:
		lt.older.complete = false // the containers it shows changed
		lt.updateHead()
	}
	lt.followOlder(t)
}

// selectionOf is what a tab's lines are narrowed to.
func selectionOf(t engine.DockTab) [2]string { return [2]string{t.Logs.Pod, t.Container} }

// open opens the tab's stream off the main goroutine and shows it in a new view, which replaces
// the one shown; until it is open the tab says so, and an error shows in the view.
func (lt *logTab) open() {
	lt.seq++
	seq, id, sh := lt.seq, lt.tab.ID, lt.sh
	go func() {
		defer sh.recoverTo("logs")
		stream, err := sh.eng.OpenLogs(lt.ctx, id)
		sh.do(func() {
			if lt.c.closed || seq != lt.seq {
				if stream != nil {
					go func() { _ = stream.Close() }() // the tab is gone or opened again; nobody to tell
				}
				return
			}
			if old := lt.c.closer; old != nil {
				go func() { _ = old.Close() }() // the stream the tab showed before
			}
			if err != nil {
				// The error shows as the tab's only line, below the toolbar, so that the option
				// that caused it (a previous container that does not exist) can be turned off.
				stream = io.NopCloser(strings.NewReader("[cogpit] could not open the logs: " + err.Error() + "\n"))
			}
			w := termview.NewLogsOn(stream, sh.do)
			w.SetHeader(lt.header...)
			w.SetControls(lt.controls...)
			w.SetOnSlash(lt.focusSearch)
			w.SetView(viewOf(lt.tab))
			lt.view, lt.c.closer, lt.failed = w, w, err != nil
			lt.resetOlder()
			lt.updateHead()
			lt.c.set(w)
			if lt.older.pending {
				lt.older.pending = false
				lt.loadOlder(time.Time{}, nil)
			}
		})
	}()
}

// setOptions dispatches a change of the tab's options; the widgets follow the engine's answer.
func (lt *logTab) setOptions(p logOptionsParams) {
	lt.sh.dispatch("logs.setOptions", p, func(_ any, err error) {
		lt.sh.showError(err)
		if err != nil {
			lt.follow(lt.tab) // back to what the engine holds
		}
	})
}

// searchChanged debounces typing: the search goes to the engine once the user pauses.
func (lt *logTab) searchChanged(text string) {
	if lt.syncing {
		return
	}
	if lt.searchTimer != nil && lt.searchTimer.Stop() {
		lt.searchBusy--
	}
	lt.searchBusy++
	id, sh := lt.tab.ID, lt.sh
	lt.searchTimer = time.AfterFunc(searchDebounce, func() {
		sh.dispatch("logs.setOptions", logOptionsParams{ID: id, Search: &text}, func(_ any, err error) {
			lt.searchBusy--
			sh.showError(err)
			if lt.searchBusy == 0 && lt.search.Text != lt.tab.Logs.Search {
				lt.follow(lt.tab)
			}
		})
	})
}

// focusSearch puts the keyboard focus into the search entry ("/").
func (lt *logTab) focusSearch() {
	if c := fyne.CurrentApp().Driver().CanvasForObject(lt.search); c != nil {
		c.Focus(lt.search)
	}
}

// checkController greys the controller button with the reason when the pod's tab cannot move to
// its controller's logs (a pod no resource controls). It asks the engine off the main goroutine.
func (lt *logTab) checkController() {
	id, sh, b := lt.tab.ID, lt.sh, lt.controller
	go func() {
		defer sh.recoverTo("logs")
		err := sh.commandEnabled("logs.controller", dockParams{ID: id})
		sh.do(func() {
			switch {
			case errors.Is(err, errUnavailable):
				b.Hide()
			case err != nil:
				b.tip = "Logs of the pod's controller: " + err.Error()
				b.Disable()
			default:
				b.tip = "Logs of every pod of the pod's controller"
				b.Enable()
			}
		})
	}()
}

// saveDialog asks where to save the lines the tab holds, as they show (names and timestamps as
// toggled, every line, not only those a filter shows), and writes them there.
func (lt *logTab) saveDialog() {
	if lt.view == nil {
		return
	}
	d := dialog.NewFileSave(func(w fyne.URIWriteCloser, err error) {
		if err != nil {
			lt.sh.showError(err)
			return
		}
		if w == nil {
			return // cancelled
		}
		lines := lt.view.ShownLines()
		sh := lt.sh
		go func() {
			defer sh.recoverTo("logs download")
			err := saveLogs(w, lines)
			sh.do(func() {
				if err != nil {
					sh.toast("error", "Could not save the logs: "+err.Error())
					return
				}
				sh.toast("info", "Saved "+formatCount(int64(len(lines)))+" lines to "+w.URI().Name())
			})
		}()
	}, lt.sh.win)
	d.SetFileName(logFileName(lt.tab.Title))
	d.Show()
}

// saveLogs writes the lines to w and closes it.
func saveLogs(w io.WriteCloser, lines []string) error {
	err := termview.WriteLines(w, lines)
	if cerr := w.Close(); err == nil {
		err = cerr
	}
	return err
}

// logFileName is the file name a download proposes: the tab's title, made safe for a file
// system, ".log".
func logFileName(title string) string {
	name := strings.TrimSpace(strings.TrimPrefix(title, "Logs:"))
	name = strings.Map(func(r rune) rune {
		switch {
		case r == '/' || r == '\\' || r == ':' || r == ' ':
			return '_'
		case r < 0x20 || r == 0x7f:
			return -1
		}
		return r
	}, name)
	if name == "" {
		name = "logs"
	}
	return name + ".log"
}

// focusLogSearch focuses the active logs tab's search ("/" when nothing has the focus).
func (d *dock) focusLogSearch() {
	if c, ok := d.contents[d.active]; ok && c.logs != nil && d.open() {
		c.logs.focusSearch()
	}
}
