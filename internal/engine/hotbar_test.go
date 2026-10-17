package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/testkit"
)

func tiles(h HotbarState) string {
	var out []string
	for _, t := range h.Clusters {
		out = append(out, t.Context)
	}
	return strings.Join(out, ",")
}

// The hotbar's operations: one list without a limit (D-83), a context once, moves shift the tiles
// between, slots numbered from 1.
func TestHotbarStateOperations(t *testing.T) {
	h := defaultHotbar()
	if h.Clusters == nil || len(h.Clusters) != 0 {
		t.Fatalf("default hotbar %+v", h)
	}
	for i := range 30 { // more than F1–F12 reach, and more than a bank held
		if err := h.AddCluster(fmt.Sprintf("c%d", i+1)); err != nil {
			t.Fatal(err)
		}
	}
	if err := h.CheckAdd("c31"); err != nil {
		t.Errorf("a thirty-first cluster: %v", err)
	}
	if err := h.CheckAdd("c3"); !errors.Is(err, ErrNotApplicable) || !strings.HasSuffix(err.Error(), "c3 is on the hotbar already") {
		t.Errorf("a context twice: %v, want not applicable (menus offer Remove)", err)
	}
	if err := h.AddCluster(" "); err == nil {
		t.Error("an empty context was added")
	}
	if err := h.MoveCluster("c30", 1); err != nil {
		t.Fatal(err)
	}
	if err := h.MoveCluster("c1", 3); err != nil {
		t.Fatal(err)
	}
	if got := tiles(h); !strings.HasPrefix(got, "c30,c2,c1,c3,c4,") || !strings.HasSuffix(got, ",c28,c29") {
		t.Errorf("after the moves: %s", got)
	}
	if err := h.CheckMove("c1", 31); err == nil || err.Error() != "position 31 is outside the hotbar's 1 to 30" {
		t.Errorf("a move past the end: %v", err)
	}
	if err := h.CheckMove("nope", 1); !errors.Is(err, ErrNotApplicable) {
		t.Errorf("moving a context not on the hotbar: %v", err)
	}
	if err := h.RemoveCluster("c2"); err != nil {
		t.Fatal(err)
	}
	if err := h.RemoveCluster("c2"); !errors.Is(err, ErrNotApplicable) || !strings.Contains(err.Error(), "c2 is not on the hotbar") {
		t.Errorf("removing it again: %v", err)
	}
	if tile, err := h.Tile(2); err != nil || tile.Context != "c1" {
		t.Errorf("slot 2 after a removal: %+v %v", tile, err)
	}
	if tile, err := h.Tile(29); err != nil || tile.Context != "c29" {
		t.Errorf("slot 29, past F12: %+v %v", tile, err)
	}
	if _, err := h.Tile(30); err == nil || err.Error() != "the hotbar has no cluster in slot 30 (it has 29)" {
		t.Errorf("an empty slot: %v", err)
	}
	if h.Position("c29") != 29 || h.Position("nope") != 0 {
		t.Errorf("positions %d %d", h.Position("c29"), h.Position("nope"))
	}
}

// A stored file is read as far as it makes sense: duplicates, empty and invalid contexts are
// dropped, never refused, and the tiles keep their order; there is no limit to their number.
func TestHotbarNormalizesWhatItReads(t *testing.T) {
	dir := t.TempDir()
	var many []HotbarTile
	for i := range 15 {
		many = append(many, HotbarTile{Context: fmt.Sprintf("c%d", i)})
	}
	many = append(many[:3], append([]HotbarTile{{Context: "c0"}, {Context: ""}, {Context: "  "}, {Context: strings.Repeat("x", maxHotbarContext+1)}}, many[3:]...)...)
	data, _ := json.Marshal(map[string]any{"version": 2, "clusters": many})
	if err := os.WriteFile(filepath.Join(dir, HotbarFile), data, 0o600); err != nil {
		t.Fatal(err)
	}
	s := New(Options{Hotbar: NewHotbar(dir)})
	defer s.Close()
	st, err := s.HotbarState()
	if err != nil {
		t.Fatal(err)
	}
	if got := tiles(st); got != "c0,c1,c2,c3,c4,c5,c6,c7,c8,c9,c10,c11,c12,c13,c14" {
		t.Errorf("tiles %s", got)
	}
	if raw, _ := os.ReadFile(filepath.Join(dir, HotbarFile)); string(raw) != string(data) {
		t.Error("reading a file of the current format wrote it")
	}
	empty := t.TempDir()
	if err := os.WriteFile(filepath.Join(empty, HotbarFile), []byte(`{"version": 2, "clusters": null}`), 0o600); err != nil {
		t.Fatal(err)
	}
	s2 := New(Options{Hotbar: NewHotbar(empty)})
	defer s2.Close()
	if st, _ := s2.HotbarState(); st.Clusters == nil {
		t.Error("an empty hotbar's clusters are null in JSON")
	}
}

// A file of the banks' time (format 1, or none) becomes one hotbar (D-83): the first bank's tiles in
// order, then each further bank's appended in order, each context once where it first appears; no
// tile is lost, a bank's thirteenth neither, whichever bank was shown. The new format is written at
// once and the old file kept as hotbar.json.v1; the next start reads the new file as it is.
func TestHotbarMergesBanks(t *testing.T) {
	var thirteen []HotbarTile
	for i := range 13 {
		thirteen = append(thirteen, HotbarTile{Context: fmt.Sprintf("p%d [prod.yaml]", i+1)})
	}
	ctx := func(ids ...string) []map[string]string {
		out := []map[string]string{}
		for _, id := range ids {
			out = append(out, map[string]string{"context": id})
		}
		return out
	}
	for name, c := range map[string]struct {
		doc  any
		want string
	}{
		"three banks, the second shown": {
			doc: map[string]any{"version": 1, "current": 2, "banks": []any{
				map[string]any{"name": "Default", "clusters": ctx("kind-dev", "demo")},
				map[string]any{"name": "Prod", "clusters": ctx("prod-eu", "kind-dev", "prod-us")},
				map[string]any{"name": "Empty", "clusters": ctx()},
				map[string]any{"name": "Staging", "clusters": ctx("staging", "", "demo")},
			}},
			want: "kind-dev,demo,prod-eu,prod-us,staging",
		},
		"no version, a full bank and one over": {
			doc: map[string]any{"current": 1, "banks": []any{
				map[string]any{"name": "Default", "clusters": thirteen},
				map[string]any{"name": "Two", "clusters": ctx("last")},
			}},
			want: tiles(HotbarState{Clusters: thirteen}) + ",last",
		},
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, HotbarFile)
			old, _ := json.Marshal(c.doc)
			if err := os.WriteFile(path, old, 0o600); err != nil {
				t.Fatal(err)
			}
			s := New(Options{Hotbar: NewHotbar(dir)})
			got := s.UIState().Hotbar
			if got == nil || tiles(*got) != c.want {
				t.Fatalf("merged %+v, want %s", got, c.want)
			}
			s.Close()
			if raw, err := os.ReadFile(filepath.Join(dir, HotbarBanksBackup)); err != nil || string(raw) != string(old) {
				t.Errorf("the banks' file is not kept: %q %v", raw, err)
			}
			if runtime.GOOS != "windows" {
				if fi, err := os.Stat(filepath.Join(dir, HotbarBanksBackup)); err != nil || fi.Mode().Perm() != 0o600 {
					t.Errorf("hotbar.json.v1: %v %v", fi, err)
				}
			}
			var doc struct {
				Version  int          `json:"version"`
				Clusters []HotbarTile `json:"clusters"`
				Banks    any          `json:"banks"`
				Current  any          `json:"current"`
			}
			raw, _ := os.ReadFile(path)
			if err := json.Unmarshal(raw, &doc); err != nil || doc.Version != 2 || doc.Banks != nil || doc.Current != nil ||
				tiles(HotbarState{Clusters: doc.Clusters}) != c.want {
				t.Errorf("hotbar.json after the merge: %s %v", raw, err)
			}
			s2 := New(Options{Hotbar: NewHotbar(dir)})
			defer s2.Close()
			if got := s2.UIState().Hotbar; got == nil || tiles(*got) != c.want {
				t.Errorf("the next start: %+v", got)
			}
			if after, _ := os.ReadFile(path); string(after) != string(raw) {
				t.Error("the next start wrote the file again")
			}
		})
	}
}

// The hotbar is kept in hotbar.json under Cogpit's configuration directory: owner-only, replaced
// whole on every change, read again at the next start; nothing else is written there.
func TestHotbarPersistsPrivately(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Cogpit")
	s := New(Options{Hotbar: NewHotbar(dir)})
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("the directory exists before the first change: %v", err)
	}
	if h := s.UIState().Hotbar; h == nil || h.Clusters == nil || len(h.Clusters) != 0 {
		t.Fatalf("a first start shows an empty hotbar: %+v", h)
	}
	if _, err := s.ChangeHotbar(func(h *HotbarState) error {
		if err := h.AddCluster("kind-dev [.kube/config]"); err != nil {
			return err
		}
		return h.AddCluster("demo")
	}); err != nil {
		t.Fatal(err)
	}
	s.Close()

	path := filepath.Join(dir, HotbarFile)
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		if fi.Mode().Perm() != 0o600 {
			t.Errorf("hotbar.json mode %v, want 0600", fi.Mode().Perm())
		}
		if di, _ := os.Stat(dir); di.Mode().Perm() != 0o700 {
			t.Errorf("directory mode %v, want 0700", di.Mode().Perm())
		}
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("the directory holds %d entries, want hotbar.json alone", len(entries))
	}
	raw, _ := os.ReadFile(path)
	if strings.Contains(string(raw), "connection") || strings.Contains(string(raw), "label") || strings.Contains(string(raw), "bank") ||
		!strings.Contains(string(raw), `"version": 2`) {
		t.Errorf("the file: %s", raw)
	}

	s2 := New(Options{Hotbar: NewHotbar(dir)})
	defer s2.Close()
	if got := s2.UIState().Hotbar; got == nil || tiles(*got) != "kind-dev [.kube/config],demo" {
		t.Errorf("after a restart: %+v", got)
	}
}

// A file Cogpit cannot read does not stop the start: the hotbar starts empty, and the file is moved
// aside at the first change instead of being overwritten.
func TestHotbarUnreadableFileIsKeptAside(t *testing.T) {
	for name, content := range map[string]string{"garbage": "{not json", "future": `{"version": 7, "clusters": []}`} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, HotbarFile)
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
			s := New(Options{Hotbar: NewHotbar(dir)})
			defer s.Close()
			if h := s.UIState().Hotbar; h == nil || len(h.Clusters) != 0 {
				t.Fatalf("hotbar %+v", h)
			}
			if raw, _ := os.ReadFile(path); string(raw) != content {
				t.Fatal("the unreadable file was changed before any change of the hotbar")
			}
			if _, err := s.ChangeHotbar(func(h *HotbarState) error { return h.AddCluster("a") }); err != nil {
				t.Fatal(err)
			}
			if raw, err := os.ReadFile(path + ".unreadable"); err != nil || string(raw) != content {
				t.Errorf("the old file is not kept aside: %q %v", raw, err)
			}
		})
	}
}

// A change that fails, in the operation or in the write, changes nothing: neither the file nor the
// UI state.
func TestHotbarFailedChangeChangesNothing(t *testing.T) {
	dir := t.TempDir()
	s := New(Options{Hotbar: NewHotbar(dir)})
	defer s.Close()
	rev := s.UIState().Revision
	if _, err := s.ChangeHotbar(func(h *HotbarState) error { return h.RemoveCluster("a") }); err == nil {
		t.Fatal("removed a tile the hotbar does not hold")
	}
	// The file's place is a directory: the write fails.
	if err := os.Mkdir(filepath.Join(dir, HotbarFile), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ChangeHotbar(func(h *HotbarState) error { return h.AddCluster("a") }); err == nil {
		t.Fatal("a failed write reported no error")
	}
	if u := s.UIState(); u.Revision != rev || len(u.Hotbar.Clusters) != 0 {
		t.Errorf("the UI state moved: revision %d → %d, %+v", rev, u.Revision, u.Hotbar)
	}
	if st, _ := s.HotbarState(); len(st.Clusters) != 0 {
		t.Errorf("the state moved: %+v", st)
	}
}

// What the user sees of each tile: its label, the connection to the active cluster, and a context
// no source offers any more marked missing instead of dropped.
func TestHotbarViewMarksConnectionAndMissing(t *testing.T) {
	src := testkit.NewMemorySource()
	fc := &fakeConnector{src: src, profiles: testProfiles()}
	s := New(Options{Connector: fc, Hotbar: NewHotbar(t.TempDir())})
	t.Cleanup(s.Close)
	if _, err := s.ChangeHotbar(func(h *HotbarState) error {
		for _, c := range []string{"test", "broken", "gone [old.yaml]"} {
			if err := h.AddCluster(c); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.Connect(context.Background(), "test"); err != nil {
		t.Fatal(err)
	}
	v, err := s.HotbarView()
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]HotbarTile{}
	for _, tl := range v.Clusters {
		got[tl.Context] = tl
	}
	if tl := got["test"]; tl.Connection != Connected || tl.Missing || tl.Label != "test" {
		t.Errorf("the connected tile %+v", tl)
	}
	if tl := got["broken"]; tl.Connection != Disconnected || tl.Missing {
		t.Errorf("another tile %+v", tl)
	}
	if tl := got["gone [old.yaml]"]; !tl.Missing || tl.Label != "gone" || tl.Connection != Disconnected {
		t.Errorf("the missing tile %+v", tl)
	}
	// The UI state carries the stored hotbar; its tiles name their contexts only.
	if u := s.UIState(); u.Hotbar == nil || u.Hotbar.Clusters[0] != (HotbarTile{Context: "test"}) {
		t.Errorf("UIState.Hotbar %+v", u.Hotbar)
	}
	// Connecting changes no hotbar; a change of the hotbar keeps the connection and the view.
	before := s.UIState()
	if _, err := s.ChangeHotbar(func(h *HotbarState) error { return h.MoveCluster("gone [old.yaml]", 1) }); err != nil {
		t.Fatal(err)
	}
	after := s.UIState()
	if after.Context != before.Context || after.View != before.View || after.Connection != before.Connection || after.Revision <= before.Revision {
		t.Errorf("a hotbar change moved more than the hotbar: %+v → %+v", before, after)
	}
}

// Concurrent changes are serialised: none is lost, and the UI state ends with the stored state.
func TestHotbarConcurrentChanges(t *testing.T) {
	s := New(Options{Hotbar: NewHotbar(t.TempDir())})
	defer s.Close()
	const n = 20
	done := make(chan error, n)
	for i := range n {
		go func() {
			_, err := s.ChangeHotbar(func(h *HotbarState) error { return h.AddCluster(fmt.Sprintf("c%d", i)) })
			done <- err
		}()
	}
	for range n {
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(10 * time.Second):
			t.Fatal("a change hangs")
		}
	}
	st, _ := s.HotbarState()
	if u := s.UIState(); len(st.Clusters) != n || tiles(*u.Hotbar) != tiles(st) {
		t.Errorf("stored %s, shown %s", tiles(st), tiles(*u.Hotbar))
	}
}

// Without a hotbar the session says so, and its UI state has none.
func TestNoHotbar(t *testing.T) {
	s := New(Options{})
	defer s.Close()
	if s.UIState().Hotbar != nil {
		t.Error("a hotbar without a store")
	}
	if _, err := s.ChangeHotbar(func(*HotbarState) error { return nil }); !errors.Is(err, ErrNoHotbar) {
		t.Errorf("ChangeHotbar: %v", err)
	}
	if _, err := s.HotbarView(); !errors.Is(err, ErrNoHotbar) {
		t.Errorf("HotbarView: %v", err)
	}
}

func TestContextLabel(t *testing.T) {
	for id, want := range map[string]string{
		"kind-dev [.kube/config]": "kind-dev", "demo": "demo", "odd [x] [y.yaml]": "odd [x]", "[a]": "[a]",
	} {
		if got := ContextLabel(id); got != want {
			t.Errorf("ContextLabel(%q) = %q, want %q", id, got, want)
		}
	}
}
