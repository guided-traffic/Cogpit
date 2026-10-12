package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

// The hotbar (G13, docs/parity/ui-parity.md › Hotbar): the clusters the user pinned to the bar at
// the window's left edge, in banks of up to twelve, one shown at a time. Captured from Lens: one
// tile per cluster with a short label, a bank switcher ("‹ 1 ›") at the bottom, up to 12 clusters
// per bank, banks created, renamed, removed and switched, F1–F12 open a cluster of the shown bank
// and ⇧F1–⇧F12 switch to a bank. Inferred (Cogpit's): a bank's tiles are packed in order without
// empty cells; at most 12 banks, the ones ⇧F1–⇧F12 reach; the first bank is called "Default";
// a tile whose context no kubeconfig source offers any more stays, marked missing.
//
// The hotbar is engine state (UIState.Hotbar) so that agents see and drive what the user sees; the
// GUI changes it only through the hotbar.* commands. It is Cogpit's own preference: kept in a file
// under Cogpit's configuration directory (HotbarFile, owner-only, replaced atomically), never in a
// kubeconfig, and no hotbar command talks to a cluster except hotbar.openCluster, which connects
// through the same path as cluster.connect.

// Hotbar limits and names.
const (
	// HotbarFile is the hotbar's file under Cogpit's configuration directory.
	HotbarFile = "hotbar.json"
	// HotbarBankSize is the most clusters a bank holds: one per key, F1 to F12 (Lens's limit).
	HotbarBankSize = 12
	// HotbarMaxBanks is the most banks the hotbar holds: one per key, ⇧F1 to ⇧F12 (Cogpit's limit).
	HotbarMaxBanks = 12
	// DefaultHotbarBank is the name of the bank a new hotbar starts with.
	DefaultHotbarBank = "Default"
	// maxBankName bounds a bank's name, in characters.
	maxBankName = 64
	// maxHotbarContext bounds a stored context id, in bytes: longer ones are dropped on load.
	maxHotbarContext = 4096
	hotbarVersion    = 1
)

// ErrNoHotbar disables the hotbar commands of a Cogpit without a hotbar (a headless session).
var ErrNoHotbar = errors.New("this Cogpit keeps no hotbar")

// HotbarState is the hotbar: its banks and the one shown. Banks are numbered from 1, as the bank
// switcher and ⇧F1–⇧F12 number them; tiles likewise, F1 being the first.
type HotbarState struct {
	Banks []HotbarBank `json:"banks"`
	// Current is the number of the shown bank, from 1.
	Current int `json:"current"`
}

// HotbarBank is one bank: a name and up to HotbarBankSize tiles, top to bottom.
type HotbarBank struct {
	Name     string       `json:"name"`
	Clusters []HotbarTile `json:"clusters"`
}

// HotbarTile is one cluster on the hotbar. Only Context is stored; the other fields describe the
// tile as it is now (Session.HotbarView: ui.state, hotbar.list) and are empty in UIState.Hotbar.
type HotbarTile struct {
	// Context is the context id as cluster.contexts lists it: "<context> [<file>]", or "demo".
	Context string `json:"context"`
	// Label is the tile's short label: the context's name without its file.
	Label string `json:"label,omitempty"`
	// Connection is the state of a connection to this context: Cogpit holds one at a time, so every
	// tile but the active cluster's is disconnected.
	Connection ConnectionState `json:"connection,omitempty"`
	// Missing is set when no kubeconfig source offers the context any more (its file or folder
	// went away, or the context was renamed): the tile stays until the user removes it.
	Missing bool `json:"missing,omitempty"`
	// Exec is the approval state of the context's credential plugin (ContextInfo.Exec), so that the
	// GUI shows the plugin for approval before it connects; empty when it needs none.
	Exec string `json:"exec,omitempty"`
	// ExecKind says what needs the approval (ContextInfo.ExecKind).
	ExecKind string `json:"execKind,omitempty"`
}

// Clone returns a deep copy.
func (h HotbarState) Clone() HotbarState {
	out := HotbarState{Current: h.Current, Banks: make([]HotbarBank, len(h.Banks))}
	for i, b := range h.Banks {
		out.Banks[i] = HotbarBank{Name: b.Name, Clusters: slices.Clone(b.Clusters)}
	}
	return out
}

// stored is the state as it is kept: tiles with their context only.
func (h HotbarState) stored() HotbarState {
	out := h.Clone()
	for i := range out.Banks {
		for j := range out.Banks[i].Clusters {
			out.Banks[i].Clusters[j] = HotbarTile{Context: out.Banks[i].Clusters[j].Context}
		}
	}
	return out
}

// defaultHotbar is the hotbar of a first start: one empty bank, shown.
func defaultHotbar() HotbarState {
	return HotbarState{Banks: []HotbarBank{{Name: DefaultHotbarBank, Clusters: []HotbarTile{}}}, Current: 1}
}

// normalize makes a state read from a file one that the commands can work on: at least one bank and
// at most HotbarMaxBanks, each named, with at most HotbarBankSize tiles of distinct, non-empty
// contexts, and the shown bank one of them.
func (h *HotbarState) normalize() {
	if len(h.Banks) > HotbarMaxBanks {
		h.Banks = h.Banks[:HotbarMaxBanks]
	}
	if len(h.Banks) == 0 {
		*h = defaultHotbar()
		return
	}
	for i := range h.Banks {
		b := &h.Banks[i]
		if name, err := cleanBankName(b.Name); err != nil {
			b.Name = fmt.Sprintf("Bank %d", i+1)
		} else {
			b.Name = name
		}
		var tiles []HotbarTile
		for _, t := range b.Clusters {
			c := t.Context
			if strings.TrimSpace(c) == "" || len(c) > maxHotbarContext || !utf8.ValidString(c) ||
				slices.ContainsFunc(tiles, func(o HotbarTile) bool { return o.Context == c }) || len(tiles) == HotbarBankSize {
				continue
			}
			tiles = append(tiles, HotbarTile{Context: c})
		}
		if tiles == nil {
			tiles = []HotbarTile{}
		}
		b.Clusters = tiles
	}
	if h.Current < 1 || h.Current > len(h.Banks) {
		h.Current = 1
	}
}

// cleanBankName trims a bank's name and refuses an empty, overlong or multi-line one.
func cleanBankName(name string) (string, error) {
	name = strings.TrimSpace(name)
	switch {
	case name == "":
		return "", errors.New("a bank needs a name")
	case !utf8.ValidString(name):
		return "", errors.New("a bank's name must be text")
	case utf8.RuneCountInString(name) > maxBankName:
		return "", fmt.Errorf("a bank's name has at most %d characters", maxBankName)
	case strings.IndexFunc(name, unicode.IsControl) >= 0:
		return "", errors.New("a bank's name is one line without control characters")
	}
	return name, nil
}

// bankIndex resolves a bank number (from 1; 0 is the shown bank) to an index into Banks.
func (h HotbarState) bankIndex(bank int) (int, error) {
	if bank == 0 {
		bank = h.Current
	}
	if bank < 1 || bank > len(h.Banks) {
		return 0, fmt.Errorf("the hotbar has no bank %d (it has %s)", bank, banksText(len(h.Banks)))
	}
	return bank - 1, nil
}

func banksText(n int) string {
	if n == 1 {
		return "1 bank"
	}
	return fmt.Sprintf("%d banks", n)
}

// bankNumber is the number of a bank argument: the shown bank's for 0.
func (h HotbarState) bankNumber(bank int) int {
	if bank == 0 {
		return h.Current
	}
	return bank
}

// Bank returns a bank by number (0: the shown bank).
func (h HotbarState) Bank(bank int) (HotbarBank, error) {
	i, err := h.bankIndex(bank)
	if err != nil {
		return HotbarBank{}, err
	}
	return h.Banks[i], nil
}

// Position returns where context is in the bank, from 1; 0 when it is not there.
func (b HotbarBank) Position(context string) int {
	return slices.IndexFunc(b.Clusters, func(t HotbarTile) bool { return t.Context == context }) + 1
}

// Tile returns the tile in a bank's slot (from 1: F1).
func (h HotbarState) Tile(bank, slot int) (HotbarTile, error) {
	b, err := h.Bank(bank)
	if err != nil {
		return HotbarTile{}, err
	}
	if slot < 1 || slot > len(b.Clusters) {
		return HotbarTile{}, fmt.Errorf("bank %d has no cluster in slot %d (it has %d)", h.bankNumber(bank), slot, len(b.Clusters))
	}
	return b.Clusters[slot-1], nil
}

// CheckAdd says why context cannot be added to a bank: no such bank, already in it (wrapping
// ErrNotApplicable, so that menus offer Remove instead), or the bank is full.
func (h HotbarState) CheckAdd(bank int, context string) error {
	b, err := h.Bank(bank)
	if err != nil {
		return err
	}
	if b.Position(context) > 0 {
		return fmt.Errorf("%w: %s is in bank %d already", ErrNotApplicable, context, h.bankNumber(bank))
	}
	if len(b.Clusters) >= HotbarBankSize {
		return fmt.Errorf("bank full: %d clusters", HotbarBankSize)
	}
	return nil
}

// AddCluster adds context at the end of a bank.
func (h *HotbarState) AddCluster(bank int, context string) error {
	if strings.TrimSpace(context) == "" {
		return errors.New("no context to add")
	}
	if err := h.CheckAdd(bank, context); err != nil {
		return err
	}
	i, _ := h.bankIndex(bank)
	h.Banks[i].Clusters = append(h.Banks[i].Clusters, HotbarTile{Context: context})
	return nil
}

// CheckInBank says why a tile command does not apply: no such bank, or context is not in it
// (wrapping ErrNotApplicable).
func (h HotbarState) CheckInBank(bank int, context string) error {
	b, err := h.Bank(bank)
	if err != nil {
		return err
	}
	if b.Position(context) == 0 {
		return fmt.Errorf("%w: %s is not in bank %d", ErrNotApplicable, context, h.bankNumber(bank))
	}
	return nil
}

// RemoveCluster removes context's tile from a bank.
func (h *HotbarState) RemoveCluster(bank int, context string) error {
	if err := h.CheckInBank(bank, context); err != nil {
		return err
	}
	i, _ := h.bankIndex(bank)
	b := &h.Banks[i]
	b.Clusters = slices.Delete(b.Clusters, b.Position(context)-1, b.Position(context))
	return nil
}

// CheckMove says why context cannot move to position (from 1) in its bank.
func (h HotbarState) CheckMove(bank int, context string, position int) error {
	if err := h.CheckInBank(bank, context); err != nil {
		return err
	}
	b, _ := h.Bank(bank)
	if position < 1 || position > len(b.Clusters) {
		return fmt.Errorf("position %d is outside the bank's 1 to %d", position, len(b.Clusters))
	}
	return nil
}

// MoveCluster moves context's tile to position (from 1) in its bank; the tiles between shift.
func (h *HotbarState) MoveCluster(bank int, context string, position int) error {
	if err := h.CheckMove(bank, context, position); err != nil {
		return err
	}
	i, _ := h.bankIndex(bank)
	b := &h.Banks[i]
	from := b.Position(context) - 1
	t := b.Clusters[from]
	b.Clusters = slices.Delete(b.Clusters, from, from+1)
	b.Clusters = slices.Insert(b.Clusters, position-1, t)
	return nil
}

// CheckCreateBank says why no bank can be added.
func (h HotbarState) CheckCreateBank() error {
	if len(h.Banks) >= HotbarMaxBanks {
		return fmt.Errorf("the hotbar has %d banks, the most that ⇧F1–⇧F12 reach", HotbarMaxBanks)
	}
	return nil
}

// CreateBank adds a bank after the others and shows it; an empty name gives "Bank <n>". It returns
// the new bank's number.
func (h *HotbarState) CreateBank(name string) (int, error) {
	if err := h.CheckCreateBank(); err != nil {
		return 0, err
	}
	n := len(h.Banks) + 1
	if strings.TrimSpace(name) == "" {
		name = fmt.Sprintf("Bank %d", n)
	}
	name, err := cleanBankName(name)
	if err != nil {
		return 0, err
	}
	h.Banks = append(h.Banks, HotbarBank{Name: name, Clusters: []HotbarTile{}})
	h.Current = n
	return n, nil
}

// RenameBank names a bank.
func (h *HotbarState) RenameBank(bank int, name string) error {
	i, err := h.bankIndex(bank)
	if err != nil {
		return err
	}
	name, err = cleanBankName(name)
	if err != nil {
		return err
	}
	h.Banks[i].Name = name
	return nil
}

// CheckRemoveBank says why a bank cannot be removed: no such bank, or it is the last one.
func (h HotbarState) CheckRemoveBank(bank int) error {
	if _, err := h.bankIndex(bank); err != nil {
		return err
	}
	if len(h.Banks) == 1 {
		return errors.New("the hotbar keeps at least one bank")
	}
	return nil
}

// RemoveBank removes a bank and returns it. The banks after it move up one number; the shown bank
// stays shown, and when it is the one removed, the bank that takes its number is (or the new last).
func (h *HotbarState) RemoveBank(bank int) (HotbarBank, error) {
	if err := h.CheckRemoveBank(bank); err != nil {
		return HotbarBank{}, err
	}
	i, _ := h.bankIndex(bank)
	removed := h.Banks[i]
	h.Banks = slices.Delete(h.Banks, i, i+1)
	switch {
	case i+1 < h.Current:
		h.Current--
	case h.Current > len(h.Banks):
		h.Current = len(h.Banks)
	}
	return removed, nil
}

// SwitchBank shows a bank.
func (h *HotbarState) SwitchBank(bank int) error {
	if bank < 1 {
		return fmt.Errorf("banks are numbered from 1, not %d", bank)
	}
	i, err := h.bankIndex(bank)
	if err != nil {
		return err
	}
	h.Current = i + 1
	return nil
}

// Hotbar keeps the hotbar in HotbarFile under one directory (the app's: <user config dir>/Cogpit).
// It is safe for concurrent use. Two Cogpit processes keep one hotbar each in memory and write the
// file on every change, so the last change wins (the file is not locked, as kubeconfigs.json).
type Hotbar struct {
	dir string // empty: kept in memory only

	// change serialises ChangeHotbar from the write to the UI state it shows, so that the UI states
	// follow the order of the writes; mu guards the state alone, so that a subscriber of the UI
	// state may read the hotbar while a change is shown.
	change sync.Mutex
	mu     sync.Mutex
	state  HotbarState
	loaded bool
	// unreadable is set when the file did not load: it is moved aside to <file>.unreadable before
	// the first save, so that a change never silently overwrites what the user may want back.
	unreadable bool
}

// NewHotbar returns the hotbar kept in dir; an empty dir keeps it in memory only. Nothing is read
// or written until the session attaches it.
func NewHotbar(dir string) *Hotbar { return &Hotbar{dir: dir} }

// Path returns the hotbar's file; empty when it is kept in memory only.
func (h *Hotbar) Path() string {
	if h.dir == "" {
		return ""
	}
	return filepath.Join(h.dir, HotbarFile)
}

type hotbarDoc struct {
	Version int          `json:"version"`
	Current int          `json:"current"`
	Banks   []HotbarBank `json:"banks"`
}

// load reads the file once: a missing file is the default hotbar; an unreadable one, too, and is
// reported. h.mu is held.
func (h *Hotbar) load() error {
	if h.loaded {
		return nil
	}
	h.loaded = true
	h.state = defaultHotbar()
	path := h.Path()
	if path == "" {
		return nil
	}
	var doc hotbarDoc
	if err := readPrivateJSON(path, maxPrivateDoc, &doc); err != nil {
		h.unreadable = true
		return fmt.Errorf("the hotbar file %s is unreadable (%v); Cogpit starts with an empty hotbar and moves the file to %s.unreadable at the first change", path, err, HotbarFile)
	}
	if doc.Version != 0 && doc.Version != hotbarVersion {
		h.unreadable = true
		return fmt.Errorf("the hotbar file %s has format version %d, this Cogpit reads %d; it starts with an empty hotbar and moves the file to %s.unreadable at the first change", path, doc.Version, hotbarVersion, HotbarFile)
	}
	if doc.Version == 0 && doc.Banks == nil { // a missing file
		return nil
	}
	st := HotbarState{Banks: doc.Banks, Current: doc.Current}
	st.normalize()
	h.state = st.stored()
	return nil
}

// save writes st to the file, owner-only and atomically (writePrivateFile). h.mu is held.
func (h *Hotbar) save(st HotbarState) error {
	path := h.Path()
	if path == "" {
		return nil
	}
	if h.unreadable {
		if err := os.Rename(path, path+".unreadable"); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("hotbar: keeping the unreadable file aside: %w", err)
		}
		h.unreadable = false
	}
	st = st.stored()
	data, err := json.MarshalIndent(hotbarDoc{Version: hotbarVersion, Current: st.Current, Banks: st.Banks}, "", "  ")
	if err != nil {
		return err
	}
	if err := writePrivateFile(path, append(data, '\n')); err != nil {
		return fmt.Errorf("hotbar: %w", err)
	}
	return nil
}

// attach loads the hotbar into the session's UI state; nil-safe.
func (h *Hotbar) attach(s *Session) {
	if h == nil {
		return
	}
	h.mu.Lock()
	err := h.load()
	st := h.state.Clone()
	h.mu.Unlock()
	if err != nil {
		s.log.Warn("hotbar", "err", err)
	}
	s.ui.Hotbar = &st
}

// Hotbar returns the session's hotbar; nil when it keeps none.
func (s *Session) Hotbar() *Hotbar { return s.opts.Hotbar }

// HotbarState returns the hotbar as stored: tiles with their contexts only.
func (s *Session) HotbarState() (HotbarState, error) {
	h := s.opts.Hotbar
	if h == nil {
		return HotbarState{}, ErrNoHotbar
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if err := h.load(); err != nil {
		s.log.Warn("hotbar", "err", err)
	}
	return h.state.Clone(), nil
}

// ChangeHotbar applies fn to a copy of the hotbar, writes it to the hotbar's file and shows it
// (UIState.Hotbar, EventUIState). If fn or the write fails, nothing changes. Changes are serialised,
// so the UI states follow the order of the writes.
func (s *Session) ChangeHotbar(fn func(*HotbarState) error) (HotbarState, error) {
	h := s.opts.Hotbar
	if h == nil {
		return HotbarState{}, ErrNoHotbar
	}
	h.change.Lock()
	defer h.change.Unlock()
	next, err := func() (HotbarState, error) {
		h.mu.Lock()
		defer h.mu.Unlock()
		if err := h.load(); err != nil {
			s.log.Warn("hotbar", "err", err)
		}
		next := h.state.Clone()
		if err := fn(&next); err != nil {
			return HotbarState{}, err
		}
		next = next.stored()
		if err := h.save(next); err != nil {
			return HotbarState{}, err
		}
		h.state = next
		return next.Clone(), nil
	}()
	if err != nil {
		return HotbarState{}, err
	}
	shown := next.Clone()
	s.UpdateUI(func(u *UIState) { u.Hotbar = &shown })
	return next, nil
}

// HotbarView returns the hotbar as the user sees it: each tile with its label, the state of a
// connection to it, whether its context is missing, and its credential plugin's approval. A
// context counts as missing only when the contexts could be listed without an error: a kubeconfig
// that fails to load leaves its tiles as they are.
func (s *Session) HotbarView() (HotbarState, error) {
	st, err := s.HotbarState()
	if err != nil {
		return st, err
	}
	known := map[string]ContextInfo{}
	ctxs, cerr := s.Contexts()
	for _, c := range ctxs {
		known[c.Name] = c
	}
	u := s.UIState()
	for i := range st.Banks {
		for j := range st.Banks[i].Clusters {
			t := &st.Banks[i].Clusters[j]
			t.Label = ContextLabel(t.Context)
			t.Connection = Disconnected
			if t.Context == u.Context {
				t.Connection = u.Connection.State
			}
			if c, ok := known[t.Context]; ok {
				t.Exec, t.ExecKind = c.Exec, c.ExecKind
			} else if cerr == nil && t.Context != u.Context {
				t.Missing = true
			}
		}
	}
	return st, nil
}

// ResolveContext returns the id of the context name names: itself when a source or the demo
// offers it, or the one source context with that bare name (D-28), as Connect resolves it.
func (s *Session) ResolveContext(name string) (string, error) {
	if r, ok := s.opts.Connector.(contextResolver); ok {
		return r.ResolveContext(name)
	}
	return name, nil
}

// ContextLabel is the short label of a context id: the context's name without the file it comes
// from ("kind-dev [.kube/config]" → "kind-dev").
func ContextLabel(id string) string {
	if i := strings.LastIndex(id, " ["); i > 0 && strings.HasSuffix(id, "]") {
		return id[:i]
	}
	return id
}
