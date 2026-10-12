package fyneui

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/ui/fyneui/yamlview"
)

// The find widget's matching: the three switches, rune columns, one line per match, empty matches
// skipped, bad regular expressions reported.
func TestFindMatches(t *testing.T) {
	const text = "Replica: 1\nreplicaCount: 2\nnote: replica-set, REPLICA\nüber: ü\n"
	tests := []struct {
		name  string
		query string
		opts  findOptions
		want  []findMatch
		err   bool
	}{
		{"case-insensitive", "replica", findOptions{}, []findMatch{{0, 0, 7}, {1, 0, 7}, {2, 6, 13}, {2, 19, 26}}, false},
		{"match case", "replica", findOptions{matchCase: true}, []findMatch{{1, 0, 7}, {2, 6, 13}}, false},
		{"whole word", "replica", findOptions{wholeWord: true}, []findMatch{{0, 0, 7}, {2, 6, 13}, {2, 19, 26}}, false},
		{"regex", `rep\w+:`, findOptions{regex: true}, []findMatch{{0, 0, 8}, {1, 0, 13}}, false},
		{"plain text is not a regex", "(a+)", findOptions{}, nil, false},
		{"regex anchors are a line's ends", `^note`, findOptions{regex: true}, []findMatch{{2, 0, 4}}, false},
		{"empty matches are skipped", `x*`, findOptions{regex: true}, nil, false},
		{"rune columns", "ü", findOptions{}, []findMatch{{3, 0, 1}, {3, 6, 7}}, false},
		{"invalid", "(", findOptions{regex: true}, nil, true},
		{"empty query", "", findOptions{}, nil, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := findMatches(text, tc.query, tc.opts)
			if (err != nil) != tc.err {
				t.Fatalf("error %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("matches %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("match %d: %v, want %v", i, got[i], tc.want[i])
				}
			}
		})
	}
	if _, err := findMatches("x", strings.Repeat("a", maxFindQuery+1), findOptions{}); err == nil {
		t.Error("a search text over the limit was taken")
	}
	many := strings.Repeat("a\n", maxFindMatches+10)
	if got, _ := findMatches(many, "a", findOptions{}); len(got) != maxFindMatches {
		t.Errorf("%d matches, want the cap %d", len(got), maxFindMatches)
	}
}

func TestReplaceAllIn(t *testing.T) {
	const text = "tag: v1\nimage: web:v1\nname: V1\n"
	got, n, err := replaceAllIn(text, "v1", "v2", findOptions{})
	if err != nil || n != 3 || got != "tag: v2\nimage: web:v2\nname: v2\n" {
		t.Errorf("plain: %q %d %v", got, n, err)
	}
	got, n, _ = replaceAllIn(text, "v1", "$0!", findOptions{matchCase: true})
	if n != 2 || got != "tag: $0!\nimage: web:$0!\nname: V1\n" {
		t.Errorf("a plain replacement is literal: %q %d", got, n)
	}
	got, n, _ = replaceAllIn(text, `(\w+): (\S+)`, "$2=$1", findOptions{regex: true})
	if n != 3 || got != "v1=tag\nweb:v1=image\nV1=name\n" {
		t.Errorf("regex: %q %d", got, n)
	}
	if _, _, err := replaceAllIn(text, "(", "x", findOptions{regex: true}); err == nil {
		t.Error("an invalid regular expression replaced")
	}
}

// findBarFixture is a find widget on a view holding text.
func findBarFixture(t *testing.T, text string) (*findBar, *yamlview.View) {
	t.Helper()
	f := kindsFake(t, "pods")
	sh, _, _ := newTestShell(t, f, Options{Theme: ThemeDark})
	v := yamlview.New()
	v.SetReadOnly(false)
	v.SetText(text)
	return newFindBar(sh, v), v
}

// The widget selects matches in the editor and counts them; next and previous wrap; the switches
// list again; replace and replace all edit the text.
func TestFindBar(t *testing.T) {
	b, v := findBarFixture(t, "a: Foo\nb: foo\nc: foobar\nd: x\n")
	changed := 0
	v.OnChanged = func() { changed++ }
	b.open()
	if !b.visible() {
		t.Fatal("open did not show the widget")
	}
	b.find.SetText("foo")
	if v.SelectedText() != "Foo" || b.count.text != "1 of 3" {
		t.Fatalf("selected %q, count %q", v.SelectedText(), b.count.text)
	}
	test.Tap(b.next)
	if line, _ := v.CursorPosition(); line != 1 || b.count.text != "2 of 3" {
		t.Errorf("next: line %d, count %q", line, b.count.text)
	}
	test.Tap(b.next)
	test.Tap(b.next)
	if line, _ := v.CursorPosition(); line != 0 || b.count.text != "1 of 3" {
		t.Errorf("next wraps: line %d, count %q", line, b.count.text)
	}
	test.Tap(b.prev)
	if line, _ := v.CursorPosition(); line != 2 || b.count.text != "3 of 3" {
		t.Errorf("previous wraps: line %d, count %q", line, b.count.text)
	}

	test.Tap(b.toggles[1]) // whole word
	if b.count.text != "2 of 2" && b.count.text != "1 of 2" {
		t.Errorf("whole word: count %q", b.count.text)
	}
	test.Tap(b.toggles[1])
	test.Tap(b.toggles[0]) // match case
	if v.SelectedText() != "foo" || !strings.HasSuffix(b.count.text, " of 2") {
		t.Errorf("match case: selected %q, count %q", v.SelectedText(), b.count.text)
	}
	test.Tap(b.toggles[0])

	b.find.SetText("nothing")
	if b.count.text != "No results" {
		t.Errorf("count %q", b.count.text)
	}
	b.toggles[2].OnTapped() // regular expression
	b.find.SetText("(")
	if b.count.text != "Invalid" {
		t.Errorf("count %q", b.count.text)
	}
	b.toggles[2].OnTapped()

	// Replace: the first press selects a match, the next replaces it and moves on.
	v.SetCursorPosition(0, 0)
	b.find.SetText("foo")
	b.replace.SetText("bar")
	test.Tap(b.replaceBtn)
	if v.Text() != "a: bar\nb: foo\nc: foobar\nd: x\n" || changed != 1 {
		t.Errorf("replace: %q (%d changes)", v.Text(), changed)
	}
	if v.SelectedText() != "foo" {
		t.Errorf("after a replace the next match is selected, got %q", v.SelectedText())
	}
	test.Tap(b.replaceAllBtn)
	if v.Text() != "a: bar\nb: bar\nc: barbar\nd: x\n" || b.count.text != "2 replaced" {
		t.Errorf("replace all: %q, count %q", v.Text(), b.count.text)
	}

	b.close()
	if b.visible() {
		t.Error("close left the widget shown")
	}
}

// Replace with a selection the user made elsewhere does not edit it, and a regular expression
// replaces with its groups.
func TestFindBarReplaceOnlyAMatch(t *testing.T) {
	b, v := findBarFixture(t, "k: v1\nk2: v2\n")
	b.open()
	b.find.SetText("v1")
	v.SelectRange(1, 0, 1, 2) // the user selected "k2" instead
	b.replace.SetText("x")
	test.Tap(b.replaceBtn)
	if v.Text() != "k: v1\nk2: v2\n" {
		t.Errorf("replaced a selection that is no match: %q", v.Text())
	}
	if v.SelectedText() != "v1" {
		t.Errorf("selected %q, want the match", v.SelectedText())
	}
	b.toggles[2].OnTapped()
	b.replace.SetText("$2$1")
	v.SetCursorPosition(0, 0)
	b.find.SetText(`(\w)(\d)`) // selects v1
	test.Tap(b.replaceBtn)
	if !strings.HasPrefix(v.Text(), "k: 1v\n") {
		t.Errorf("regex replace: %q", v.Text())
	}
}

// The tab's search icon opens and closes the widget, and an edit made with it reaches the engine's
// text before Save upgrades (Save sends the editor's text and binds to that revision).
func TestHelmUpgradeTabFindAndReplace(t *testing.T) {
	f, ref := releasesFake(t)
	up := &upgradeRecorder{}
	up.install(f)
	ed := newEditorFake(f)
	tab := engine.DockTab{ID: "helmUpgrade-1", Type: engine.DockHelmUpgrade, Title: "Helm Upgrade: ingress-nginx", Context: "demo", Ref: &ref,
		Helm: &engine.HelmUpgradeTab{Chart: "ingress-nginx", Version: "4.12.1", Versions: []string{"4.12.1"}, UserSuppliedOnly: true}}
	ed.texts[tab.ID], ed.revs[tab.ID] = "controller:\n  replicaCount: 2\n", 1
	f.ui.Dock, f.ui.ActiveDock = []engine.DockTab{tab}, tab.ID
	sh, _, q := newTestShell(t, f, Options{Theme: ThemeDark})
	var u *helmUpgradeTab
	q.until(t, "the tab", func() bool {
		if c := sh.dock.contents[tab.ID]; c != nil {
			u, _ = c.editor.(*helmUpgradeTab)
		}
		return u != nil && !u.busy && strings.Contains(u.pane.view.Text(), "replicaCount")
	})
	if u.find.visible() {
		t.Fatal("the find widget is open at first")
	}
	test.Tap(u.searchBtn)
	if !u.find.visible() {
		t.Fatal("the search icon did not open the find widget")
	}
	u.find.find.SetText("2")
	u.find.replace.SetText("5")
	test.Tap(u.find.replaceBtn)
	if u.pane.view.Text() != "controller:\n  replicaCount: 5\n" {
		t.Fatalf("text %q", u.pane.view.Text())
	}
	test.Tap(u.saveBtn)
	q.until(t, "the upgrade", func() bool { return len(up.recorded()) == 1 && !u.busy })
	ed.mu.Lock()
	sent, rev := ed.texts[tab.ID], ed.revs[tab.ID]
	ed.mu.Unlock()
	if sent != "controller:\n  replicaCount: 5\n" || up.recorded()[0].TextRevision != rev || rev == 0 {
		t.Errorf("sent %q at revision %d, upgrade %+v", sent, rev, up.recorded()[0])
	}
	test.Tap(u.searchBtn)
	if u.find.visible() {
		t.Error("the search icon did not close the find widget")
	}
}
