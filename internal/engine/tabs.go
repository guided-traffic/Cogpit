package engine

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"unicode"
	"unicode/utf8"

	"github.com/guided-traffic/cogpit/internal/model"
)

// The tab bar of cluster views (ui-parity › Layout, G5). Captured from Lens Desktop (2026-10-11): a
// single click in the navigator opens a preview tab, a double click (or ⇧↵) a fixed tab; a tab reads
// "<view> - <context>" with the cluster's icon and a close button; the preview tab's title is
// italic. From Lens's shortcuts page: ⌘W closes the tab, ⌃- and ⌃⇧- go back and forward in the
// tab's history, ⌘⌥→ and ⌘⌥← to the next and the previous tab, ⌘n to the n-th.
//
// Cogpit's defaults (inferred, not the owner's decision, 2026-10-13):
//   - one preview tab in all: opening a view in preview shows it in that tab, whatever cluster
//     the tab showed, and the view it showed before goes into its history; a fixed tab stays until
//     it is closed;
//   - opening a view that a fixed tab of the cluster shows activates that tab;
//   - a tab keeps its own namespace filter, search, status filter, sort, selection and drawer;
//     changing them does not fix a preview tab;
//   - a new tab opens right of the active one;
//   - closing the active tab shows the nearest tab of the same cluster, right first, else nothing:
//     the workspace stays empty until a view is opened (closing the last tab too);
//   - Cogpit holds one cluster connection at a time (architecture §6), so the tabs of other
//     clusters stay in the bar and activating one connects to its cluster as the navigator does,
//     then shows the tab's view again; while a connect runs no tab is active; a cluster connected
//     from the navigator or the hotbar shows its most recently used tab, else its default view in
//     the preview tab;
//   - a tab's history holds view keys, at most maxTabHistory each way; back and forward open the
//     view as ui.openView does (selection, drawer and search cleared, the namespace filter kept);
//   - ⌘9 is the last tab, as browsers have it; next and previous wrap around;
//   - the tabs of a cluster removed from Cogpit (cluster.remove, kubeconfig.remove) close with it
//     (Session.RemoveKubeconfigSource, contextlink.go).
//
// The engine keeps the tabs in step with the rest of the UI state in one place, UIState.syncTabs,
// which UpdateUI runs after every change: the active tab's fields follow the live ones, a view
// changed by any command (ui.navigate, a link) goes into the active tab's history, and a connect
// shows the cluster's tab. The tab commands (commands/tabs.go) call the Session methods below.

// ViewTab is one tab of the tab bar: a view of one cluster with its own state.
type ViewTab struct {
	ID string `json:"id"` // "tab-1"
	// Context is the cluster the tab belongs to; a preview tab's changes when it shows a view of
	// another cluster.
	Context string `json:"context"`
	// View is the kind or page key the tab shows ("pods", "workloads/overview").
	View string `json:"view"`
	// Title is "<view> - <context>", the context by its name without its kubeconfig file.
	Title string `json:"title"`
	// Preview marks the preview tab, which the next view opened in preview replaces; the others
	// are fixed.
	Preview bool `json:"preview,omitempty"`
	// The tab's own state. While the tab is active and its cluster connected, UIState's fields of
	// the same names are the live ones and these follow them; otherwise they are what the tab shows
	// when it is activated again.
	Namespaces []string          `json:"namespaces,omitempty"`
	Groups     []string          `json:"groups,omitempty"`
	Filter     string            `json:"filter,omitempty"`
	Sort       SortSpec          `json:"sort"`
	Selected   []model.ObjectRef `json:"selected,omitempty"`
	Drawer     *model.ObjectRef  `json:"drawer,omitempty"`
	// FilterOptions are the tab's search toggles (filteroptions.go); they stay when its view changes.
	FilterOptions FilterOptions `json:"filterOptions,omitzero"`
	// States is the tab's status filter (statusfilter.go); it ends when its view changes.
	States []string `json:"states,omitempty"`
	// Back and Forward are the tab's history: the views before and after the one it shows, the
	// nearest last (ui.tab.back, ui.tab.forward).
	Back    []string `json:"back,omitempty"`
	Forward []string `json:"forward,omitempty"`

	used uint64 // when the tab was last activated (UIState.tabClock)
}

// maxTabHistory bounds a tab's history each way (inferred: Lens's bound is not documented).
const maxTabHistory = 50

// Errors of the tab operations.
var (
	ErrUnknownTab = errors.New("no such tab")
	ErrNoTab      = errors.New("no tab is open")
)

// errTabsChanged reports that the tabs changed between reading and changing them.
var errTabsChanged = errors.New("the tabs changed meanwhile; try again")

func (t ViewTab) clone() ViewTab {
	out := t
	out.Namespaces = slices.Clone(t.Namespaces)
	out.Groups = slices.Clone(t.Groups)
	out.States = slices.Clone(t.States)
	out.Selected = slices.Clone(t.Selected)
	if t.Drawer != nil {
		d := *t.Drawer
		out.Drawer = &d
	}
	out.Back = slices.Clone(t.Back)
	out.Forward = slices.Clone(t.Forward)
	return out
}

func cloneTabs(tabs []ViewTab) []ViewTab {
	if tabs == nil {
		return nil
	}
	out := make([]ViewTab, len(tabs))
	for i, t := range tabs {
		out[i] = t.clone()
	}
	return out
}

// Tab returns the tab with the given id.
func (u UIState) Tab(id string) (ViewTab, bool) {
	if i := u.tabIndex(id); i >= 0 {
		return u.Tabs[i], true
	}
	return ViewTab{}, false
}

func (u *UIState) tabIndex(id string) int {
	if id == "" {
		return -1
	}
	return slices.IndexFunc(u.Tabs, func(t ViewTab) bool { return t.ID == id })
}

func (u *UIState) activeIndex() int { return u.tabIndex(u.ActiveTab) }

func (u *UIState) previewIndex() int {
	return slices.IndexFunc(u.Tabs, func(t ViewTab) bool { return t.Preview })
}

// fixedTabOf returns the index of a fixed tab of the cluster showing the view, or -1.
func (u *UIState) fixedTabOf(contextID, view string) int {
	return slices.IndexFunc(u.Tabs, func(t ViewTab) bool { return !t.Preview && t.Context == contextID && t.View == view })
}

// lastUsedTab returns the index of the cluster's most recently active tab, or -1.
func (u *UIState) lastUsedTab(contextID string) int {
	best := -1
	for i, t := range u.Tabs {
		if t.Context == contextID && (best < 0 || t.used > u.Tabs[best].used) {
			best = i
		}
	}
	return best
}

func (u *UIState) connected() bool { return u.Connection.State == Connected }

// touch marks a tab as the most recently active one.
func (u *UIState) touch(i int) {
	u.tabClock++
	u.Tabs[i].used = u.tabClock
}

// newTab inserts a tab right of the active one (at the end without one) and returns its index.
func (u *UIState) newTab(contextID, view string, preview bool) int {
	u.tabSeq++
	t := ViewTab{ID: "tab-" + strconv.FormatUint(u.tabSeq, 10), Context: contextID, View: view, Preview: preview}
	at := len(u.Tabs)
	if i := u.activeIndex(); i >= 0 {
		at = i + 1
	}
	u.Tabs = slices.Insert(u.Tabs, at, t)
	return at
}

// activateTab shows a tab: its id becomes ActiveTab and, when its cluster is the connected one, its
// state becomes the live state.
func (u *UIState) activateTab(i int) {
	u.ActiveTab = u.Tabs[i].ID
	u.touch(i)
	if u.connected() && u.Tabs[i].Context == u.Context && u.Tabs[i].View != "" {
		u.loadTab(i)
	}
}

// loadTab makes a tab's state the live state.
func (u *UIState) loadTab(i int) {
	t := u.Tabs[i]
	u.View, u.Filter, u.FilterOptions, u.Sort = t.View, t.Filter, t.FilterOptions, t.Sort
	if u.Sort.Column == "" {
		u.Sort = DefaultSort
	}
	u.Namespaces, u.Groups = slices.Clone(t.Namespaces), slices.Clone(t.Groups)
	u.States, u.statesView = slices.Clone(t.States), t.View
	u.Selected = slices.Clone(t.Selected)
	u.Drawer = nil
	if t.Drawer != nil {
		d := *t.Drawer
		u.Drawer = &d
	}
}

// saveTab makes the live state the tab's.
func (u *UIState) saveTab(i int) {
	t := &u.Tabs[i]
	t.View, t.Filter, t.FilterOptions, t.Sort = u.View, u.Filter, u.FilterOptions, u.Sort
	t.Namespaces, t.Groups = slices.Clone(u.Namespaces), slices.Clone(u.Groups)
	t.States = slices.Clone(u.States)
	t.Selected = slices.Clone(u.Selected)
	t.Drawer = nil
	if u.Drawer != nil {
		d := *u.Drawer
		t.Drawer = &d
	}
}

// pushHistory appends a view to one direction of a tab's history, dropping the oldest beyond
// maxTabHistory.
func pushHistory(h []string, view string) []string {
	h = append(h, view)
	if len(h) > maxTabHistory {
		h = slices.Delete(h, 0, len(h)-maxTabHistory)
	}
	return h
}

// showInPreview shows a view of the connected cluster in the preview tab, made or taken over from
// another cluster as needed, and activates it. The live state is left alone.
func (u *UIState) showInPreview(view string) {
	p := u.previewIndex()
	switch {
	case p < 0:
		p = u.newTab(u.Context, view, true)
	case u.Tabs[p].Context != u.Context:
		t := &u.Tabs[p]
		t.Context, t.Back, t.Forward = u.Context, nil, nil // the other cluster's history means nothing here
	case u.Tabs[p].View != "" && u.Tabs[p].View != view:
		t := &u.Tabs[p]
		t.Back, t.Forward = pushHistory(t.Back, t.View), nil
	}
	u.Tabs[p].View = view
	u.ActiveTab = u.Tabs[p].ID
	u.touch(p)
}

// syncTabs keeps the tabs in step with a change of the UI state from old to u, whichever command
// made it. UpdateUI runs it after every change; viewTitle names a view for the tabs' titles.
//
//   - A connect to another cluster than the active tab's leaves no tab active until it connected.
//   - Once a cluster is connected, the active tab of that cluster shows its own state again; without
//     one, the cluster's most recently used tab does; without any, the preview tab shows the
//     connected view.
//   - A view opened while no tab is active goes to the preview tab.
//   - A view changed within the active tab by a command that knows nothing of tabs (ui.navigate)
//     goes into the tab's history.
//   - While the active tab's cluster is connected, the tab's state follows the live state.
func (u *UIState) syncTabs(old UIState, viewTitle func(string) string) {
	if i := u.activeIndex(); u.ActiveTab != "" && (i < 0 || u.Tabs[i].Context != u.Context) {
		u.ActiveTab = ""
	}
	if u.connected() {
		becameConnected := old.Connection.State != Connected || old.Context != u.Context
		i := u.activeIndex()
		switch {
		case becameConnected && i >= 0 && u.Tabs[i].View != "":
			u.loadTab(i) // a reconnect: the tab's own view comes back
		case becameConnected && i < 0 && u.lastUsedTab(u.Context) >= 0:
			j := u.lastUsedTab(u.Context)
			u.ActiveTab = u.Tabs[j].ID
			u.touch(j)
			if u.Tabs[j].View != "" {
				u.loadTab(j)
			}
		case i < 0 && u.View != "":
			u.showInPreview(u.View)
		case i >= 0 && u.View != "" && u.Tabs[i].View != "" && u.View != u.Tabs[i].View:
			t := &u.Tabs[i]
			t.Back, t.Forward = pushHistory(t.Back, t.View), nil
		}
		if i := u.activeIndex(); i >= 0 {
			u.saveTab(i)
		}
	}
	for k := range u.Tabs {
		u.Tabs[k].Title = tabTitle(viewTitle, u.Tabs[k])
	}
}

// openTab opens a view of the connected cluster: in the preview tab, or in a new fixed tab when
// fixed is set. A fixed tab of the cluster that shows the view already is activated instead, with
// its own state (only ui.openView of the active tab's view opens it afresh, as it always did);
// asked for fixed, the preview tab showing the view becomes fixed.
func (u *UIState) openTab(view string, sort SortSpec, fixed bool) {
	p := u.previewIndex()
	switch i := u.fixedTabOf(u.Context, view); {
	case i >= 0 && (fixed || u.ActiveTab != u.Tabs[i].ID):
		if u.ActiveTab != u.Tabs[i].ID {
			u.activateTab(i)
		}
		return
	case i >= 0: // ui.openView of the active tab's view: opened afresh
	case fixed && p >= 0 && u.Tabs[p].Context == u.Context && u.Tabs[p].View == view:
		u.Tabs[p].Preview = false
		if u.ActiveTab != u.Tabs[p].ID {
			u.activateTab(p)
		}
		return
	case fixed:
		n := u.newTab(u.Context, view, false)
		u.ActiveTab = u.Tabs[n].ID
		u.touch(n)
	default:
		u.showInPreview(view)
	}
	u.View, u.Sort, u.Filter = view, sort, ""
	u.SetStates(nil) // opened afresh: no status filter, the active tab's own view too (statusfilter.go)
	u.Selected, u.Drawer = nil, nil
}

// closeTab removes a tab ("" is the active one). Closing the active tab shows the nearest tab of the
// same cluster, right first, or else leaves the workspace empty.
func (u *UIState) closeTab(id string) error {
	if id == "" {
		id = u.ActiveTab
	}
	i := u.tabIndex(id)
	if i < 0 {
		if id == "" {
			return ErrNoTab
		}
		return fmt.Errorf("%w %q", ErrUnknownTab, id)
	}
	closed := u.Tabs[i]
	u.Tabs = slices.Delete(u.Tabs, i, i+1)
	if u.ActiveTab != id {
		return nil
	}
	u.ActiveTab = ""
	next := -1
	for j := i; j < len(u.Tabs) && next < 0; j++ {
		if u.Tabs[j].Context == closed.Context {
			next = j
		}
	}
	for j := i - 1; j >= 0 && next < 0; j-- {
		if u.Tabs[j].Context == closed.Context {
			next = j
		}
	}
	switch {
	case next >= 0:
		u.activateTab(next)
	case closed.Context == u.Context:
		u.View, u.Filter = "", ""
		u.Selected, u.Drawer = nil, nil
	}
	return nil
}

// pinTab makes a preview tab ("" is the active one) fixed.
func (u *UIState) pinTab(id string) error {
	if id == "" {
		id = u.ActiveTab
	}
	i := u.tabIndex(id)
	if i < 0 {
		if id == "" {
			return ErrNoTab
		}
		return fmt.Errorf("%w %q", ErrUnknownTab, id)
	}
	u.Tabs[i].Preview = false
	return nil
}

// historyTarget returns the view back (or forward) in the active tab's history, or why there is none.
func (u *UIState) historyTarget(back bool) (string, error) {
	i := u.activeIndex()
	if i < 0 {
		return "", ErrNoTab
	}
	if !u.connected() || u.Tabs[i].Context != u.Context {
		return "", errors.New("the tab's cluster is not connected")
	}
	h, what := u.Tabs[i].Forward, "no later view in this tab"
	if back {
		h, what = u.Tabs[i].Back, "no earlier view in this tab"
	}
	if len(h) == 0 {
		return "", errors.New(what)
	}
	return h[len(h)-1], nil
}

// stepHistory goes back (or forward) in the active tab's history to view, which historyTarget
// named; the view opens as ui.openView opens one.
func (u *UIState) stepHistory(back bool, view string, sort SortSpec) error {
	if target, err := u.historyTarget(back); err != nil {
		return err
	} else if target != view {
		return errTabsChanged
	}
	t := &u.Tabs[u.activeIndex()]
	if back {
		t.Back = t.Back[:len(t.Back)-1]
		t.Forward = pushHistory(t.Forward, t.View)
	} else {
		t.Forward = t.Forward[:len(t.Forward)-1]
		t.Back = pushHistory(t.Back, t.View)
	}
	t.View = view // the history moved already: syncTabs must not add to it
	u.View, u.Sort, u.Filter = view, sort, ""
	u.Selected, u.Drawer = nil, nil
	return nil
}

// tabTitle is a tab's title: "<view> - <context>", the context by its name without its kubeconfig
// file (as the navigator's tooltips), only the context while the tab has no view.
func tabTitle(viewTitle func(string) string, t ViewTab) string {
	name := ContextLabel(t.Context)
	if t.View == "" {
		return name
	}
	return viewTitle(t.View) + " - " + name
}

// NoteViewTitle remembers a view's title for the tabs' titles. The commands that open a view call
// it with the profile they resolved, so that a tab of a cluster that is not connected keeps its
// title.
func (s *Session) NoteViewTitle(view, title string) {
	if view != "" && title != "" {
		s.titles.Store(view, title)
	}
}

// noteKindTitle remembers the title of a connected cluster's view.
func (s *Session) noteKindTitle(view string) {
	if _, page := PageTitle(view); page {
		return
	}
	if _, p, err := s.Kind(view); err == nil {
		s.NoteViewTitle(view, p.Title)
	}
}

// viewTitle names a view for a tab's title: a page's title, the title remembered for a kind, else
// the key with a capital letter ("pods" → "Pods", which the default view is). It reads no lock
// but the map's, so UpdateUI may call it.
func (s *Session) viewTitle(view string) string {
	if t, ok := PageTitle(view); ok {
		return t
	}
	if v, ok := s.titles.Load(view); ok {
		return v.(string)
	}
	if view == "" {
		return ""
	}
	r, n := utf8.DecodeRuneInString(view)
	return string(unicode.ToUpper(r)) + view[n:]
}

// OpenTab opens a view of the connected cluster (a kind key or a page key the caller checked) in
// the preview tab, or in a fixed tab when fixed is set (UIState.openTab), and returns the new state.
func (s *Session) OpenTab(view string, fixed bool) UIState {
	s.noteKindTitle(view)
	sort := s.ViewSort(view)
	return s.UpdateUI(func(u *UIState) { u.openTab(view, sort, fixed) })
}

// SelectTab activates a tab. A tab of the connected cluster shows at once; a tab of another
// cluster, or of the connected one while it is not connected, connects to its cluster (as
// cluster.connect does, waiting for it) and shows once connected. Like cluster.connect it returns
// the state, with no error, when the connect failed: Connection says why.
func (s *Session) SelectTab(ctx context.Context, id string) (UIState, error) {
	u := s.UIState()
	i := u.tabIndex(id)
	if i < 0 {
		return u, fmt.Errorf("%w %q", ErrUnknownTab, id)
	}
	t := u.Tabs[i]
	if t.Context == u.Context && u.connected() {
		var err error
		st := s.UpdateUI(func(u *UIState) {
			j := u.tabIndex(id)
			switch {
			case j < 0 || u.Tabs[j].Context != u.Context || !u.connected():
				err = errTabsChanged
			case u.ActiveTab != id:
				u.activateTab(j)
			}
		})
		return st, err
	}
	// The tab becomes its cluster's most recently used one, which syncTabs shows once connected.
	// Another tab still marked active (the one shown last before a restart, its cluster named but
	// not connected: D-41) gives way, or syncTabs would show it instead of the one clicked.
	s.UpdateUI(func(u *UIState) {
		if j := u.tabIndex(id); j >= 0 {
			u.touch(j)
			if u.ActiveTab != id {
				u.ActiveTab = ""
			}
		}
	})
	if t.Context == u.Context && u.Connection.State == Connecting {
		return s.UIState(), nil // the connect under way shows it
	}
	err := s.Connect(ctx, t.Context)
	st := s.UIState()
	if err != nil && st.Connection.State != ConnectError {
		return st, err
	}
	return st, nil
}

// SelectTabAt activates the n-th tab (from 1) the tab bar shows, or the last one: the tabs of the
// navigator's scope (navscope.go).
func (s *Session) SelectTabAt(ctx context.Context, n int, last bool) (UIState, error) {
	u := s.UIState()
	tabs := u.TabsInScope()
	if len(tabs) == 0 {
		return u, ErrNoTab
	}
	switch {
	case last:
		n = len(tabs)
	case n < 1 || n > len(tabs):
		return u, fmt.Errorf("%w: tab %d of %d", ErrUnknownTab, n, len(tabs))
	}
	return s.SelectTab(ctx, tabs[n-1].ID)
}

// CycleTab activates the tab d places right of the active one (left for a negative d) among the
// tabs the tab bar shows (the navigator's scope's), wrapping around; without an active tab there,
// the first (d > 0) or the last.
func (s *Session) CycleTab(ctx context.Context, d int) (UIState, error) {
	u := s.UIState()
	tabs := u.TabsInScope()
	n := len(tabs)
	if n == 0 {
		return u, ErrNoTab
	}
	j := n - 1
	if i := slices.IndexFunc(tabs, func(t ViewTab) bool { return t.ID == u.ActiveTab }); i >= 0 {
		j = ((i+d)%n + n) % n
	} else if d > 0 {
		j = 0
	}
	return s.SelectTab(ctx, tabs[j].ID)
}

// CloseTab closes a tab ("" is the active one) and returns the new state (UIState.closeTab).
func (s *Session) CloseTab(id string) (UIState, error) {
	var err error
	st := s.UpdateUI(func(u *UIState) { err = u.closeTab(id) })
	return st, err
}

// PinTab makes a preview tab ("" is the active one) fixed.
func (s *Session) PinTab(id string) (UIState, error) {
	var err error
	st := s.UpdateUI(func(u *UIState) { err = u.pinTab(id) })
	return st, err
}

// CheckTabHistory returns why the active tab cannot go back (or forward), or nil.
func (s *Session) CheckTabHistory(back bool) error {
	u := s.UIState()
	_, err := u.historyTarget(back)
	return err
}

// TabHistory goes back (or forward) in the active tab's history.
func (s *Session) TabHistory(back bool) (UIState, error) {
	u := s.UIState()
	view, err := u.historyTarget(back)
	if err != nil {
		return u, err
	}
	s.noteKindTitle(view)
	sort := s.ViewSort(view)
	st := s.UpdateUI(func(u *UIState) { err = u.stepHistory(back, view, sort) })
	return st, err
}
