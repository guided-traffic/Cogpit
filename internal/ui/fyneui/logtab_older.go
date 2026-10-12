package fyneui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/ui/fyneui/termview"
)

// Older lines in a logs tab (ui-parity › Logs, "Load older lines when scrolling up"): a row above
// the lines reads "Load older lines" while the tab may not hold its containers' whole logs.
// Selecting it dispatches logs.loadOlder, which an agent may call too; the tab follows the
// request in the engine's state (LogOptions.LoadOlder) by asking the engine for the lines before
// its oldest (Session.OlderLogs) and putting them on top, the rows in view staying in view. Lens
// loads them when the user scrolls up (its documentation's "View logs"); Cogpit asks only on that
// explicit row, never on scrolling or a timer (Cogpit's default, so that no request goes out the
// user did not ask for).

// Texts of the head row.
const (
	headLoadOlder = "Load older lines"
	headLoading   = "Loading older lines…"
	headBeginning = "Beginning of the log"
)

// olderLogsEngine is the part of the engine a logs tab loads older lines with: *engine.Session,
// and the tests' fake.
type olderLogsEngine interface {
	OlderLogs(ctx context.Context, tabID string, req engine.OlderRequest) (engine.OlderLines, error)
}

var _ olderLogsEngine = (*engine.Session)(nil)

// olderState is what the tab knows about its older lines.
type olderState struct {
	seen     int             // the engine's LoadOlder this tab has carried out
	pending  bool            // a request that came before the stream opened
	busy     bool            // a request runs
	err      string          // the last request's error
	complete bool            // the tab holds the whole log of every container it shows
	whole    map[string]bool // the sources whose whole log the tab holds
}

// resetOlder forgets what the tab knew about older lines: its stream opened again.
func (lt *logTab) resetOlder() {
	lt.older.busy, lt.older.err, lt.older.complete, lt.older.whole = false, "", false, map[string]bool{}
}

// requestOlder is the head row's action: the engine's state asks for older lines.
func (lt *logTab) requestOlder() {
	lt.sh.dispatch("logs.loadOlder", dockParams{ID: lt.tab.ID}, func(_ any, err error) { lt.sh.showError(err) })
}

// followOlder carries out the requests for older lines the engine's state shows and the tab has
// not carried out.
func (lt *logTab) followOlder(t engine.DockTab) {
	if t.Logs.LoadOlder == lt.older.seen {
		return
	}
	lt.older.seen = t.Logs.LoadOlder
	lt.loadOlder(time.Time{}, nil)
}

// loadOlder asks the engine for the lines before the oldest the tab holds (since: from then on,
// for a jump) and puts them on top, then calls then (on the main goroutine) if set. The view
// stops following, so that the lines stay where the user reads them.
func (lt *logTab) loadOlder(since time.Time, then func()) {
	eng, ok := lt.sh.eng.(olderLogsEngine)
	switch {
	case !ok || lt.failed:
		return
	case lt.view == nil:
		lt.older.pending = true
		return
	case lt.older.busy:
		return
	}
	view := lt.view
	room := view.Room()
	if room == 0 {
		lt.updateHead()
		if then != nil {
			then()
		}
		return
	}
	req := engine.OlderRequest{Since: since, Room: room}
	for _, h := range view.Held() {
		req.Held = append(req.Held, engine.HeldLog{Source: h.Source, Lines: h.Lines, Oldest: h.Oldest, Whole: lt.older.whole[h.Source]})
	}
	lt.older.busy, lt.older.err = true, ""
	lt.updateHead()
	id, sh := lt.tab.ID, lt.sh
	go func() {
		defer sh.recoverTo("older logs")
		res, err := eng.OlderLogs(lt.ctx, id, req)
		sh.do(func() {
			if lt.c.closed || view != lt.view {
				return // the stream opened again meanwhile: what the request held is gone
			}
			lt.older.busy = false
			if err != nil {
				lt.older.err = err.Error()
			} else {
				for _, w := range res.Whole {
					lt.older.whole[w] = true
				}
				view.SetFollow(false)
				view.Prepend(res.Lines)
				if since.IsZero() {
					lt.older.complete = res.Complete
				}
				if len(res.Notes) > 0 {
					sh.toast("warning", strings.Join(res.Notes, "\n"))
				}
			}
			lt.updateHead()
			if then != nil {
				then()
			}
		})
	}()
}

// updateHead shows the head row for the tab's state: loading, failed, the beginning of the log,
// full, or the "Load older lines" trigger.
func (lt *logTab) updateHead() {
	v := lt.view
	if v == nil {
		return
	}
	if _, ok := lt.sh.eng.(olderLogsEngine); !ok || lt.failed {
		v.SetHead("", termview.HeadInfo, nil)
		return
	}
	switch {
	case lt.older.busy:
		v.SetHead(headLoading, termview.HeadInfo, nil)
	case lt.older.err != "":
		v.SetHead("Could not load older lines: "+oneLineText(lt.older.err)+" · select to try again", termview.HeadError, lt.requestOlder)
	case lt.older.complete:
		v.SetHead(headBeginning, termview.HeadInfo, nil)
	case v.Room() == 0:
		v.SetHead(fmt.Sprintf("This tab holds %s lines, the most it keeps", formatCount(int64(v.Capacity()))), termview.HeadInfo, nil)
	default:
		v.SetHead(headLoadOlder, termview.HeadAction, lt.requestOlder)
	}
}

// oneLineText makes an error one line for the head row.
func oneLineText(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
