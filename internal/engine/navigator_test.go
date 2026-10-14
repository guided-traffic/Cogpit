package engine

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// The navigator's state (D-51): the search and the keyboard's row are set as the commands say and
// follow the level and the open view as the window's navigator does: a connect drills in onto the
// cluster's root with no search, a view opened puts the keyboard on its row, going up to the section
// puts it on the cluster it came from; rows of the other level are refused.
func TestNavigatorStateFollowsTheScopeAndTheView(t *testing.T) {
	s, _ := scopeSession(t)
	ctx := context.Background()
	u := s.UIState()
	if u.Navigator.Row != "" || u.NavigatorRow() != NavRowRoot {
		t.Fatalf("at start: row %q, reads %q", u.Navigator.Row, u.NavigatorRow())
	}
	if _, err := s.SetNavigatorSearch("kind"); err != nil {
		t.Fatal(err)
	}
	if err := s.Connect(ctx, "test"); err != nil {
		t.Fatal(err)
	}
	u = s.UIState()
	if u.Navigator.Search != "" || u.Navigator.Row != "pods" {
		t.Errorf("connected: search %q, row %q (the default view's)", u.Navigator.Search, u.Navigator.Row)
	}

	u, err := s.SetNavigatorSearch("dep")
	if err != nil || u.Navigator.Search != "dep" {
		t.Fatalf("search %q, %v", u.Navigator.Search, err)
	}
	u = s.OpenTab("services", false)
	if u.Navigator.Row != "services" || u.Navigator.Search != "dep" {
		t.Errorf("a view opened: row %q, search %q", u.Navigator.Row, u.Navigator.Search)
	}
	for _, row := range []string{NavRowRoot, "group:workloads", "workloads/overview", "alias:gateways.gateway.networking.k8s.io", "pods"} {
		if u, err := s.SetNavigatorRow(row); err != nil || u.Navigator.Row != row {
			t.Errorf("row %q at a cluster: %q, %v", row, u.Navigator.Row, err)
		}
	}
	for _, row := range []string{"", "context:test", "source:.kube/", "two words", "a\nb"} {
		if _, err := s.SetNavigatorRow(row); !errors.Is(err, ErrNavRow) {
			t.Errorf("row %q at a cluster: %v, want ErrNavRow", row, err)
		}
	}
	if s.UIState().Navigator.Row != "pods" {
		t.Errorf("a refused row changed the state: %q", s.UIState().Navigator.Row)
	}

	u, err = s.SetNavigatorScope(ctx, NavScopeLocal)
	if err != nil {
		t.Fatal(err)
	}
	if u.Navigator.Row != NavRowContext("test") || u.Navigator.Search != "" {
		t.Errorf("up to the section: row %q, search %q", u.Navigator.Row, u.Navigator.Search)
	}
	for _, row := range []string{NavRowRoot, "context:prod [prod.yaml]", "source:dev.yaml"} {
		if u, err := s.SetNavigatorRow(row); err != nil || u.Navigator.Row != row {
			t.Errorf("row %q at the section: %q, %v", row, u.Navigator.Row, err)
		}
	}
	for _, row := range []string{"pods", "group:workloads", "context:", "source: "} {
		if _, err := s.SetNavigatorRow(row); !errors.Is(err, ErrNavRow) {
			t.Errorf("row %q at the section: %v, want ErrNavRow", row, err)
		}
	}
	u = s.OpenTab("pods", false) // a view opened at the section leaves the section's keyboard alone
	if u.Navigator.Row != "source:dev.yaml" {
		t.Errorf("a view opened at the section: row %q", u.Navigator.Row)
	}
	u, err = s.SetNavigatorScope(ctx, NavScopeCluster("test"))
	if err != nil || u.Navigator.Row != NavRowRoot {
		t.Errorf("back into the cluster: row %q, %v", u.Navigator.Row, err)
	}
}

// The search takes one line of at most 1,000 bytes, as the view's search.
func TestNavigatorSearchBounds(t *testing.T) {
	for text, ok := range map[string]bool{
		"":                                true,
		"pods":                            true,
		"Payments prod":                   true,
		strings.Repeat("x", 1000):         true,
		strings.Repeat("x", 1001):         false,
		"a\nb":                            false,
		"tab\there":                       false,
		string([]byte{'a', 0xff, 'b'}):    false,
		"ünïcödé · ⌘":                     true,
		strings.Repeat("é", 500):          true,
		strings.Repeat("é", 500) + "x":    false,
		"context:kind-cogpit [.kube/conf": true,
	} {
		if err := CheckNavigatorSearch(text); (err == nil) != ok {
			t.Errorf("%.20q: %v, want ok %v", text, err, ok)
		}
	}
	s, _ := scopeSession(t)
	if _, err := s.SetNavigatorSearch(strings.Repeat("x", 1001)); !errors.Is(err, ErrNavSearch) {
		t.Errorf("a long search: %v", err)
	}
}

// A source group collapses and opens again; collapsing twice keeps one entry; the label is bounded.
func TestNavigatorGroups(t *testing.T) {
	s, _ := scopeSession(t)
	for _, step := range []struct {
		group     string
		collapsed bool
		want      []string
	}{
		{".kube/", true, []string{".kube/"}},
		{"dev.yaml", true, []string{".kube/", "dev.yaml"}},
		{".kube/", true, []string{".kube/", "dev.yaml"}},
		{".kube/", false, []string{"dev.yaml"}},
		{"never.yaml", false, []string{"dev.yaml"}},
		{"dev.yaml", false, nil},
	} {
		u, err := s.SetNavigatorGroup(step.group, step.collapsed)
		if err != nil || !slices.Equal(u.Navigator.Collapsed, step.want) {
			t.Fatalf("%q collapsed %v: %q, %v; want %q", step.group, step.collapsed, u.Navigator.Collapsed, err, step.want)
		}
	}
	for _, bad := range []string{"", strings.Repeat("x", 513), "a\nb"} {
		if _, err := s.SetNavigatorGroup(bad, true); err == nil {
			t.Errorf("group %.20q accepted", bad)
		}
	}
}

// The collapsed source groups survive a restart in workspace.json (A63); the search and the row do
// not (the lead's night decision): the navigator starts at the section.
func TestWorkspaceKeepsTheCollapsedSourceGroups(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Cogpit")
	a, _ := workspaceSession(t, dir, time.Hour)
	if _, err := a.SetNavigatorGroup(".kube/", true); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SetNavigatorGroup("team/", true); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SetNavigatorSearch("prod"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SetNavigatorRow("source:team/"); err != nil {
		t.Fatal(err)
	}
	a.Close()
	if doc := readWorkspace(t, dir); !slices.Equal(doc.CollapsedSources, []string{".kube/", "team/"}) {
		t.Fatalf("file keeps %q", doc.CollapsedSources)
	}

	b, _ := workspaceSession(t, dir, time.Hour)
	u := b.UIState()
	if !slices.Equal(u.Navigator.Collapsed, []string{".kube/", "team/"}) || u.Navigator.Search != "" || u.Navigator.Row != "" {
		t.Errorf("restored %+v", u.Navigator)
	}
	b.Close()

	// What is out of bounds in a file read back is dropped alone, a repeat too.
	doc := readWorkspace(t, dir)
	doc.CollapsedSources = []string{"a/", "", strings.Repeat("x", 600), "a/", "b.yaml"}
	writeWorkspace(t, dir, doc)
	c, _ := workspaceSession(t, dir, time.Hour)
	if got := c.UIState().Navigator.Collapsed; !slices.Equal(got, []string{"a/", "b.yaml"}) {
		t.Errorf("normalized %q", got)
	}
}

// The palette's text is engine state with who typed it last; closing the palette clears both, and
// ui.palette.setText opens a closed palette. An agent's text counts as another origin's.
func TestPaletteText(t *testing.T) {
	s, _ := scopeSession(t)
	u, err := s.SetPaletteText("delete", OriginExternalAgent)
	if err != nil || !u.Palette || u.PaletteText != "delete" || u.PaletteTextOrigin != OriginExternalAgent || !u.PaletteTextByOther() {
		t.Fatalf("an agent's text: %+v, %v", u, err)
	}
	u, _ = s.SetPaletteText("delete p", OriginGUI)
	if u.PaletteTextOrigin != OriginGUI || u.PaletteTextByOther() {
		t.Errorf("the user's text: origin %q", u.PaletteTextOrigin)
	}
	u, _ = s.SetPaletteText("", OriginBuiltInAgent)
	if u.PaletteText != "" || u.PaletteTextOrigin != "" || u.PaletteTextByOther() || !u.Palette {
		t.Errorf("cleared: %q %q open %v", u.PaletteText, u.PaletteTextOrigin, u.Palette)
	}
	if _, err := s.SetPaletteText("logs", OriginBuiltInAgent); err != nil {
		t.Fatal(err)
	}
	u = s.UpdateUI(func(u *UIState) { u.Palette = false }) // ui.palette.close, Escape
	if u.PaletteText != "" || u.PaletteTextOrigin != "" {
		t.Errorf("closed with text %q by %q", u.PaletteText, u.PaletteTextOrigin)
	}
	// The window's report that arrives after the user closed the palette is dropped: it never opens it.
	if u, _ = s.SetPaletteText("late", OriginGUI); u.Palette || u.PaletteText != "" {
		t.Errorf("a late report of the window: open %v, text %q", u.Palette, u.PaletteText)
	}
	if _, err := s.SetPaletteText(strings.Repeat("x", 1001), OriginGUI); err == nil {
		t.Error("a long text accepted")
	}
	if _, err := s.OpenWelcome(); err != nil { // the welcome page closes the palette and its text
		t.Fatal(err)
	}
	if _, err := s.SetPaletteText("x", OriginExternalAgent); err != nil {
		t.Fatal(err)
	}
	if u, _ := s.OpenWelcome(); u.Palette || u.PaletteText != "" {
		t.Errorf("after the welcome page: open %v, text %q", u.Palette, u.PaletteText)
	}
}
