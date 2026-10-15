package engine

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/guided-traffic/cogpit/internal/model"
)

// A Config Map's values edited in the drawer and not saved, kept in memory and as a draft on disk
// (the owner's answer of 2026-10-15 to the lead's question of D-55: "as the Edit tab's"). The
// drawer hands the edited values to configMap.keepEdits as the user types; the session keeps them
// for the drawer's next visit of the Config Map (configMap.readData returns them to the window),
// and the drafts (drafts.go) write them a moment after the typing stops into drafts/<id>.json,
// owner-only, removed when the user saves or reverts them. A quit or a crash leaves the draft,
// which is offered when its context connects (EventDrafts, editor.drafts): editor.restoreDraft
// opens the drawer on the Config Map with the values in its editors, based on the version they
// were edited at, editor.discardDraft deletes it. Only the window keeps such edits
// (configMap.keepEdits is GUI-only, and only the window reads them back): an agent cannot plant
// values the user would take for their own, and changes a Config Map with configMap.saveData,
// which the user sees. A Secret's values are never kept here, nor anywhere on disk (D-59): the
// window keeps them in its memory only.

// DraftConfigMapData is the type of a draft of a Config Map's values edited in the drawer: the
// drawer's, no dock tab's.
const DraftConfigMapData DockType = "configMapData"

// drawerDraftPrefix starts the key the drafts keep a Config Map's edits under, beside the editor
// tabs' ids.
const drawerDraftPrefix = "drawer\x00"

// ErrConfigMapEditsGUIOnly refuses an agent or the CLI configMap.keepEdits: unsaved values in the
// drawer are the user's typing in Cogpit's window.
var ErrConfigMapEditsGUIOnly = errors.New("the values a Config Map's drawer keeps unsaved are the user's typing in Cogpit's " +
	"own window: agents and the CLI cannot keep, plant or read them; an agent changes values with configMap_saveData")

// ConfigMapEdits are a Config Map's values edited in the drawer and not saved.
type ConfigMapEdits struct {
	// BaseResourceVersion is the version the values are based on.
	BaseResourceVersion string            `json:"baseResourceVersion"`
	Values              map[string]string `json:"values"`
	// From holds each edited key's value at BaseResourceVersion, for the review's rebase.
	From map[string]string `json:"from,omitempty"`
	// Restored: the values come from a draft an earlier Cogpit kept (editor.restoreDraft), not from
	// typing in this window since.
	Restored bool `json:"restored,omitempty"`
}

// ConfigMapDraftRestored is what editor.restoreDraft returns for a draft of a Config Map's values:
// which Config Map the drawer opened on, the version the values are based on and their keys,
// never a value.
type ConfigMapDraftRestored struct {
	Ref                 model.ObjectRef `json:"ref"`
	BaseResourceVersion string          `json:"baseResourceVersion"`
	Keys                []string        `json:"keys"`
}

// drawerEditStore holds the Config Maps' unsaved drawer edits of a session, by drawerEditKey.
type drawerEditStore struct {
	mu   sync.Mutex
	m    map[string]*keptEdits
	revs map[string]uint64 // the newest keep's revision per Config Map, kept after a drop
}

type keptEdits struct {
	ref   model.ObjectRef
	edits ConfigMapEdits
}

// drawerEditKey names a Config Map's edits: its context, namespace and name.
func drawerEditKey(ref model.ObjectRef) string {
	return ref.Context + "\x00" + ref.Namespace + "\x00" + ref.Name
}

// KeepConfigMapEdits keeps a Config Map's values edited in the drawer: in memory for the drawer's
// next visit, and as a draft (when the session keeps drafts) a moment after the last call. rev
// orders the window's calls, which run concurrently: a call older than the newest one for the
// Config Map is ignored. No values drop the edits and their draft. The Config Map is named, not
// looked up: the drawer keeps its edits when it closes because the user switched clusters.
func (s *Session) KeepConfigMapEdits(ref model.ObjectRef, rev uint64, e ConfigMapEdits) error {
	if ref.Kind != configMapsKind {
		return fmt.Errorf("%w: this applies to %s", ErrNotApplicable, configMapsKind)
	}
	if ref.Context == "" {
		if b := s.Backend(); b != nil {
			ref.Context = b.Context
		}
	}
	if ref.Context == "" || ref.Name == "" {
		return errors.New("give the Config Map with its context and name")
	}
	if err := checkKeptEdits(e); err != nil {
		return err
	}
	key := drawerEditKey(ref)
	st := &s.drawerEdits
	st.mu.Lock()
	if st.revs == nil {
		st.m, st.revs = map[string]*keptEdits{}, map[string]uint64{}
	}
	if last, ok := st.revs[key]; ok && rev <= last {
		st.mu.Unlock()
		return nil // an older call of the window's, overtaken
	}
	st.revs[key] = rev
	if len(e.Values) == 0 {
		delete(st.m, key)
	} else {
		e.Values, e.From, e.Restored = maps.Clone(e.Values), maps.Clone(e.From), false
		st.m[key] = &keptEdits{ref: ref, edits: e}
	}
	st.mu.Unlock()
	if d := s.opts.Drafts; d != nil {
		if len(e.Values) == 0 {
			d.dropDrawer(key)
		} else {
			d.schedule(drawerDraftPrefix + key)
		}
	}
	return nil
}

// checkKeptEdits bounds kept edits as a save is bounded.
func checkKeptEdits(e ConfigMapEdits) error {
	if len(e.Values) == 0 {
		return nil
	}
	if e.BaseResourceVersion == "" {
		return errors.New("give resourceVersion: the version of the Config Map the values are based on")
	}
	if len(e.Values) > ConfigMapDataKeys || len(e.From) > ConfigMapDataKeys {
		return fmt.Errorf("at most %d values", ConfigMapDataKeys)
	}
	size := 0
	for k, v := range e.Values {
		size += len(k) + len(v) + len(e.From[k])
	}
	if size > 2*maxConfigMapValues {
		return fmt.Errorf("the values are %s, more than a Config Map holds", model.FormatBytes(int64(size)))
	}
	return nil
}

// keptConfigMapEdits returns a copy of a Config Map's kept edits; nil when there are none.
func (s *Session) keptConfigMapEdits(ref model.ObjectRef) *ConfigMapEdits {
	st := &s.drawerEdits
	st.mu.Lock()
	defer st.mu.Unlock()
	k := st.m[drawerEditKey(ref)]
	if k == nil {
		return nil
	}
	e := k.edits
	e.Values, e.From = maps.Clone(e.Values), maps.Clone(e.From)
	return &e
}

// drawerEditsOf returns the kept edits under a drafts key and their Config Map.
func (s *Session) drawerEditsOf(key string) (model.ObjectRef, *ConfigMapEdits) {
	st := &s.drawerEdits
	st.mu.Lock()
	defer st.mu.Unlock()
	k := st.m[key]
	if k == nil {
		return model.ObjectRef{}, nil
	}
	e := k.edits
	e.Values, e.From = maps.Clone(e.Values), maps.Clone(e.From)
	return k.ref, &e
}

// flushDrawerLocked writes or removes the draft of a Config Map's kept edits; wmu is held.
func (d *Drafts) flushDrawerLocked(s *Session, tabID string, t *draftTab) {
	d.mu.Lock()
	id := t.id
	d.mu.Unlock()
	ref, e := s.drawerEditsOf(strings.TrimPrefix(tabID, drawerDraftPrefix))
	if e == nil {
		d.mu.Lock()
		if d.tabs[tabID] == t {
			delete(d.tabs, tabID)
		}
		d.mu.Unlock()
		if id != "" {
			d.removeFile(id)
		}
		return
	}
	if id == "" {
		var err error
		if id, err = newDraftID(); err != nil {
			d.warn(err)
			return
		}
		d.mu.Lock()
		if t.gone {
			d.mu.Unlock()
			return
		}
		t.id = id
		d.mu.Unlock()
	}
	r := ref
	d.write(Draft{Version: draftVersion, ID: id, Type: DraftConfigMapData, Context: ref.Context, Title: "ConfigMap " + ref.Name,
		Ref: &r, APIVersion: "v1", Kind: configMapAPIKind, BaseResourceVersion: e.BaseResourceVersion, Saved: d.now().UTC(),
		Owner: d.self(), Values: e.Values, From: e.From})
}

// dropDrawer removes the draft of a Config Map's edits, which were saved or reverted.
func (d *Drafts) dropDrawer(key string) {
	tabID := drawerDraftPrefix + key
	d.wmu.Lock()
	defer d.wmu.Unlock()
	d.mu.Lock()
	t := d.tabs[tabID]
	id := ""
	if t != nil {
		if t.timer != nil {
			t.timer.Stop()
			t.timer = nil
		}
		t.gen++
		t.gone = true
		id = t.id
		delete(d.tabs, tabID)
	}
	d.mu.Unlock()
	if id != "" {
		d.removeFile(id)
	}
}

// DrawerDraft reports whether id names an offered draft of a Config Map's drawer values (restored
// by RestoreConfigMapDraft), not an editor tab's.
func (s *Session) DrawerDraft(id string) bool {
	d := s.opts.Drafts
	if d == nil {
		return false
	}
	d.wmu.Lock()
	defer d.wmu.Unlock()
	dr, err := d.read(id)
	return err == nil && dr.Type == DraftConfigMapData
}

// RestoreConfigMapDraft gives the session an offered draft of a Config Map's drawer values: the
// window's drawer shows them in its editors the next time it reads the Config Map, based on the
// version they were edited at, and the draft is this session's from then on. The caller opens the
// drawer. A Config Map with unsaved drawer values of this session's own is refused.
func (s *Session) RestoreConfigMapDraft(_ context.Context, id string) (ConfigMapDraftRestored, error) {
	d := s.opts.Drafts
	if d == nil {
		return ConfigMapDraftRestored{}, ErrNoDrafts
	}
	d.wmu.Lock()
	dr, err := d.offeredDraft(id)
	if err == nil && dr.Type != DraftConfigMapData {
		err = fmt.Errorf("the draft %s is an editor tab's", id)
	}
	if err != nil {
		d.wmu.Unlock()
		return ConfigMapDraftRestored{}, err
	}
	b := s.Backend()
	switch {
	case b == nil:
		err = ErrNotConnected
	case b.Context != dr.Context:
		err = fmt.Errorf("%w: the draft of %s belongs to context %q; connect to it to restore the draft", ErrWrongCluster, dr.what(), dr.Context)
	}
	if err != nil {
		d.wmu.Unlock()
		return ConfigMapDraftRestored{}, err
	}
	ref := *dr.Ref
	key := drawerEditKey(ref)
	st := &s.drawerEdits
	st.mu.Lock()
	if st.m[key] != nil {
		st.mu.Unlock()
		d.wmu.Unlock()
		return ConfigMapDraftRestored{}, fmt.Errorf("the drawer holds unsaved values of %s of its own; save or revert them, then restore the draft", dr.what())
	}
	if st.revs == nil {
		st.m, st.revs = map[string]*keptEdits{}, map[string]uint64{}
	}
	st.m[key] = &keptEdits{ref: ref, edits: ConfigMapEdits{BaseResourceVersion: dr.BaseResourceVersion,
		Values: maps.Clone(dr.Values), From: maps.Clone(dr.From), Restored: true}}
	st.mu.Unlock()

	tabID := drawerDraftPrefix + key
	d.mu.Lock()
	t := &draftTab{id: dr.ID}
	d.tabs[tabID] = t
	d.mu.Unlock()
	d.flushDrawerLocked(s, tabID, t) // rewritten at once, naming this process
	d.wmu.Unlock()
	d.changed("")
	return ConfigMapDraftRestored{Ref: ref, BaseResourceVersion: dr.BaseResourceVersion, Keys: slices.Sorted(maps.Keys(dr.Values))}, nil
}

// drawerDraftInfo fills a draft listing's fields of a Config Map's drawer values: their keys and
// size, never a value.
func drawerDraftInfo(dr Draft, info DraftInfo) DraftInfo {
	info.Keys = slices.Sorted(maps.Keys(dr.Values))
	info.Lines, info.Bytes = 0, 0
	for _, v := range dr.Values {
		info.Bytes += len(v)
	}
	return info
}

// checkDrawerDraft checks a draft of a Config Map's drawer values read back.
func checkDrawerDraft(dr Draft) error {
	switch {
	case dr.Ref == nil || dr.Ref.Kind != configMapsKind || dr.Ref.Name == "":
		return fmt.Errorf("the draft %s names no Config Map", dr.ID)
	case len(dr.Values) == 0:
		return fmt.Errorf("the draft %s holds no values", dr.ID)
	}
	return checkKeptEdits(ConfigMapEdits{BaseResourceVersion: dr.BaseResourceVersion, Values: dr.Values, From: dr.From})
}
