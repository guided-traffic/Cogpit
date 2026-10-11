package fyneui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
)

// findAll returns every object of type T in the tree under root, widgets' renderers included.
func findAll[T any](root fyne.CanvasObject) []T {
	var out []T
	var walk func(o fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		if o == nil {
			return
		}
		if t, ok := any(o).(T); ok {
			out = append(out, t)
		}
		switch x := o.(type) {
		case *fyne.Container:
			for _, c := range x.Objects {
				walk(c)
			}
		case fyne.Widget:
			for _, c := range test.WidgetRenderer(x).Objects() {
				walk(c)
			}
		}
	}
	walk(root)
	return out
}

func decodeParams[P any](t *testing.T, inv engine.Invocation) P {
	t.Helper()
	var p P
	if err := json.Unmarshal(inv.Params, &p); err != nil {
		t.Fatalf("%s params %s: %v", inv.ID, inv.Params, err)
	}
	return p
}

func headerFor(t *testing.T, sh *shell, col int) *headerCell {
	t.Helper()
	for _, h := range findAll[*headerCell](sh.view.table) {
		if h.col == col && h.Visible() {
			return h
		}
	}
	t.Fatalf("no header for column %d", col)
	return nil
}

func TestHeaderTapDispatchesSort(t *testing.T) {
	f := connectedFake(t, 200)
	sh, w, q := newTestShell(t, f, Options{})
	q.until(t, "kinds", func() bool { return len(sh.kinds) > 0 })
	settle(w)

	restarts := 3 // the visible index of the Restarts column
	test.Tap(headerFor(t, sh, restarts))
	q.until(t, "ui.sort", func() bool { return len(f.callsOf("ui.sort")) == 1 })
	if p := decodeParams[sortParams](t, f.callsOf("ui.sort")[0]); p.Column != "restarts" || p.Desc {
		t.Fatalf("first tap sent %+v, want restarts ascending", p)
	}
	q.until(t, "the sort to show", func() bool { return sh.ui.Sort.Column == "restarts" && sh.view.snap.Sort.Column == "restarts" })
	settle(w)
	if h := headerFor(t, sh, restarts); h.sort != 1 {
		t.Fatalf("header arrow %d, want ascending", h.sort)
	}

	test.Tap(headerFor(t, sh, restarts))
	q.until(t, "second ui.sort", func() bool { return len(f.callsOf("ui.sort")) == 2 })
	if p := decodeParams[sortParams](t, f.callsOf("ui.sort")[1]); p.Column != "restarts" || !p.Desc {
		t.Fatalf("second tap sent %+v, want restarts descending", p)
	}
}

func TestRowTapDispatchesSelect(t *testing.T) {
	f := connectedFake(t, 200)
	sh, w, q := newTestShell(t, f, Options{})
	q.until(t, "kinds", func() bool { return len(sh.kinds) > 0 })
	settle(w)

	var cell *tableCell
	for _, c := range sh.view.cells {
		if c.row == 3 && c.col == 1 && c.Visible() {
			cell = c
		}
	}
	if cell == nil {
		t.Fatal("row 3 is not on screen")
	}
	pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(cell).AddXY(8, cell.Size().Height/2)
	test.TapCanvas(w.Canvas(), pos)
	q.until(t, "ui.select", func() bool { return len(f.callsOf("ui.select")) == 1 })
	p := decodeParams[selectParams](t, f.callsOf("ui.select")[0])
	want := f.tables["pods"].Rows[3].Ref
	if len(p.Refs) != 1 || p.Refs[0] != want || p.NoDrawer {
		t.Fatalf("ui.select %+v, want %v with the drawer", p, want)
	}
	q.until(t, "the drawer", func() bool { return sh.drawer.shown != nil && *sh.drawer.shown == want })
	if !sh.view.rowSelected(3) {
		t.Fatal("row 3 not highlighted after the engine selected it")
	}

	// Fyne's own selection would swallow a second tap on the same cell; the shell clears it.
	test.TapCanvas(w.Canvas(), pos)
	q.until(t, "second ui.select", func() bool { return len(f.callsOf("ui.select")) == 2 })
}

func TestNavigateScrollsOnceSnapshotHasRow(t *testing.T) {
	f := connectedFake(t, 100)
	big := fixturePods(10000, 7)
	target := big.Rows[7000].Ref
	sh, w, q := newTestShell(t, f, Options{})
	q.until(t, "kinds", func() bool { return len(sh.kinds) > 0 })
	settle(w)

	// An agent's ui.navigate: the state arrives before a snapshot that contains the object.
	go func() {
		_, _ = f.Dispatch(context.Background(), engine.Invocation{ID: "ui.navigate", Origin: engine.OriginExternalAgent, Params: mustJSON(t, refParams{Ref: target})})
	}()
	q.until(t, "the selection", func() bool { return len(sh.ui.Selected) == 1 && sh.ui.Selected[0] == target })
	if sh.view.pending == nil {
		t.Fatal("no pending scroll target while the snapshot lacks the row")
	}
	for _, c := range sh.view.cells {
		if c.row >= 1000 {
			t.Fatalf("scrolled to row %d before the snapshot had the object", c.row)
		}
	}

	go f.setTable("pods", big)
	q.until(t, "the scroll", func() bool {
		for _, c := range sh.view.cells {
			if c.row == 7000 && c.selected {
				return true
			}
		}
		return false
	})
	if sh.view.pending != nil {
		t.Fatal("pending scroll target kept after scrolling")
	}
}

func TestConfirmerAnswersAndHonoursContext(t *testing.T) {
	f := connectedFake(t, 10)
	sh, _, q := newTestShell(t, f, Options{})
	req := engine.ConfirmRequest{Command: "object.delete", Label: "Delete", Danger: engine.DangerDestructive,
		Origin: engine.OriginGUI, Context: "demo", Namespaces: []string{"default"},
		Objects: []model.ObjectRef{{Kind: "pods", Namespace: "default", Name: "web-0"}}}
	type answer struct {
		ok  bool
		err error
	}
	ask := func(ctx context.Context) chan answer {
		ch := make(chan answer, 1)
		go func() {
			ok, err := sh.confirmer.Confirm(ctx, req)
			ch <- answer{ok, err}
		}()
		q.until(t, "the dialog", func() bool { return sh.confirmer.current != nil })
		return ch
	}

	ch := ask(context.Background())
	if imp := sh.confirmer.current.confirm.Importance; imp != widget.DangerImportance {
		t.Errorf("destructive confirm button importance %v, want danger", imp)
	}
	test.Tap(sh.confirmer.current.confirm)
	if a := <-ch; !a.ok || a.err != nil {
		t.Fatalf("confirm: got %+v, want true", a)
	}
	q.until(t, "the dialog to close", func() bool { return sh.confirmer.current == nil })

	ch = ask(context.Background())
	test.Tap(sh.confirmer.current.cancel)
	if a := <-ch; a.ok || a.err != nil {
		t.Fatalf("cancel: got %+v, want false", a)
	}
	q.until(t, "the dialog to close", func() bool { return sh.confirmer.current == nil })

	ctx, cancel := context.WithCancel(context.Background())
	ch = ask(ctx)
	cancel()
	if a := <-ch; a.ok || !errors.Is(a.err, context.Canceled) {
		t.Fatalf("cancelled context: got %+v, want context.Canceled", a)
	}
	q.until(t, "the dialog to close after ctx ended", func() bool { return sh.confirmer.current == nil })
}

func TestDeleteGoesThroughConfirmation(t *testing.T) {
	f := connectedFake(t, 20)
	row := f.tables["pods"].Rows[2]
	f.details[row.Ref.String()] = fixturePodDetail(row)
	ref := row.Ref
	f.ui.Selected, f.ui.Drawer = []model.ObjectRef{ref}, &ref
	sh, _, q := newTestShell(t, f, Options{})
	q.until(t, "the toolbar", func() bool { return len(sh.drawer.actions.Objects) == 3 })
	del := sh.drawer.actions.Objects[2].(*widget.Button)
	if del.Text != "Delete" || del.Disabled() {
		t.Fatalf("third action %q disabled=%v", del.Text, del.Disabled())
	}
	test.Tap(del)
	q.until(t, "the confirmation", func() bool { return sh.confirmer.current != nil })
	test.Tap(sh.confirmer.current.cancel)
	q.until(t, "the dialog to close", func() bool { return sh.confirmer.current == nil })
	if n := len(f.callsOf("object.delete")); n != 1 {
		t.Fatalf("object.delete dispatched %d times", n)
	}
}

func TestDrawerActionsDisabledWithReason(t *testing.T) {
	f := connectedFake(t, 20)
	f.disabled["pod.shell"] = "not available on the demo cluster"
	f.disabled["object.delete"] = "not available on the demo cluster"
	row := f.tables["pods"].Rows[1]
	f.details[row.Ref.String()] = fixturePodDetail(row)
	ref := row.Ref
	f.ui.Selected, f.ui.Drawer = []model.ObjectRef{ref}, &ref
	sh, _, q := newTestShell(t, f, Options{})
	q.until(t, "the toolbar", func() bool { return len(sh.drawer.actions.Objects) == 3 })
	states := map[string]bool{}
	for _, o := range sh.drawer.actions.Objects {
		b := o.(*widget.Button)
		states[b.Text] = b.Disabled()
	}
	if states["Logs"] || !states["Shell"] || !states["Delete"] {
		t.Fatalf("disabled states %v, want Shell and Delete disabled", states)
	}
	if got := sh.drawer.reasons.text; got != "Shell, Delete: not available on the demo cluster" {
		t.Fatalf("reason line %q", got)
	}
}

func TestSearchIsDebounced(t *testing.T) {
	f := connectedFake(t, 20)
	sh, _, q := newTestShell(t, f, Options{})
	q.until(t, "kinds", func() bool { return len(sh.kinds) > 0 })
	test.Type(sh.view.search, "web")
	q.until(t, "ui.setFilter", func() bool { return len(f.callsOf("ui.setFilter")) == 1 })
	if p := decodeParams[filterParams](t, f.callsOf("ui.setFilter")[0]); p.Filter != "web" {
		t.Fatalf("filter %q, want the whole word", p.Filter)
	}
	time.Sleep(3 * searchDebounce)
	q.drain()
	if n := len(f.callsOf("ui.setFilter")); n != 1 {
		t.Fatalf("%d ui.setFilter calls for one burst of typing", n)
	}

	// A filter set elsewhere (an agent) shows in the entry.
	go f.update(func(u *engine.UIState) { u.Filter = "checkout" })
	q.until(t, "the entry to follow", func() bool { return sh.view.search.Text == "checkout" })
}

func TestDockOpensAndClosesStreams(t *testing.T) {
	f := connectedFake(t, 20)
	sh, _, q := newTestShell(t, f, Options{})
	ref := f.tables["pods"].Rows[0].Ref
	go func() {
		_, _ = f.Dispatch(context.Background(), engine.Invocation{ID: "pod.logs", Origin: engine.OriginGUI,
			Params: mustJSON(t, containerParams{Ref: ref, Container: "api"})})
	}()
	q.until(t, "the log stream", func() bool {
		f.mu.Lock()
		defer f.mu.Unlock()
		return f.opened["logs-1"] == 1
	})
	if !sh.dock.open() {
		t.Fatal("dock did not open for a new tab")
	}
	go func() {
		_, _ = f.Dispatch(context.Background(), engine.Invocation{ID: "dock.close", Origin: engine.OriginGUI,
			Params: mustJSON(t, dockParams{ID: "logs-1"})})
	}()
	q.until(t, "the stream to close", func() bool {
		f.mu.Lock()
		defer f.mu.Unlock()
		return f.closed["logs-1"] == 1
	})
	if sh.dock.open() {
		t.Fatal("dock still open without tabs")
	}
}

func TestPickerConnects(t *testing.T) {
	f := newFakeEngine(t)
	f.contexts = fixtureContexts()
	f.kinds = fixtureKinds()
	f.tables["pods"] = fixturePods(50, 1)
	sh, _, q := newTestShell(t, f, Options{})
	q.until(t, "the picker", func() bool { return sh.picker.visible() && len(sh.picker.list.Objects) == len(f.contexts) })
	if sh.picker.selected != "kind-cogpit" {
		t.Fatalf("preselected %q, want the kubeconfig's current context", sh.picker.selected)
	}
	rows := findAll[*pickerRow](sh.picker.list)
	test.Tap(rows[0]) // the demo cluster
	q.drain()
	if sh.picker.selected != "demo" {
		t.Fatalf("selected %q after tapping the demo row", sh.picker.selected)
	}
	test.Tap(sh.picker.connectB)
	q.until(t, "the connection", func() bool { return sh.ui.Connection.State == engine.Connected && !sh.picker.visible() })
	if p := decodeParams[contextParams](t, f.callsOf("cluster.connect")[0]); p.Context != "demo" {
		t.Fatalf("connected %q", p.Context)
	}
	q.until(t, "the pods table", func() bool { return sh.view.snap != nil && len(sh.kinds) > 0 })
}

func TestAutoConnectSkipsPicker(t *testing.T) {
	f := newFakeEngine(t)
	f.contexts = fixtureContexts()
	f.kinds = fixtureKinds()
	sh, _, q := newTestShell(t, f, Options{AutoConnect: "demo"})
	q.until(t, "the connection", func() bool { return sh.ui.Connection.State == engine.Connected })
	if sh.picker.visible() {
		t.Fatal("picker shown although AutoConnect is set")
	}
}

func TestHiddenColumnsAreSkipped(t *testing.T) {
	f := connectedFake(t, 10)
	f.tables["events"] = &engine.TableSnapshot{
		View: "events", Title: "Events", Namespaced: true, Synced: true, Total: 1,
		Columns: []model.Column{
			{ID: "name", Title: "Name", Type: model.ColumnText, Hidden: true},
			{ID: "type", Title: "Type", Type: model.ColumnText},
			{ID: "message", Title: "Message", Type: model.ColumnText},
			{ID: "age", Title: "Age", Type: model.ColumnAge},
		},
		Rows: []model.Row{{Ref: model.ObjectRef{Kind: "events", Namespace: "default", Name: "web.1"},
			Cells: []model.Cell{{Text: "web.1"}, {Text: "Warning", Status: model.StatusWarning}, {Text: "Back-off"}, {Num: fixedNow.Unix() - 60}}}},
	}
	f.ui.View = "events"
	sh, w, q := newTestShell(t, f, Options{})
	q.until(t, "kinds", func() bool { return len(sh.kinds) > 0 })
	settle(w)
	if rows, cols := sh.view.length(); rows != 1 || cols != 3 {
		t.Fatalf("table %d×%d, want 1×3 without the hidden name", rows, cols)
	}
	texts := map[int]string{}
	for _, c := range sh.view.cells {
		if c.row == 0 && c.Visible() {
			texts[c.col] = c.full
		}
	}
	if texts[0] != "Warning" || texts[1] != "Back-off" || texts[2] != "1m" {
		t.Fatalf("row cells %v", texts)
	}
	if sh.view.flexColumn() != 1 {
		t.Fatalf("flex column %d, want the message", sh.view.flexColumn())
	}
}

// TestEngineEventsWithoutRaces floods the shell with events from several engine goroutines while
// the "main goroutine" applies them. Run with -race: a widget touched outside sh.do is a race
// between an engine goroutine and the test goroutine.
func TestEngineEventsWithoutRaces(t *testing.T) {
	f := connectedFake(t, 2000)
	snaps := []*engine.TableSnapshot{fixturePods(2000, 1), fixturePods(2000, 2), fixturePods(1500, 3)}
	row := f.tables["pods"].Rows[0]
	f.details[row.Ref.String()] = fixturePodDetail(row)
	sh, _, q := newTestShell(t, f, Options{})

	const n = 150
	var wg sync.WaitGroup
	wg.Add(4)
	go func() {
		defer wg.Done()
		for i := range n {
			f.setTable("pods", snaps[i%len(snaps)])
		}
	}()
	go func() {
		defer wg.Done()
		for i := range n {
			f.update(func(u *engine.UIState) {
				u.Filter = strings.Repeat("x", i%4)
				r := row.Ref
				u.Selected, u.Drawer = []model.ObjectRef{r}, &r
			})
		}
	}()
	go func() {
		defer wg.Done()
		for range n {
			f.emit(engine.Event{Type: engine.EventDiagnostics})
			f.emit(engine.Event{Type: engine.EventDetail})
		}
	}()
	go func() {
		defer wg.Done()
		for i := range n {
			f.emit(engine.Event{Type: engine.EventToast, Level: "info", Message: "event " + strings.Repeat("·", i%3)})
		}
	}()
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	for running := true; running; {
		select {
		case <-done:
			running = false
		default:
			q.drain()
		}
	}
	want := f.UIState().Revision
	q.until(t, "the last state", func() bool { return sh.ui.Revision == want && sh.view.snap == f.Table("pods") })
	// Every refresh measures a fresh template cell; only shown cells may be kept.
	if n := len(sh.view.cells); n > 600 {
		t.Fatalf("%d cells tracked after %d snapshots: template cells leak", n, n)
	}
}

func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestConfirmationsOneAtATimeAndArmed checks the review finding that confirmations stacked: a
// second request waits until the first is answered, and a fresh dialog's confirm button ignores
// taps until it arms.
func TestConfirmationsOneAtATimeAndArmed(t *testing.T) {
	f := connectedFake(t, 10)
	sh, _, q := newTestShell(t, f, Options{})
	sh.confirmer.armDelay = 150 * time.Millisecond
	mk := func(name string) engine.ConfirmRequest {
		return engine.ConfirmRequest{Command: "object.delete", Label: "Delete", Danger: engine.DangerDestructive,
			Origin: engine.OriginExternalAgent, Context: "demo", Namespaces: []string{"default"},
			Objects: []model.ObjectRef{{Kind: "pods", Namespace: "default", Name: name}}}
	}
	answers := make(chan string, 2)
	ask := func(name string) {
		go func() {
			ok, _ := sh.confirmer.Confirm(context.Background(), mk(name))
			answers <- fmt.Sprintf("%s=%v", name, ok)
		}()
	}
	ask("first")
	q.until(t, "the first dialog", func() bool { return sh.confirmer.current != nil })
	first := sh.confirmer.current
	ask("second")
	time.Sleep(50 * time.Millisecond)
	q.drain()
	if sh.confirmer.current != first {
		t.Fatal("a second dialog opened while the first was on screen")
	}
	test.Tap(first.confirm) // not armed yet: ignored
	select {
	case a := <-answers:
		t.Fatalf("an unarmed confirm button answered: %s", a)
	case <-time.After(30 * time.Millisecond):
	}
	q.until(t, "the confirm button to arm", func() bool { return !first.confirm.Disabled() })
	test.Tap(first.confirm)
	if a := <-answers; a != "first=true" {
		t.Fatalf("first answer %q", a)
	}
	q.until(t, "the second dialog", func() bool { return sh.confirmer.current != nil && sh.confirmer.current != first })
	test.Tap(sh.confirmer.current.cancel)
	if a := <-answers; a != "second=false" {
		t.Fatalf("second answer %q", a)
	}
}
