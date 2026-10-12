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

func tiles(b HotbarBank) string {
	var out []string
	for _, t := range b.Clusters {
		out = append(out, t.Context)
	}
	return strings.Join(out, ",")
}

// The bank operations: twelve clusters per bank, a context once per bank, moves shift the tiles
// between, banks numbered from 1 with 0 meaning the shown one, at least one and at most twelve.
func TestHotbarStateOperations(t *testing.T) {
	h := defaultHotbar()
	if len(h.Banks) != 1 || h.Banks[0].Name != DefaultHotbarBank || h.Current != 1 {
		t.Fatalf("default hotbar %+v", h)
	}
	for i := range HotbarBankSize {
		if err := h.AddCluster(0, fmt.Sprintf("c%d", i+1)); err != nil {
			t.Fatal(err)
		}
	}
	if err := h.CheckAdd(0, "c13"); err == nil || err.Error() != "bank full: 12 clusters" {
		t.Errorf("a thirteenth cluster: %v", err)
	}
	if err := h.CheckAdd(1, "c3"); !errors.Is(err, ErrNotApplicable) {
		t.Errorf("a context twice: %v, want not applicable (menus offer Remove)", err)
	}
	if err := h.AddCluster(0, " "); err == nil {
		t.Error("an empty context was added")
	}
	if err := h.MoveCluster(0, "c12", 1); err != nil {
		t.Fatal(err)
	}
	if err := h.MoveCluster(0, "c1", 3); err != nil {
		t.Fatal(err)
	}
	if got := tiles(h.Banks[0]); got != "c12,c2,c1,c3,c4,c5,c6,c7,c8,c9,c10,c11" {
		t.Errorf("after the moves: %s", got)
	}
	if err := h.CheckMove(0, "c1", 13); err == nil {
		t.Error("a move past the end")
	}
	if err := h.CheckMove(0, "nope", 1); !errors.Is(err, ErrNotApplicable) {
		t.Errorf("moving a context not in the bank: %v", err)
	}
	if err := h.RemoveCluster(0, "c2"); err != nil {
		t.Fatal(err)
	}
	if tile, err := h.Tile(0, 2); err != nil || tile.Context != "c1" {
		t.Errorf("slot 2 after a removal: %+v %v", tile, err)
	}
	if _, err := h.Tile(0, 12); err == nil || !strings.Contains(err.Error(), "no cluster in slot 12") {
		t.Errorf("an empty slot: %v", err)
	}

	if err := h.CheckRemoveBank(0); err == nil || err.Error() != "the hotbar keeps at least one bank" {
		t.Errorf("removing the last bank: %v", err)
	}
	n, err := h.CreateBank("")
	if err != nil || n != 2 || h.Current != 2 || h.Banks[1].Name != "Bank 2" {
		t.Fatalf("CreateBank: %d %v %+v", n, err, h)
	}
	if _, err := h.CreateBank("two\nlines"); err == nil {
		t.Error("a bank name with a newline")
	}
	if err := h.RenameBank(0, "  Production  "); err != nil || h.Banks[1].Name != "Production" {
		t.Errorf("RenameBank: %v %q", err, h.Banks[1].Name)
	}
	if err := h.RenameBank(1, strings.Repeat("x", maxBankName+1)); err == nil {
		t.Error("an overlong name")
	}
	if err := h.RenameBank(3, "x"); err == nil || err.Error() != "the hotbar has no bank 3 (it has 2 banks)" {
		t.Errorf("renaming bank 3: %v", err)
	}
	if err := h.AddCluster(2, "c1"); err != nil { // another bank may hold the same context
		t.Fatal(err)
	}
	for h.CheckCreateBank() == nil {
		if _, err := h.CreateBank(""); err != nil {
			t.Fatal(err)
		}
	}
	if len(h.Banks) != HotbarMaxBanks || !strings.Contains(h.CheckCreateBank().Error(), "⇧F1–⇧F12") {
		t.Errorf("%d banks: %v", len(h.Banks), h.CheckCreateBank())
	}
	if err := h.SwitchBank(2); err != nil || h.Current != 2 {
		t.Fatal(err)
	}
	if err := h.SwitchBank(0); err == nil {
		t.Error("SwitchBank(0)")
	}
	// Removing a bank before the shown one keeps the shown one shown, under its new number.
	if _, err := h.RemoveBank(1); err != nil || h.Current != 1 || h.Banks[0].Name != "Production" {
		t.Errorf("removing bank 1 while 2 is shown: %v, current %d, %q", err, h.Current, h.Banks[0].Name)
	}
	// Removing the shown last bank shows the new last.
	h.Current = len(h.Banks)
	if _, err := h.RemoveBank(0); err != nil || h.Current != len(h.Banks) {
		t.Errorf("removing the shown last bank: %v, current %d of %d", err, h.Current, len(h.Banks))
	}
	// Removing the shown bank in the middle shows the one that takes its number.
	h.Current = 2
	next := h.Banks[2].Name
	removed, err := h.RemoveBank(0)
	if err != nil || h.Current != 2 || h.Banks[1].Name != next || removed.Name == next {
		t.Errorf("removing the shown middle bank: %v, current %d, %q", err, h.Current, h.Banks[1].Name)
	}
}

// A stored file is read as far as it makes sense: too many tiles, duplicates, empty contexts and
// names, and a shown bank that does not exist are mended, never refused.
func TestHotbarNormalizesWhatItReads(t *testing.T) {
	dir := t.TempDir()
	var many []HotbarTile
	for i := range 15 {
		many = append(many, HotbarTile{Context: fmt.Sprintf("c%d", i)})
	}
	many = append(many[:3], append([]HotbarTile{{Context: "c0"}, {Context: ""}, {Context: "  "}}, many[3:]...)...)
	doc := hotbarDoc{Version: 1, Current: 9, Banks: []HotbarBank{{Name: "", Clusters: many}, {Name: "Ops\x07", Clusters: nil}}}
	data, _ := json.Marshal(doc)
	if err := os.WriteFile(filepath.Join(dir, HotbarFile), data, 0o600); err != nil {
		t.Fatal(err)
	}
	s := New(Options{Hotbar: NewHotbar(dir)})
	defer s.Close()
	st, err := s.HotbarState()
	if err != nil {
		t.Fatal(err)
	}
	if st.Current != 1 || len(st.Banks) != 2 || st.Banks[0].Name != "Bank 1" || st.Banks[1].Name != "Bank 2" {
		t.Errorf("banks %+v", st)
	}
	if got := tiles(st.Banks[0]); got != "c0,c1,c2,c3,c4,c5,c6,c7,c8,c9,c10,c11" {
		t.Errorf("tiles %s", got)
	}
	if st.Banks[1].Clusters == nil {
		t.Error("an empty bank's clusters are null in JSON")
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
	if s.UIState().Hotbar == nil || len(s.UIState().Hotbar.Banks) != 1 {
		t.Fatalf("a first start shows one empty bank: %+v", s.UIState().Hotbar)
	}
	if _, err := s.ChangeHotbar(func(h *HotbarState) error {
		if err := h.AddCluster(0, "kind-dev [.kube/config]"); err != nil {
			return err
		}
		_, err := h.CreateBank("Prod")
		return err
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
	if strings.Contains(string(raw), "connection") || strings.Contains(string(raw), "label") {
		t.Errorf("derived fields stored: %s", raw)
	}

	s2 := New(Options{Hotbar: NewHotbar(dir)})
	defer s2.Close()
	got := s2.UIState().Hotbar
	if got == nil || got.Current != 2 || len(got.Banks) != 2 || tiles(got.Banks[0]) != "kind-dev [.kube/config]" || got.Banks[1].Name != "Prod" {
		t.Errorf("after a restart: %+v", got)
	}
}

// A file Cogpit cannot read does not stop the start: the hotbar starts empty, and the file is moved
// aside at the first change instead of being overwritten.
func TestHotbarUnreadableFileIsKeptAside(t *testing.T) {
	for name, content := range map[string]string{"garbage": "{not json", "future": `{"version": 7, "banks": []}`} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, HotbarFile)
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
			s := New(Options{Hotbar: NewHotbar(dir)})
			defer s.Close()
			if h := s.UIState().Hotbar; h == nil || len(h.Banks) != 1 || len(h.Banks[0].Clusters) != 0 {
				t.Fatalf("hotbar %+v", h)
			}
			if raw, _ := os.ReadFile(path); string(raw) != content {
				t.Fatal("the unreadable file was changed before any change of the hotbar")
			}
			if _, err := s.ChangeHotbar(func(h *HotbarState) error { return h.AddCluster(0, "a") }); err != nil {
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
	if _, err := s.ChangeHotbar(func(h *HotbarState) error { _, err := h.RemoveBank(0); return err }); err == nil {
		t.Fatal("removed the last bank")
	}
	// The file's place is a directory: the write fails.
	if err := os.Mkdir(filepath.Join(dir, HotbarFile), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ChangeHotbar(func(h *HotbarState) error { return h.AddCluster(0, "a") }); err == nil {
		t.Fatal("a failed write reported no error")
	}
	if u := s.UIState(); u.Revision != rev || len(u.Hotbar.Banks[0].Clusters) != 0 {
		t.Errorf("the UI state moved: revision %d → %d, %+v", rev, u.Revision, u.Hotbar)
	}
	if st, _ := s.HotbarState(); len(st.Banks[0].Clusters) != 0 {
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
			if err := h.AddCluster(0, c); err != nil {
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
	for _, tl := range v.Banks[0].Clusters {
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
	if u := s.UIState(); u.Hotbar == nil || u.Hotbar.Banks[0].Clusters[0] != (HotbarTile{Context: "test"}) {
		t.Errorf("UIState.Hotbar %+v", u.Hotbar)
	}
	// Connecting changes no hotbar; a change of the hotbar keeps the connection and the view.
	before := s.UIState()
	if _, err := s.ChangeHotbar(func(h *HotbarState) error { return h.MoveCluster(0, "gone [old.yaml]", 1) }); err != nil {
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
	done := make(chan error, HotbarBankSize)
	for i := range HotbarBankSize {
		go func() {
			_, err := s.ChangeHotbar(func(h *HotbarState) error { return h.AddCluster(0, fmt.Sprintf("c%d", i)) })
			done <- err
		}()
	}
	for range HotbarBankSize {
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
	if u := s.UIState(); len(st.Banks[0].Clusters) != HotbarBankSize || tiles(u.Hotbar.Banks[0]) != tiles(st.Banks[0]) {
		t.Errorf("stored %s, shown %s", tiles(st.Banks[0]), tiles(u.Hotbar.Banks[0]))
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
