package helm

import (
	"cmp"
	"errors"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"helm.sh/helm/v4/pkg/release/common"
	rspb "helm.sh/helm/v4/pkg/release/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"

	"github.com/guided-traffic/cogpit/internal/model"
)

// ReleaseKind is the kind key of Helm releases, a virtual kind (architecture §4.5, §5): Cogpit's
// own name, a bare plural without a dot, so that it never collides with a custom resource kind
// such as Flux's helmreleases.helm.toolkit.fluxcd.io.
const ReleaseKind = "helmreleases"

// The apiVersion and kind of a derived release object: Cogpit's own, not an API the cluster serves.
const (
	releaseAPIVersion = model.LocalGroup + "/v1"
	releaseObjectKind = "HelmRelease"
)

// Titles of the release drawer's sections.
const (
	TitleHistory  = "History"
	TitleValues   = "Values"
	TitleManifest = "Manifest"
)

// Bounds on what the drawer keeps of a release. A release's history is at most what Helm keeps
// (10 revisions by default); the values and the manifest are cut for the drawer, which renders one
// line per line.
const (
	historyRows   = 100      // revisions in the History table
	valuesBytes   = 16 << 10 // of the user-supplied values as YAML
	valuesLines   = 200
	manifestBytes = 32 << 10 // of the rendered manifest
	manifestLines = 400
)

// The memo's bounds: entries for Secrets that are gone leave it after two generations of this
// many Derive calls without being seen; it never holds more than memoEntries entries.
const (
	memoGeneration = 64
	memoEntries    = 4096
)

// NewReleaseProfile returns the profile of the Helm releases of one cluster connection: a virtual
// kind over the Secrets labelled owner=helm (D-13), one object per release, its newest revision
// winning. Each call returns a profile with its own memo of decoded payloads; the connector makes
// one per backend.
func NewReleaseProfile() *model.KindProfile {
	m := &memo{cur: map[types.UID]*revision{}, prev: map[types.UID]*revision{}}
	return &model.KindProfile{
		Key:           ReleaseKind,
		Version:       "v1",
		Resource:      "secrets",
		Kind:          releaseObjectKind,
		Title:         "Releases",
		Namespaced:    true,
		LabelSelector: ReleaseSelector,
		Columns: []model.Column{
			{ID: "name", Title: "Name", Type: model.ColumnText},
			{ID: "namespace", Title: "Namespace", Type: model.ColumnText},
			{ID: "chart", Title: "Chart", Type: model.ColumnText},
			{ID: "revision", Title: "Revision", Type: model.ColumnNumber},
			{ID: "version", Title: "Version", Type: model.ColumnText},
			{ID: "appVersion", Title: "App Version", Type: model.ColumnText},
			{ID: "status", Title: "Status", Type: model.ColumnText},
			{ID: "updated", Title: "Updated", Type: model.ColumnAge},
		},
		Cells: releaseCells,
		Fields: map[string]func(*unstructured.Unstructured) []string{
			"chart":  func(o *unstructured.Unstructured) []string { return nonEmpty(model.NestedString(o, "chart")) },
			"status": func(o *unstructured.Unstructured) []string { return nonEmpty(model.NestedString(o, "status")) },
		},
		Derive: m.derive,
		Detail: m.detail,
	}
}

func nonEmpty(s string) []string {
	if s == "" {
		return nil
	}
	return []string{s}
}

// StatusToken colours a release status (inferred: Lens's colours are not captured): deployed is
// good, failed an error, the pending states pending, uninstalling like a deletion; superseded and
// uninstalled (a release uninstalled with its history kept) have no colour.
func StatusToken(status string) model.Status {
	switch common.Status(status) {
	case common.StatusDeployed:
		return model.StatusOK
	case common.StatusFailed:
		return model.StatusError
	case common.StatusPendingInstall, common.StatusPendingUpgrade, common.StatusPendingRollback:
		return model.StatusPending
	case common.StatusUninstalling:
		return model.StatusTerminating
	case common.StatusSuperseded, common.StatusUninstalled:
		return model.StatusNone
	}
	return model.StatusUnknown
}

// Pending reports whether a status is one a crash can leave behind: an install, upgrade or
// rollback that did not finish, or an uninstall.
func Pending(status string) bool {
	return common.Status(status).IsPending() || common.Status(status) == common.StatusUninstalling
}

// ReleaseRevision returns a release object's newest revision.
func ReleaseRevision(o *unstructured.Unstructured) int {
	n, _ := model.NestedInt64(o, "revision")
	return int(n)
}

// ReleaseStatus returns the status of a release object's newest revision.
func ReleaseStatus(o *unstructured.Unstructured) string { return model.NestedString(o, "status") }

// ReleaseRevisions returns the revisions in a release object's history, newest first.
func ReleaseRevisions(o *unstructured.Unstructured) []int {
	var out []int
	for _, h := range model.NestedSlice(o, "history") {
		if n, ok := model.NestedInt64(h, "revision"); ok {
			out = append(out, int(n))
		}
	}
	return out
}

func releaseCells(o *unstructured.Unstructured) ([]model.Cell, model.Status) {
	status := model.NestedString(o, "status")
	token := StatusToken(status)
	rev, _ := model.NestedInt64(o, "revision")
	ns := o.GetNamespace()
	return []model.Cell{
		{Text: o.GetName()},
		{Text: ns, Link: &model.ObjectRef{Kind: "namespaces", Name: ns}},
		{Text: model.NestedString(o, "chart")},
		{Text: strconv.FormatInt(rev, 10), Num: rev},
		{Text: model.NestedString(o, "chartVersion")},
		{Text: model.NestedString(o, "appVersion")},
		{Text: status, Status: token},
		model.AgeCell(model.NestedTime(o, "updated")),
	}, token
}

// revision is what the memo keeps of one release Secret: one revision of one release.
type revision struct {
	uid       types.UID
	rv        string
	secret    string // the Secret's name
	namespace string
	created   time.Time // the Secret's creation

	name, status                    string
	number                          int
	chart, chartVersion, appVersion string
	updated, firstDeployed          time.Time
	description                     string
	err                             string // why the payload could not be read; the rest is from the labels

	// heavy are the values and the manifest, kept for the newest revision of each release only.
	heavy *heavy
}

// heavy is what the drawer shows of a release's newest revision beyond its summary.
type heavy struct {
	values   string // the user-supplied values as YAML, cut to valuesBytes and valuesLines
	manifest string // the rendered manifest, Secret documents redacted, cut to manifestBytes and manifestLines
}

// memo keeps the revisions Derive decoded, by the Secret's UID, valid for one resourceVersion, so
// that a snapshot decodes only the Secrets that changed. Entries move from prev to cur when they
// are used; every memoGeneration calls (or when cur grows past memoEntries) cur becomes prev and
// the old prev, entries no Derive asked for since, is dropped: that is how entries for Secrets
// that are gone leave it, whichever namespaces a call asked for.
type memo struct {
	mu    sync.Mutex
	cur   map[types.UID]*revision
	prev  map[types.UID]*revision
	calls int
}

// get returns the memo's revision for a Secret, or nil. Called with m.mu held.
func (m *memo) get(uid types.UID, rv string) *revision {
	if r, ok := m.cur[uid]; ok && r.rv == rv {
		return r
	}
	if r, ok := m.prev[uid]; ok && r.rv == rv {
		delete(m.prev, uid)
		m.put(r)
		return r
	}
	return nil
}

// put stores a revision. Called with m.mu held.
func (m *memo) put(r *revision) {
	if len(m.cur) >= memoEntries {
		m.rotate()
	}
	m.cur[r.uid] = r
}

func (m *memo) rotate() {
	m.prev, m.cur = m.cur, map[types.UID]*revision{}
}

// size returns how many revisions the memo holds, for tests.
func (m *memo) size() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.cur) + len(m.prev)
}

// derive is the profile's Derive: one object per release in the given release Secrets (one or all
// namespaces), from its newest revision, with the history of the revisions present. It decodes
// only Secrets the memo does not hold in their current version, outside the memo's lock.
func (m *memo) derive(objs []*unstructured.Unstructured) []*unstructured.Unstructured {
	byUID := make(map[types.UID]*unstructured.Unstructured, len(objs))
	revs := make([]*revision, 0, len(objs))
	var misses []*unstructured.Unstructured
	m.mu.Lock()
	for _, o := range objs {
		if o == nil || o.GetUID() == "" {
			continue
		}
		byUID[o.GetUID()] = o
		if r := m.get(o.GetUID(), o.GetResourceVersion()); r != nil {
			revs = append(revs, r)
		} else {
			misses = append(misses, o)
		}
	}
	m.mu.Unlock()

	decoded := make([]*revision, 0, len(misses))
	for _, o := range misses {
		decoded = append(decoded, decodeRevision(o))
	}

	m.mu.Lock()
	for _, r := range decoded {
		m.put(r)
	}
	revs = append(revs, decoded...)
	groups := groupReleases(revs)
	var refill []*revision
	for _, g := range groups {
		for _, r := range g[1:] {
			r.heavy = nil // only the newest revision's values and manifest are kept
		}
		if newest := g[0]; newest.heavy == nil && newest.err == "" {
			refill = append(refill, newest)
		}
	}
	m.mu.Unlock()

	// A revision that was older and became the newest (a newer one was deleted) is decoded again
	// for its values and manifest.
	for _, r := range refill {
		if o := byUID[r.uid]; o != nil {
			h := decodeRevision(o).heavy
			m.mu.Lock()
			if r.heavy == nil {
				r.heavy = h
			}
			m.mu.Unlock()
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*unstructured.Unstructured, 0, len(groups))
	for _, g := range groups {
		out = append(out, releaseObject(g))
	}
	if m.calls++; m.calls%memoGeneration == 0 {
		m.rotate()
	}
	return out
}

// groupReleases groups revisions by namespace and release name, newest revision first in each
// group, the groups in order of namespace and name. Called with m.mu held: it reads revisions.
func groupReleases(revs []*revision) [][]*revision {
	idx := map[string]int{}
	var groups [][]*revision
	for _, r := range revs {
		key := r.namespace + "/" + r.name
		i, ok := idx[key]
		if !ok {
			i = len(groups)
			idx[key] = i
			groups = append(groups, nil)
		}
		groups[i] = append(groups[i], r)
	}
	for _, g := range groups {
		slices.SortFunc(g, func(a, b *revision) int {
			if c := cmp.Compare(b.number, a.number); c != 0 {
				return c
			}
			return cmp.Compare(b.secret, a.secret) // two Secrets claiming one revision: a stable choice
		})
	}
	slices.SortFunc(groups, func(a, b []*revision) int {
		return cmp.Or(cmp.Compare(a[0].namespace, b[0].namespace), cmp.Compare(a[0].name, b[0].name))
	})
	return groups
}

// releaseObject builds a release's object from its revisions, newest first: the newest revision's
// summary, its Secret's UID and resourceVersion (a confirmation binds to the revision the user
// saw), and the history. It holds no values and no manifest: the YAML tab shows this object, and
// the drawer's Values and Manifest come from the memo (detail). Called with m.mu held.
func releaseObject(g []*revision) *unstructured.Unstructured {
	r := g[0]
	created := r.firstDeployed
	if created.IsZero() {
		created = g[len(g)-1].created
	}
	meta := map[string]any{
		"name": r.name, "namespace": r.namespace, "uid": string(r.uid), "resourceVersion": r.rv,
	}
	if !created.IsZero() {
		meta["creationTimestamp"] = created.UTC().Format(time.RFC3339)
	}
	history := make([]any, 0, min(len(g), historyRows))
	for _, h := range g[:min(len(g), historyRows)] {
		history = append(history, map[string]any{
			"revision": int64(h.number), "updated": timeText(h.updated), "status": h.status,
			"chart": chartText(h.chart, h.chartVersion), "appVersion": h.appVersion, "description": h.description,
		})
	}
	obj := map[string]any{
		"apiVersion": releaseAPIVersion, "kind": releaseObjectKind, "metadata": meta,
		"chart": r.chart, "chartVersion": r.chartVersion, "appVersion": r.appVersion,
		"revision": int64(r.number), "status": r.status, "updated": timeText(r.updated),
		"description": r.description, "secret": r.secret, "history": history,
	}
	if r.err != "" {
		obj["error"] = r.err
	}
	return &unstructured.Unstructured{Object: obj}
}

func timeText(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// chartText is a chart as helm list shows it: "web-1.2.0".
func chartText(name, version string) string {
	if version == "" {
		return name
	}
	return name + "-" + version
}

// decodeRevision reads one release Secret. The payload is untrusted input: a Secret that does not
// decode yields a revision from its labels and name, with the error, never a panic.
func decodeRevision(o *unstructured.Unstructured) *revision {
	labels := o.GetLabels()
	r := &revision{
		uid: o.GetUID(), rv: o.GetResourceVersion(), secret: o.GetName(), namespace: o.GetNamespace(),
		created: o.GetCreationTimestamp().Time,
		name:    labels["name"], status: labels["status"],
	}
	r.number, _ = strconv.Atoi(labels["version"])
	if r.name == "" || r.number == 0 {
		name, number := parseSecretName(o.GetName())
		r.name = cmp.Or(r.name, name)
		r.number = cmp.Or(r.number, number)
	}
	rls, err := decodeSafely(o)
	if err != nil {
		r.err = err.Error()
		r.name = cmp.Or(r.name, o.GetName()) // a release needs a name: the Secret's, when nothing else says one
		return r
	}
	r.name = cmp.Or(rls.Name, r.name)
	r.number = cmp.Or(rls.Version, r.number)
	if rls.Info != nil {
		r.status = cmp.Or(string(rls.Info.Status), r.status)
		r.updated, r.firstDeployed, r.description = rls.Info.LastDeployed, rls.Info.FirstDeployed, rls.Info.Description
	}
	if rls.Chart != nil && rls.Chart.Metadata != nil {
		r.chart, r.chartVersion, r.appVersion = rls.Chart.Metadata.Name, rls.Chart.Metadata.Version, rls.Chart.Metadata.AppVersion
	}
	r.heavy = &heavy{manifest: cut(redactManifest(rls.Manifest), manifestBytes, manifestLines)}
	if len(rls.Config) > 0 {
		if y, err := yaml.Marshal(rls.Config); err == nil {
			r.heavy.values = cut(strings.TrimSuffix(string(y), "\n"), valuesBytes, valuesLines)
		}
	}
	return r
}

// errUnreadable stands for a decoder that panicked on a payload.
var errUnreadable = errors.New("the release payload could not be read")

// decodeSafely decodes a release Secret, turning a panic in a decoder into an error.
func decodeSafely(o *unstructured.Unstructured) (rls *rspb.Release, err error) {
	defer func() {
		if p := recover(); p != nil {
			rls, err = nil, errUnreadable
		}
	}()
	return DecodeSecret(o)
}

// parseSecretName reads the release name and revision from a Secret named the way Helm's Secrets
// driver names them, sh.helm.release.v1.<name>.v<revision>.
func parseSecretName(s string) (string, int) {
	rest, ok := strings.CutPrefix(s, "sh.helm.release.v1.")
	if !ok {
		return "", 0
	}
	i := strings.LastIndex(rest, ".v")
	if i <= 0 {
		return rest, 0
	}
	n, err := strconv.Atoi(rest[i+2:])
	if err != nil {
		return rest, 0
	}
	return rest[:i], n
}

// cut returns text within a byte and a line limit, cut at a line boundary, with a note saying how
// much was left out.
func cut(text string, maxBytes, maxLines int) string {
	total := strings.Count(text, "\n") + 1
	shown := text
	if len(shown) > maxBytes {
		end := maxBytes
		for end > 0 && !utf8.RuneStart(shown[end]) {
			end--
		}
		shown = shown[:end]
		if i := strings.LastIndexByte(shown, '\n'); i > 0 {
			shown = shown[:i]
		}
	}
	if lines := strings.SplitAfterN(shown, "\n", maxLines+1); len(lines) > maxLines {
		shown = strings.TrimSuffix(strings.Join(lines[:maxLines], ""), "\n")
	}
	if len(shown) == len(text) {
		return text
	}
	return shown + "\n# [Cogpit shows the first " + strconv.Itoa(strings.Count(shown, "\n")+1) + " of " +
		strconv.Itoa(total) + " lines here]"
}
