package engine

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"
	"unicode/utf8"
)

// What the dock's logs and terminal tabs show, for agents (G17, dock.read; D-44). The lines live in
// the window, which streams them (a logs tab's view, a terminal's screen); the engine reads them
// through the DockOutputReader the window installs, page by page, and decides who may: a logs tab's
// lines are cluster data agents read like every other read (D-2), a terminal tab's (a local
// terminal, a pod shell, attach, a node shell) only once the user allowed it for that tab
// (DockTab.AgentRead, set by dock.allowAgentRead, which only the window runs), because a terminal
// shows what the user types and what any program prints, credentials included.
//
// A logs tab's lines are those its view shows: the pod and container it narrows to, its filter
// applied, each line with the resource names and timestamps as the tab shows them (D-31's format),
// without escape sequences or control characters (the view strips them as it reads); a terminal
// tab's are its screen's rows, top to bottom, without colours or escape sequences. The widget keeps
// no scrollback (fyne-io/terminal): what scrolled off the screen is gone, for the user too.

// Bounds of a page of dock output.
const (
	DefaultDockLines = 200
	MaxDockLines     = 2000
	// MaxDockLineBytes bounds one line; a longer one is cut and ends with DockLineCut.
	MaxDockLineBytes = 8 << 10
	// MaxDockPageBytes bounds a page's lines together; a page stops before the line that would pass
	// it (it has one line at least).
	MaxDockPageBytes = 256 << 10
	// dockReadWait bounds how long a read waits for the window.
	dockReadWait = 2 * time.Second
)

// DockLineCut ends a line cut at MaxDockLineBytes.
const DockLineCut = " …"

// Errors of dock.read.
var (
	// ErrTerminalOutputWithheld refuses an agent, or the CLI, a terminal tab the user has not
	// allowed agents to read (D-44).
	ErrTerminalOutputWithheld = errors.New("agents may not read this terminal tab: a terminal shows what the user types " +
		"and what programs print, credentials included, so only the user lets agents read one, with the tab's " +
		"\"Agents may read\" switch in Cogpit's window (D-44)")
	// ErrDockReadGUIOnly refuses an agent or the CLI the switch that lets agents read a terminal tab.
	ErrDockReadGUIOnly = errors.New("only the user, in Cogpit's own window, can let agents read a terminal tab; agents " +
		"and the CLI are refused, so that no agent allows itself (D-44)")
	// ErrNoDockWindow: the session has no window, so no tab shows any output (a headless engine).
	ErrNoDockWindow = errors.New("this Cogpit runs without a window, so no dock tab shows any output")
)

// DockOutputQuery asks for a page of a dock tab's lines. Lines are numbered in the order the tab
// shows them, ascending; the numbers are only for paging (a logs tab's start anywhere and skip the
// lines its filter hides; a terminal's rows are 1 to the screen's height).
type DockOutputQuery struct {
	// After: the oldest lines numbered above it; Before: the newest numbered below it; neither:
	// the newest lines. At most one is set.
	After  *int64
	Before *int64
	// Limit is the most lines a page has, 1 to MaxDockLines.
	Limit int
}

// DockLines is what a tab shows, line by line, oldest (topmost) first. The window implements it
// over its widgets and pages it with PageDockLines on its main goroutine.
type DockLines interface {
	Len() int
	// Number is line i's number, ascending in i.
	Number(i int) int64
	// Text is line i as the tab shows it.
	Text(i int) string
}

// DockPage is a page of a tab's lines (PageDockLines).
type DockPage struct {
	Lines []string `json:"lines"`
	// First and Last are the numbers of the first and last line returned (0 without lines): page
	// on with after: last, or back with before: first.
	First int64 `json:"first"`
	Last  int64 `json:"last"`
	// More is set when the tab shows lines beyond the page in its direction: older ones for a page
	// of the newest or before, newer ones for after.
	More bool `json:"more"`
	// Cut counts the lines cut at MaxDockLineBytes.
	Cut int `json:"cut,omitempty"`
}

// DockOutput is dock.read's result: a page of what a logs or terminal tab shows.
type DockOutput struct {
	Tab DockTab `json:"tab"`
	DockPage
	// Shown is the number of lines the tab shows: a logs tab's after its filter, a terminal's rows
	// down to the last one with text.
	Shown int `json:"shown"`
	// Held is the number of lines a logs tab holds before its filter; Dropped the older ones it no
	// longer holds (its scrollback's bound).
	Held    int   `json:"held,omitempty"`
	Dropped int64 `json:"dropped,omitempty"`
	// Status is a logs tab's status line as the window shows it ("1000 lines · stream ended").
	Status string `json:"status,omitempty"`
	// Ended is set once the stream or session ended; Opening while it has not opened yet (no lines).
	Ended   bool `json:"ended,omitempty"`
	Opening bool `json:"opening,omitempty"`
}

// DockOutputReader reads what the window shows in a logs or terminal dock tab. The window
// installs one (Session.SetDockOutputReader); a headless session has none. ReadDockOutput may be
// called on any goroutine; the reader does its work on the window's main goroutine and returns when
// ctx ends.
type DockOutputReader interface {
	ReadDockOutput(ctx context.Context, tab DockTab, q DockOutputQuery) (DockOutput, error)
}

// SetDockOutputReader installs the window's reader (nil removes it).
func (s *Session) SetDockOutputReader(r DockOutputReader) {
	s.mu.Lock()
	s.dockReader = r
	s.mu.Unlock()
}

func (s *Session) dockOutputReader() DockOutputReader {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dockReader
}

// CheckDockOutput is dock.read's Enabled: a window must show the tabs.
func (s *Session) CheckDockOutput() error {
	if s.dockOutputReader() == nil {
		return ErrNoDockWindow
	}
	return nil
}

// IsTerminalTab reports whether a dock tab type is a terminal: a local terminal, a pod shell,
// attach or a node shell.
func IsTerminalTab(t DockType) bool {
	switch t {
	case DockTerminal, DockShell, DockAttach, DockNodeShell:
		return true
	}
	return false
}

// CheckDockQuery checks a query and returns it with its limit: DefaultDockLines for 0, at most
// MaxDockLines.
func CheckDockQuery(q DockOutputQuery) (DockOutputQuery, error) {
	switch {
	case q.After != nil && q.Before != nil:
		return q, errors.New("give after or before, not both")
	case q.Limit < 0:
		return q, fmt.Errorf("limit %d is negative", q.Limit)
	case q.Limit == 0:
		q.Limit = DefaultDockLines
	}
	q.Limit = min(q.Limit, MaxDockLines)
	return q, nil
}

// ReadDockOutput reads a page of what a logs or terminal tab shows, for the origin that dispatched
// the running command (OriginOf): a terminal tab only for the GUI, or once the user allowed agents
// to read it (D-44). The window reads its widgets; nothing is asked of a cluster.
func (s *Session) ReadDockOutput(ctx context.Context, id string, q DockOutputQuery) (DockOutput, error) {
	q, err := CheckDockQuery(q)
	if err != nil {
		return DockOutput{}, err
	}
	gui := OriginOf(ctx) == OriginGUI
	tab, err := s.readableDockTab(id, gui)
	if err != nil {
		return DockOutput{}, err
	}
	r := s.dockOutputReader()
	if r == nil {
		return DockOutput{}, ErrNoDockWindow
	}
	rctx, cancel := context.WithTimeout(ctx, dockReadWait)
	defer cancel()
	out, err := r.ReadDockOutput(rctx, tab, q)
	if err != nil {
		return DockOutput{}, err
	}
	// The user may have closed the tab, or switched agents off, while the window read it: then
	// nothing of it is returned.
	if tab, err = s.readableDockTab(id, gui); err != nil {
		return DockOutput{}, err
	}
	out.Tab = tab
	return out, nil
}

// readableDockTab returns an open logs or terminal tab that the origin may read.
func (s *Session) readableDockTab(id string, gui bool) (DockTab, error) {
	tab, ok := s.UIState().DockTab(id)
	if !ok {
		return DockTab{}, fmt.Errorf("no dock tab %q is open; ui_state lists the open ones (dock)", id)
	}
	switch {
	case tab.Type == DockLogs:
		return tab, nil
	case IsTerminalTab(tab.Type):
		if !gui && !tab.AgentRead {
			return DockTab{}, ErrTerminalOutputWithheld
		}
		return tab, nil
	}
	return DockTab{}, fmt.Errorf("dock tab %q is a %s tab; dock_read reads logs and terminal tabs%s", id, tab.Type, otherReader(tab.Type))
}

// otherReader names the command that reads a tab dock.read does not.
func otherReader(t DockType) string {
	switch t {
	case DockEdit, DockCreate, DockHelmUpgrade, DockHelmInstall:
		return ", editor_get reads its text"
	case DockAudit:
		return ", audit_list reads the audit log"
	case DockDiagnostics:
		return ", diagnostics_get reads the diagnostics"
	}
	return ""
}

// AllowAgentRead lets agents read a terminal tab's output, or stops them (dock.allowAgentRead, the
// window's switch on the tab). It returns the tab.
func (s *Session) AllowAgentRead(id string, allowed bool) (DockTab, error) {
	var (
		out DockTab
		err error
	)
	s.UpdateUI(func(u *UIState) {
		for i := range u.Dock {
			t := &u.Dock[i]
			if t.ID != id {
				continue
			}
			if !IsTerminalTab(t.Type) {
				err = fmt.Errorf("dock tab %q is a %s tab: agents read logs tabs without it, and only terminal tabs have the switch", id, t.Type)
				return
			}
			t.AgentRead = allowed
			out = *t
			return
		}
		err = fmt.Errorf("no dock tab %q is open", id)
	})
	return out, err
}

// PageDockLines returns a page of src as q asks (CheckDockQuery checked it): each line cut at
// MaxDockLineBytes, the page bounded by q.Limit lines and MaxDockPageBytes. Call it where src may
// be read (the window's main goroutine).
func PageDockLines(src DockLines, q DockOutputQuery) DockPage {
	n := src.Len()
	limit := max(q.Limit, 1)
	var from, to int // the page is src[from:to]
	if q.After != nil {
		from = sort.Search(n, func(i int) bool { return src.Number(i) > *q.After })
		to = from
	} else {
		to = n
		if q.Before != nil {
			to = sort.Search(n, func(i int) bool { return src.Number(i) >= *q.Before })
		}
		from = to
	}
	page := DockPage{Lines: []string{}}
	bytes := 0
	take := func(i int) bool {
		line, cut := cutDockLine(src.Text(i))
		if len(page.Lines) > 0 && (len(page.Lines) >= limit || bytes+len(line) > MaxDockPageBytes) {
			return false
		}
		bytes += len(line)
		if cut {
			page.Cut++
		}
		page.Lines = append(page.Lines, line)
		return true
	}
	if q.After != nil {
		for to < n && take(to) {
			to++
		}
		page.More = to < n
	} else {
		for from > 0 && take(from-1) {
			from--
		}
		reverse(page.Lines)
		page.More = from > 0
	}
	if len(page.Lines) > 0 {
		page.First, page.Last = src.Number(from), src.Number(to-1)
	}
	return page
}

// cutDockLine cuts a line at MaxDockLineBytes, on a rune's start, and reports whether it did.
func cutDockLine(s string) (string, bool) {
	if len(s) <= MaxDockLineBytes {
		return s, false
	}
	cut := MaxDockLineBytes - len(DockLineCut)
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + DockLineCut, true
}

func reverse(s []string) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}
