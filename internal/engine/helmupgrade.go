package engine

import (
	"context"
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/helm"
	"github.com/guided-traffic/cogpit/internal/model"
)

// The Helm Upgrade dock tab (architecture §8, G11): Lens's "Helm Upgrade: <name>" tab (captured
// 2026-10-12) holds a release's user-supplied values in an editor whose text lives in the engine
// (editor.get, editor.setText, as Edit tabs), with the chart and the version the upgrade installs
// on the tab (DockTab.Helm). The values come from the cache: the newest revision's release Secret,
// decoded. The tab remembers the revision its values are based on, as an Edit tab remembers its
// object's resourceVersion: helm.release.upgrade refuses to upgrade a release that moved to
// another revision since, so that an upgrade never overwrites a change the tab has not shown.

// helmUpgradeTitle starts a Helm Upgrade tab's title, as Lens titles it.
const helmUpgradeTitle = "Helm Upgrade: "

// helmBase is the revision a Helm Upgrade tab's values are based on: the release's newest
// revision, and that revision's release Secret's UID, which "" stands for while the cache has not
// seen the revision an upgrade from the tab made.
type helmBase struct {
	revision int
	uid      string
}

// cachedRelease returns a release object and the release Secret of its newest revision from the
// cache, waiting (bounded by ctx and syncTimeout) for the release Secrets' initial list.
func (s *Session) cachedRelease(ctx context.Context, ref model.ObjectRef) (*Backend, model.ObjectRef, *unstructured.Unstructured, *unstructured.Unstructured, error) {
	if ref.Kind != helm.ReleaseKind {
		return nil, ref, nil, nil, fmt.Errorf("%w: %s is not a Helm release", ErrNotApplicable, ref.String())
	}
	b, ref, p, err := s.resolve(ref)
	if err != nil {
		return nil, ref, nil, nil, err
	}
	h := b.Acquire(p)
	defer h.Release()
	if err := waitSynced(ctx, h, p.Key); err != nil {
		return nil, ref, nil, nil, err
	}
	obj, ok := KindGet(h, p, ref.Namespace, ref.Name)
	if !ok {
		return nil, ref, nil, nil, fmt.Errorf("%s: %w", ref.String(), ErrNotFound)
	}
	secret, ok := h.Get(ref.Namespace, helm.ReleaseSecret(obj))
	if !ok {
		return nil, ref, nil, nil, fmt.Errorf("%s: the release Secret of revision %d: %w", ref.String(), helm.ReleaseRevision(obj), ErrNotFound)
	}
	return b, ref, obj, secret, nil
}

// ReleaseValues returns a release's values from the cache: the user-supplied ones, or with all
// every value with the chart's defaults. The caller decides who may see them (helm.release.values
// withholds them from agents).
func (s *Session) ReleaseValues(ctx context.Context, ref model.ObjectRef, all bool) (helm.Values, error) {
	_, ref, _, secret, err := s.cachedRelease(ctx, ref)
	if err != nil {
		return helm.Values{}, err
	}
	v, err := helm.ReleaseValues(secret, all)
	if err != nil {
		return helm.Values{}, fmt.Errorf("release %s: %w", ref.Name, err)
	}
	return v, nil
}

// OpenHelmUpgradeTab opens a release's Helm Upgrade tab, its editor filled with the release's
// user-supplied values from the cache, or activates the tab already open on it.
func (s *Session) OpenHelmUpgradeTab(ctx context.Context, ref model.ObjectRef) (EditorState, bool, error) {
	b, ref, obj, secret, err := s.cachedRelease(ctx, ref)
	if err != nil {
		return EditorState{}, false, err
	}
	vals, err := helm.ReleaseValues(secret, false)
	if err != nil {
		return EditorState{}, false, fmt.Errorf("release %s: %w", ref.Name, err)
	}
	if len(vals.YAML) > MaxManifestBytes {
		return EditorState{}, false, fmt.Errorf("release %s: %d bytes of values is more than the %d an editor takes", ref.Name, len(vals.YAML), MaxManifestBytes)
	}
	chart, version := helm.ReleaseChart(obj)
	r := ref
	e := &editor{
		tab: DockTab{
			ID: s.NewDockID(DockHelmUpgrade), Type: DockHelmUpgrade, Title: helmUpgradeTitle + ref.Name, Context: b.Context, Ref: &r,
			Helm: &HelmUpgradeTab{Chart: chart, Version: version, Versions: []string{version}, UserSuppliedOnly: true},
		},
		text: vals.YAML, rev: 1,
		helm: &helmBase{revision: helm.ReleaseRevision(obj), uid: string(obj.GetUID())},
	}
	s.putEditor(e)
	tab, created := e.tab, true
	ui := s.UpdateUI(func(u *UIState) {
		for _, x := range u.Dock {
			if x.Type == DockHelmUpgrade && x.Ref != nil && x.Ref.Same(ref) && x.Context == ref.Context {
				tab, created = x, false
				break
			}
		}
		if created {
			u.Dock = append(u.Dock, tab)
		}
		u.ActiveDock = tab.ID
	})
	s.settleEditor(e.tab.ID, created, ui.Revision)
	st, err := s.EditorState(ctx, tab.ID)
	return st, created, err
}

// helmEditorState fills an editor state of a Helm Upgrade tab with the revision its values are
// based on and what the cache holds now: a release at another revision is stale (an upgrade would
// be refused), a release that is gone missing.
func (s *Session) helmEditorState(st *EditorState, e editor) {
	if e.helm == nil || e.tab.Ref == nil {
		return
	}
	st.BaseRevision = e.helm.revision
	b := s.Backend()
	if b == nil || b.Context != e.tab.Context {
		return
	}
	obj, err := s.Lookup(*e.tab.Ref)
	switch {
	case errors.Is(err, ErrNotFound):
		st.Missing = true
	case err == nil:
		st.CachedRevision = helm.ReleaseRevision(obj)
		st.Stale = helmMoved(*e.helm, obj)
	}
}

// helmMoved reports whether a release object is no longer at the revision a tab's values are
// based on: another revision, or the same number on another release Secret (uninstalled and
// installed again).
func helmMoved(base helmBase, obj *unstructured.Unstructured) bool {
	return helm.ReleaseRevision(obj) != base.revision || (base.uid != "" && base.uid != string(obj.GetUID()))
}

// HelmUpgradeTabInfo is a Helm Upgrade tab as helm.release.upgrade uses it: the tab, its values
// and the revision they are based on.
type HelmUpgradeTabInfo struct {
	Tab          DockTab
	Values       string
	TextRevision uint64 // the editor's text revision Values is at
	BaseRevision int
	baseUID      string
}

// HelmUpgradeTab returns an open Helm Upgrade tab's state.
func (s *Session) HelmUpgradeTab(id string) (HelmUpgradeTabInfo, error) {
	e, err := s.editorOf(id)
	if err != nil {
		return HelmUpgradeTabInfo{}, err
	}
	if e.tab.Type != DockHelmUpgrade || e.helm == nil || e.tab.Ref == nil {
		return HelmUpgradeTabInfo{}, fmt.Errorf("dock tab %q is a %s tab, not a Helm Upgrade tab", id, e.tab.Type)
	}
	return HelmUpgradeTabInfo{Tab: e.tab, Values: e.text, TextRevision: e.rev, BaseRevision: e.helm.revision, baseUID: e.helm.uid}, nil
}

// CheckReleaseBase fails with ErrTargetsChanged when a release object is no longer at the revision
// a tab's (or a caller's) values are based on: uid "" checks the revision only.
func CheckReleaseBase(obj *unstructured.Unstructured, revision int, uid string) error {
	if !helmMoved(helmBase{revision: revision, uid: uid}, obj) {
		return nil
	}
	return fmt.Errorf("%w: release %s is at revision %d now, the values are based on revision %d; nothing was done: read the values again",
		ErrTargetsChanged, obj.GetName(), helm.ReleaseRevision(obj), revision)
}

// CheckTabBase is CheckReleaseBase for a Helm Upgrade tab.
func (t HelmUpgradeTabInfo) CheckTabBase(obj *unstructured.Unstructured) error {
	return CheckReleaseBase(obj, t.BaseRevision, t.baseUID)
}

// RebaseHelmUpgradeTab moves a Helm Upgrade tab's base to the revision an upgrade from it made,
// with that revision's release Secret's UID once the cache has seen it (uid "" until then).
func (s *Session) RebaseHelmUpgradeTab(id string, revision int, uid string) {
	_, _ = s.updateEditor(id, func(e *editor) {
		if e.helm != nil {
			e.helm = &helmBase{revision: revision, uid: uid} // a new value: copies of the editor share the old one
		}
	})
}
