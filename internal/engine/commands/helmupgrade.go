package commands

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/helm"
	"github.com/guided-traffic/cogpit/internal/model"
)

// Helm upgrade (G11, architecture §8): Lens's Upgrade opens a "Helm Upgrade: <name>" dock tab
// with the release's values (captured 2026-10-12); helm.release.upgrade upgrades the release with
// the tab's values, or with values a caller passes (the drawer's Save), through the Helm SDK in
// Cogpit's process, mutating (privilege-footprint.md), recorded in the Helm journal first (D-11).
// The values are the user's: agents read them withheld, and no log, error, toast, audit line or
// journal entry carries them.

// orderUpgrade places Upgrade between Rollback (200) and Delete in the drawer's toolbar and the
// row menu, as Lens's drawer orders its icons (captured 2026-10-12).
const orderUpgrade = 300

type upgradeTabParams struct {
	Ref model.ObjectRef `json:"ref" jsonschema:"the release: kind helmreleases, its namespace and name"`
}

type upgradeParams struct {
	Ref model.ObjectRef `json:"ref" jsonschema:"the release: kind helmreleases, its namespace and name"`
	Tab string          `json:"tab,omitempty" jsonschema:"a Helm Upgrade dock tab id as helm_release_upgradeTab returned it, e.g. helmUpgrade-1: upgrade with the tab's values (editor_setText changes them), based on the revision the tab read them from; give tab or values"`
	// Values is a pointer so that empty values (no user-supplied values) differ from none given.
	Values   *string `json:"values,omitempty" jsonschema:"the release's new user-supplied values as YAML, replacing the old ones whole (empty: none); give tab or values"`
	Revision int     `json:"revision,omitempty" jsonschema:"with values: the release's newest revision the values are based on (object_get or helm_release_history shows it); the upgrade is refused when the release moved past it"`
	// TextRevision binds an upgrade from a tab to the text the user reviewed: a change of the
	// tab's text after it (an agent's editor.setText while the confirmation is open) refuses it.
	TextRevision uint64 `json:"textRevision,omitempty" jsonschema:"with tab: the tab's text revision the upgrade takes the values at (editor_get and editor_setText return it); the upgrade is refused when the text changed since"`
	Version      string `json:"version,omitempty" jsonschema:"the chart version to install; empty: the release's own, the only one without chart repositories"`
}

type releaseValuesParams struct {
	Ref model.ObjectRef `json:"ref" jsonschema:"the release: kind helmreleases, its namespace and name"`
	All bool            `json:"all,omitempty" jsonschema:"every value, the chart's defaults included (Lens's User-supplied values only unchecked); default: the user-supplied values only"`
}

// ReleaseHistory is what helm.release.history returns: the release's newest revision and its
// history from the cache, newest first.
type ReleaseHistory struct {
	Ref      model.ObjectRef `json:"ref"`
	Revision int             `json:"revision"`
	Status   string          `json:"status"`
	// UID is the newest revision's release Secret's UID, which a rollback's confirmation binds.
	UID     string              `json:"uid"`
	History []helm.HistoryEntry `json:"history"`
}

// releaseCacheReads are the reads of the commands that read releases from the cache: the watch
// on the release Secrets.
var releaseCacheReads = []engine.Verb{{Verb: nwVerbList, Resource: kindSecrets}, {Verb: nwVerbWatch, Resource: kindSecrets}}

func registerHelmUpgrade(g *registrar) {
	add(g, engine.Spec[upgradeTabParams]{
		ID: helm.UpgradeTabCommand, Label: "Upgrade", MenuPath: "Helm/Upgrade", Kinds: []string{helm.ReleaseKind},
		Order: orderUpgrade, Danger: engine.DangerUI, Verbs: releaseCacheReads,
		Description: "Opens a release's Helm Upgrade dock tab, or activates the one already open on it: the release's chart, " +
			"the version the upgrade installs (the release's own: Cogpit has no chart repositories yet) and its user-supplied " +
			"values in an editor, read from Cogpit's cache. Writes nothing. Agents read the tab's values withheld " +
			"(editor_get) but may replace them (editor_setText); helm_release_upgrade with the tab id upgrades the release.",
		Enabled: func(s *engine.Session, p upgradeTabParams) error {
			if _, err := cachedOfKind(s, p.Ref, helm.ReleaseKind); err != nil {
				return err
			}
			b, err := writable(s)
			if err != nil {
				return err
			}
			_, err = b.HelmWriter()
			return err
		},
		Run: func(ctx context.Context, s *engine.Session, p upgradeTabParams) (any, error) {
			st, created, err := s.OpenHelmUpgradeTab(ctx, p.Ref)
			if err != nil {
				return nil, err
			}
			if engine.OriginOf(ctx).IsAgent() {
				st.Text = model.Withheld
			}
			return EditResult{Tab: st.Tab, Created: created, Text: st.Text, Revision: st.Revision}, nil
		},
	})

	add(g, engine.Spec[upgradeParams]{
		ID: "helm.release.upgrade", Label: "Upgrade", Kinds: []string{helm.ReleaseKind},
		Danger: engine.DangerMutating, Verbs: releaseSecretVerbs("list", "get", "create", "update", "delete"),
		Description: "Upgrades a Helm release as helm upgrade does with its defaults (hooks waited for, ten revisions kept), " +
			"with the release's own chart and new user-supplied values, which replace the old ones whole: the values of a " +
			"Helm Upgrade tab (tab), or values given here with the revision they are based on. Refused when the release " +
			"moved to another revision since the values were read, and while another Helm action on it is pending. " +
			"Besides the release's Secrets it needs whatever verbs the release's objects need. An agent's request waits " +
			"for the user's acceptance. An upgrade a crash of Cogpit interrupts stays in helm_pendingOperations; " +
			"helm_release_rollback to its fromRevision finishes it.",
		Enabled: func(s *engine.Session, p upgradeParams) error {
			obj, err := cachedOfKind(s, p.Ref, helm.ReleaseKind)
			if err != nil {
				return err
			}
			if st := helm.ReleaseStatus(obj); helm.Pending(st) {
				return fmt.Errorf("release %s is %s: another Helm action on it did not finish; finish or roll it back first", p.Ref.Name, st)
			}
			switch {
			case p.Tab != "" && p.Values != nil:
				return errors.New("give tab or values, not both")
			case p.Tab != "":
				if _, err := upgradeTab(s, p); err != nil {
					return err
				}
			case p.Values == nil:
				return errors.New("give the values: a Helm Upgrade tab (tab) or values with the revision they are based on")
			case p.Revision < 1:
				return errors.New("give the revision the values are based on (the release's newest revision when you read them)")
			}
			return helmWritable(s, p.Ref)
		},
		Targets: func(s *engine.Session, p upgradeParams) ([]model.ObjectRef, error) { return targetOf(s, p.Ref) },
		Detail: func(s *engine.Session, p upgradeParams) string {
			chart, version := "", p.Version
			if obj, err := s.Lookup(p.Ref); err == nil {
				name, own := helm.ReleaseChart(obj)
				chart = name
				if version == "" {
					version = own
				}
			}
			from := "the values"
			if p.Tab != "" {
				from = fmt.Sprintf("the values in the tab %s at its text revision %d (review them there; a change refuses the upgrade)", p.Tab, p.TextRevision)
			}
			return fmt.Sprintf("The release is upgraded to chart %s %s with %s; its objects change to the new manifest.", chart, version, from)
		},
		Run: runUpgrade,
	})

	add(g, engine.Spec[releaseValuesParams]{
		ID: "helm.release.values", Label: "Release Values", Danger: engine.DangerRead, Verbs: releaseCacheReads,
		Description: "Returns a release's values from Cogpit's cache, the newest revision's: the user-supplied ones, or all " +
			"of them with the chart's defaults (all), with the revision and chart they are of. Values can hold credentials: " +
			"agents read them withheld; the user sees them in Cogpit's own window.",
		Enabled: func(s *engine.Session, p releaseValuesParams) error {
			_, err := cachedOfKind(s, p.Ref, helm.ReleaseKind)
			return err
		},
		Run: func(ctx context.Context, s *engine.Session, p releaseValuesParams) (any, error) {
			v, err := s.ReleaseValues(ctx, p.Ref, p.All)
			if err != nil {
				return nil, err
			}
			if engine.OriginOf(ctx).IsAgent() {
				v.YAML = model.Withheld
			}
			return v, nil
		},
	})

	add(g, engine.Spec[upgradeTabParams]{
		ID: "helm.release.history", Label: "Release History", Danger: engine.DangerRead, Verbs: releaseCacheReads,
		Description: "Returns a release's revisions from Cogpit's cache, newest first (as many as Helm keeps, ten by default): " +
			"each revision's chart and version, app version, status, when it was deployed and its description, with the " +
			"release's newest revision and status. Rollback picks one of them.",
		Enabled: func(s *engine.Session, p upgradeTabParams) error {
			_, err := cachedOfKind(s, p.Ref, helm.ReleaseKind)
			return err
		},
		Run: func(ctx context.Context, s *engine.Session, p upgradeTabParams) (any, error) {
			ref, _, err := s.ResolveRef(p.Ref)
			if err != nil {
				return nil, err
			}
			obj, err := s.Object(ctx, ref)
			if err != nil {
				return nil, err
			}
			h := helm.ReleaseHistory(obj)
			if h == nil {
				h = []helm.HistoryEntry{}
			}
			return ReleaseHistory{Ref: ref, Revision: helm.ReleaseRevision(obj), Status: helm.ReleaseStatus(obj), UID: string(obj.GetUID()), History: h}, nil
		},
	})
}

// upgradeTab returns the Helm Upgrade tab an upgrade names, which must be one on its release.
func upgradeTab(s *engine.Session, p upgradeParams) (engine.HelmUpgradeTabInfo, error) {
	t, err := s.HelmUpgradeTab(p.Tab)
	if err != nil {
		return t, err
	}
	if !t.Tab.Ref.Same(p.Ref) {
		return t, fmt.Errorf("the tab %s upgrades release %s/%s, not %s/%s", p.Tab, t.Tab.Ref.Namespace, t.Tab.Ref.Name, p.Ref.Namespace, p.Ref.Name)
	}
	return t, nil
}

// rebaseWait bounds how long a successful upgrade from a tab waits for the cache to show the new
// revision, whose release Secret's UID becomes the tab's base.
const rebaseWait = 3 * time.Second

func runUpgrade(ctx context.Context, s *engine.Session, p upgradeParams) (any, error) {
	w, ref, obj, err := releaseTarget(ctx, s, p.Ref)
	if err != nil {
		return nil, err
	}
	text := ""
	var tab engine.HelmUpgradeTabInfo
	if p.Tab != "" {
		if tab, err = upgradeTab(s, p); err != nil {
			return nil, err
		}
		if p.TextRevision == 0 {
			return nil, fmt.Errorf("give textRevision, the revision of the tab %s's text the upgrade takes (editor_get shows it); nothing was done", p.Tab)
		}
		if tab.TextRevision != p.TextRevision {
			return nil, fmt.Errorf("%w: the values in the tab %s changed since text revision %d (now %d); nothing was done: review them and upgrade again",
				engine.ErrTargetsChanged, p.Tab, p.TextRevision, tab.TextRevision)
		}
		if err := tab.CheckTabBase(obj); err != nil {
			return nil, err
		}
		text = tab.Values
	} else {
		if p.Values == nil {
			return nil, errors.New("give tab or values")
		}
		if err := engine.CheckReleaseBase(obj, p.Revision, ""); err != nil {
			return nil, err
		}
		text = *p.Values
	}
	if st := helm.ReleaseStatus(obj); helm.Pending(st) {
		return nil, fmt.Errorf("release %s is %s: another Helm action on it did not finish; nothing was done", ref.Name, st)
	}
	chart, version := helm.ReleaseChart(obj)
	if p.Version != "" && p.Version != version {
		return nil, fmt.Errorf("release %s: chart %s %s: %w (%s)", ref.Name, chart, p.Version, helm.ErrNoRepository, version)
	}
	vals, err := helm.ParseValues(text)
	if err != nil {
		return nil, fmt.Errorf("release %s: %w; nothing was done", ref.Name, err)
	}
	cur := helm.ReleaseRevision(obj)
	op := engine.PendingHelmOp{
		Context: ref.Context, Namespace: ref.Namespace, Release: ref.Name, Action: engine.HelmUpgrade,
		FromRevision: cur, Chart: chart, ChartVersion: version, Started: s.Now().UTC(), StartedBy: startedBy(ctx),
	}
	var revision int
	err = runHelmOp(s, op, fmt.Sprintf("Upgrading release %s…", ref.Name), func() (err error) {
		defer func() {
			if r := recover(); r != nil { // scrubbed here: helmSafely would quote the panic as it is
				err = helm.Scrubbed(fmt.Errorf("internal error: %v", r), vals)
			}
		}()
		revision, err = w.UpgradeRelease(ctx, ref.Namespace, ref.Name, engine.HelmUpgradeRequest{ChartVersion: version, Values: vals})
		return helm.Scrubbed(err, vals)
	})
	if err != nil {
		return nil, err
	}
	if revision <= cur {
		revision = cur + 1
	}
	if p.Tab != "" {
		s.RebaseHelmUpgradeTab(p.Tab, revision, newRevisionUID(ctx, s, ref, revision))
	}
	userToast(s, engine.LevelInfo, fmt.Sprintf("Release %s upgraded to revision %d", ref.Name, revision))
	return HelmResult{Ref: ref, Action: "upgraded", Revision: revision}, nil
}

// newRevisionUID waits, bounded by rebaseWait, for the cache to show a release at a revision and
// returns its release Secret's UID, or "" when the cache has not seen it yet.
func newRevisionUID(ctx context.Context, s *engine.Session, ref model.ObjectRef, revision int) string {
	deadline := time.Now().Add(rebaseWait)
	for {
		if obj, err := s.Lookup(ref); err == nil && helm.ReleaseRevision(obj) == revision {
			return string(obj.GetUID())
		}
		if time.Now().After(deadline) || ctx.Err() != nil {
			return ""
		}
		time.Sleep(20 * time.Millisecond)
	}
}
