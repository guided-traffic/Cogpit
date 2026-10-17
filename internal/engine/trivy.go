package engine

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/model"
)

// The Trivy integration (D-78; the owner's I6, 2026-10-17: "an allen Pods das schlechteste
// Image-Rating aller Container, im Pod Drawer das Image Rating pro Image, wenn verfügbar"). Trivy
// Operator scans the images of a cluster's workloads and writes one VulnerabilityReport
// (aquasecurity.github.io/v1alpha1, namespaced) per workload and container, labelled with the
// workload's kind, name and namespace and the container's name (model.TrivyLabel*). Cogpit installs
// nothing: it reads the reports a running Trivy Operator wrote.
//
// While the integration is on for a connected cluster that serves the reports' kind (its custom
// resource definition, through the custom kinds watch), the integration's follower puts its overlay
// on the backend (trivySource, overlay.go) and takes it off when the integration is switched off.
// The overlay holds one handle on the reports, read as their namespace plan says (in all
// namespaces, or in each namespace the user named for a user who may not list them everywhere;
// nothing while the access reviews have not said which), in a trimmed scope (cache.Scope.Trim): the
// cache keeps each report without its vulnerability list (trimVulnerabilityReport), while the
// reports' own view, YAML and editor read the untrimmed scope and keep them whole. The index over
// them is built from the cache on the first lookup after a change and maps a workload and a
// container to the report's summary; a change of the reports tells the overlays, coalesced, which
// wakes the open view and drawer. The overlay gives the Pods list its column "Image Rating" (the
// worst rating of a pod's images) and a pod's drawer its "Image Ratings". Switched off: no overlay,
// so no column, no section and no read.
//
// A pod's reports are its controller's, as Trivy Operator resolves a pod's workload (its
// ReportOwner, read in its source and seen on kind 2026-10-17): a ReplicaSet's (a Deployment's
// pods), a StatefulSet's, a DaemonSet's, a ReplicationController's, a Job's, the CronJob's of a
// Job a CronJob made, and the pod's own for a bare pod or one another controller made. The
// CronJob of a Job is found by the Job's name, as the CronJob controller names its Jobs
// ("<cronjob>-<scheduled minute>"), after the Job's own reports: the lead's decision (2026-10-17),
// so that a pod costs no read of its Job (the Jobs are not watched for it); a Job made by hand from
// a CronJob under another name has no rating.

// trivyReportsKind is the kind key of Trivy Operator's vulnerability reports.
const trivyReportsKind = "vulnerabilityreports.aquasecurity.github.io"

// trivyNotServed is the Trivy integration's detail on a cluster without the reports' kind.
const trivyNotServed = "Trivy Operator's reports are not served"

// trivyTrim names the trim of the reports' scope (cache.RegisterTrim).
const trivyTrim = "trivy-summary"

// trivyColumnID is the Pods list's column of the worst image rating, its title, its width and the
// column it shows after.
const (
	trivyColumnID    = "imageRating"
	trivyColumnTitle = "Image Rating"
	trivyColumnWidth = 96
	trivyColumnAfter = specContainers // the Containers column: the images' rating beside their squares
)

// The workload kinds a report names that a pod's lookup picks itself, and a pod spec's container lists.
const (
	trivyKindPod       = "Pod"
	trivyKindJob       = "Job"
	trivyKindCronJob   = "CronJob"
	specContainers     = "containers"
	specInitContainers = "initContainers"
)

func init() {
	cache.RegisterTrim(trivyTrim, trimVulnerabilityReport)
	integrationProbes[IntegrationTrivy] = probeTrivy
	integrationFollowers = append(integrationFollowers, followTrivy)
}

// trimVulnerabilityReport keeps what the integration reads of a report: everything but its
// vulnerability list (report.vulnerabilities), which is 95 % of a report's weight (D-78, measured
// on kind). Metadata, the artifact, the registry, the summary and the scan's time stay.
func trimVulnerabilityReport(u *unstructured.Unstructured) {
	if report, ok := u.Object["report"].(map[string]any); ok {
		delete(report, "vulnerabilities")
	}
}

// followTrivy is the Trivy integration's follower: it puts the overlay on a connected cluster while
// the integration is on and the cluster serves the reports, puts a new one when the reports' kind
// or namespace plan changes, and takes it off when either ends. It runs while the engine reconciles
// (state.mu held): it does not wait.
func followTrivy(st *state, b *Backend, now UIState) {
	if b == nil || b.Cache == nil {
		return
	}
	cur, _ := overlayOf[*trivySource](b)
	p, served := b.Profile(trivyReportsKind)
	want := served && p.Namespaced && now.IntegrationOn(b.Context, IntegrationTrivy)
	key := ""
	if want {
		key = p.GVR().String() + "=" + b.namespacePlan(p).key()
	}
	switch {
	case cur == nil && !want, cur != nil && cur.key == key:
		return
	case want:
		b.overlays.put(IntegrationTrivy, startTrivy(st.s, b, p, key))
	default:
		b.overlays.put(IntegrationTrivy, nil)
	}
	ctx := b.Context
	st.s.goSafe("trivy integration", func() { st.s.integrationsChanged(ctx) })
}

// isPodsKind reports whether a profile is the core Pods'.
func isPodsKind(p *model.KindProfile) bool {
	return p != nil && p.Key == logKindPods && p.Group == ""
}

// trivyWorkload names the workload a report is of: its namespace, kind and name.
type trivyWorkload struct{ namespace, kind, name string }

// trivySource is the integration's read of the reports on one cluster and the index over them; it
// is the backend's Trivy overlay while the integration is on.
type trivySource struct {
	s       *Session
	b       *Backend
	p       *model.KindProfile
	key     string // what it reads (followTrivy): another key puts a new overlay
	plan    nsPlan
	h       *cache.Handle
	unsub   func()
	wake    chan struct{}
	quit    chan struct{}
	stopped atomic.Bool
	gen     atomic.Uint64 // bumped by every event of the reports

	mu    sync.Mutex
	built uint64 // gen that idx was built at
	idx   trivyReportIndex
	count int
}

// trivyReportIndex maps a workload to its reports by container name. Built once, never changed.
type trivyReportIndex map[trivyWorkload]map[string]model.VulnerabilityReport

// newTrivyReportIndex indexes report objects; it returns the index and how many reports it holds.
func newTrivyReportIndex(objs []*unstructured.Unstructured) (trivyReportIndex, int) {
	idx := trivyReportIndex{}
	count := 0
	for _, obj := range objs {
		r, ok := model.ParseVulnerabilityReport(obj)
		if !ok {
			continue
		}
		count++
		w := trivyWorkload{namespace: r.Namespace, kind: r.Kind, name: r.Workload}
		byContainer := idx[w]
		if byContainer == nil {
			byContainer = map[string]model.VulnerabilityReport{}
			idx[w] = byContainer
		}
		if cur, ok := byContainer[r.Container]; !ok || r.Scanned.After(cur.Scanned) {
			byContainer[r.Container] = r // two reports of one container: the newer scan
		}
	}
	return idx, count
}

// startTrivy acquires the reports' trimmed scope as their plan says and starts the loop that tells
// the overlays of changes.
func startTrivy(s *Session, b *Backend, p *model.KindProfile, key string) *trivySource {
	t := &trivySource{s: s, b: b, p: p, key: key, plan: b.namespacePlan(p),
		wake: make(chan struct{}, 1), quit: make(chan struct{})}
	sc := b.Scope(p)
	sc.Trim = trivyTrim
	switch {
	case t.plan.pending: // the access reviews have not said whether the user may list them everywhere
		t.h = cache.Pending(sc)
	case t.plan.perNamespace:
		t.h = b.Cache.AcquireNamespaces(sc, t.plan.watch)
	default:
		t.h = b.Cache.Acquire(sc)
	}
	t.unsub = t.h.Subscribe(func(cache.Event) {
		t.gen.Add(1)
		signal(t.wake)
	})
	s.goSafe("trivy reports "+b.Context, t.run)
	return t
}

// run tells the overlays, at most once per snapshot interval, that the reports changed (the open
// view's ratings and the drawer's sections follow), the settings page when what the integration
// knows changed (the reports listed, a refusal), and the session when a plan that was pending is
// decided (the follower then reads as it says), until the overlay stops.
func (t *trivySource) run() {
	settled, replanned := t.settled(), false
	poll := func() bool {
		return t.settled() != settled || (!replanned && t.planSettled())
	}
	runCoalesced(t.quit, t.wake, t.s.opts.SnapshotInterval, poll, func() {
		if t.stopped.Load() {
			return
		}
		t.b.overlays.changed()
		if now := t.settled(); now != settled {
			settled = now
			t.s.integrationsChanged(t.b.Context)
		}
		if !replanned && t.planSettled() {
			replanned = true
			t.b.ns.wakeReplan()
		}
	})
}

// settled is what the integration's state reads of the handle: synced, and its error.
func (t *trivySource) settled() string {
	return fmt.Sprint(t.h.HasSynced(), errText(t.h.Err()))
}

// planSettled reports whether the read started while its plan was pending and the plan is decided now.
func (t *trivySource) planSettled() bool {
	return t.plan.pending && !t.b.namespacePlan(t.p).pending
}

func (t *trivySource) stop() {
	if !t.stopped.CompareAndSwap(false, true) {
		return
	}
	close(t.quit)
	t.unsub()
	t.h.Release()
}

// synced reports whether the reports' initial list is done.
func (t *trivySource) synced() bool { return t != nil && t.h.HasSynced() }

// waitSynced waits, bounded by ctx, until the reports listed or were refused: an agent's read sees
// what a connect just started to read.
func (t *trivySource) waitSynced(ctx context.Context) {
	if t.plan.pending {
		return
	}
	for !t.h.HasSynced() && !cache.IsRefused(t.h.Err()) && ctx.Err() == nil && !t.stopped.Load() {
		select {
		case <-ctx.Done():
		case <-time.After(10 * time.Millisecond):
		}
	}
}

// index returns the index, built from the cache first when the reports changed since.
func (t *trivySource) index() (trivyReportIndex, int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	gen := t.gen.Load() // before the list: an event during the build builds it again next time
	if t.idx == nil || t.built != gen {
		t.idx, t.count = newTrivyReportIndex(t.h.List())
		t.built = gen
	}
	return t.idx, t.count
}

// reports returns how many reports the index holds.
func (t *trivySource) reports() int {
	_, n := t.index()
	return n
}

// columns: the Pods list's "Image Rating", shown after Containers, while the overlay is on (also
// while no pod has a rating yet: a column that comes and goes with the reports would move the
// columns after it). Other kinds get none.
func (t *trivySource) columns(p *model.KindProfile) []overlayColumn {
	if !isPodsKind(p) {
		return nil
	}
	return []overlayColumn{{
		Column: model.Column{ID: trivyColumnID, Title: trivyColumnTitle, Type: model.ColumnRank, Width: trivyColumnWidth},
		After:  trivyColumnAfter,
	}}
}

// cell is a pod's worst image rating ("Critical · 39"), sorted by the rating, then the count; none
// while the reports list or when none of its images has a report.
func (t *trivySource) cell(p *model.KindProfile, column string, obj *unstructured.Unstructured) (model.Cell, bool) {
	if column != trivyColumnID || !isPodsKind(p) || !t.synced() {
		return model.Cell{}, false
	}
	idx, _ := t.index()
	worst, ok := worstImage(idx.podImages(obj))
	if !ok {
		return model.Cell{}, false
	}
	r, c := worst.rating()
	return model.Cell{Text: model.RatingText(r, c), Status: r.Status(), Num: model.RatingRank(r, c)}, true
}

// worstImage returns the image of the worst rating among those with a report (by rating, then the
// count of that severity); ok is false when none has one.
func worstImage(images []podImage) (podImage, bool) {
	var worst podImage
	found := false
	for _, im := range images {
		if im.report == nil {
			continue
		}
		r, c := im.rating()
		if !found || model.RatingRank(r, c) > model.RatingRank(worst.rating()) {
			worst, found = im, true
		}
	}
	return worst, found
}

// sections: a pod's "Image Ratings" (imageRatingSections).
func (t *trivySource) sections(p *model.KindProfile, obj *unstructured.Unstructured, f model.TimeFormat) []model.DetailSection {
	if !isPodsKind(p) || obj == nil || !t.synced() {
		return nil
	}
	idx, _ := t.index()
	return f.WriteSections(imageRatingSections(idx.podImages(obj), t.s.displayNow()))
}

// trivyWorkloadsOf returns the workloads whose reports hold a pod's containers, in the order to
// try them (the file's comment).
func trivyWorkloadsOf(pod *unstructured.Unstructured) []trivyWorkload {
	ns := pod.GetNamespace()
	own := []trivyWorkload{{ns, trivyKindPod, pod.GetName()}}
	for _, o := range pod.GetOwnerReferences() {
		if o.Controller == nil || !*o.Controller {
			continue
		}
		group, _, grouped := strings.Cut(o.APIVersion, "/")
		if !grouped {
			group = "" // "v1": the core group
		}
		switch {
		case group == "apps" && (o.Kind == apiKindReplicaSet || o.Kind == apiKindStatefulSet || o.Kind == apiKindDaemonSet),
			group == "" && o.Kind == "ReplicationController":
			return []trivyWorkload{{ns, o.Kind, o.Name}}
		case group == "batch" && o.Kind == trivyKindJob:
			out := []trivyWorkload{{ns, trivyKindJob, o.Name}}
			if cj := cronJobOfJob(o.Name); cj != "" {
				out = append(out, trivyWorkload{ns, trivyKindCronJob, cj})
			}
			return out
		}
		return own // another controller: Trivy Operator scans the pod itself
	}
	return own
}

// cronJobOfJob returns the CronJob a Job's name says made it, "<cronjob>-<scheduled minute>" as
// the CronJob controller names its Jobs; "" for another name.
func cronJobOfJob(job string) string {
	i := strings.LastIndexByte(job, '-')
	if i <= 0 || i == len(job)-1 {
		return ""
	}
	for _, c := range job[i+1:] {
		if c < '0' || c > '9' {
			return ""
		}
	}
	return job[:i]
}

// podImage is one image of a pod: the containers that run it (an init container marked so) and its
// report, nil when it has none (not scanned, or a report of another image not replaced yet).
type podImage struct {
	image      string
	containers []string
	report     *model.VulnerabilityReport
}

// rating is the image's rating and its counts.
func (im podImage) rating() (model.ImageRating, model.VulnerabilityCounts) {
	if im.report == nil {
		return model.RatingNotScanned, model.VulnerabilityCounts{}
	}
	return im.report.Counts.Rating(), im.report.Counts
}

// podImages returns a pod's images in the order its init containers and then its containers name
// them, each with its report. Ephemeral containers are left out: Trivy Operator scans workloads'
// pod templates, which have none.
func (idx trivyReportIndex) podImages(pod *unstructured.Unstructured) []podImage {
	var reports map[string]model.VulnerabilityReport
	for _, w := range trivyWorkloadsOf(pod) {
		if reports = idx[w]; len(reports) > 0 {
			break
		}
	}
	var out []podImage
	at := map[string]int{}
	add := func(c map[string]any, init bool) {
		name, image := model.NestedString(c, "name"), model.NestedString(c, "image")
		if name == "" || image == "" {
			return
		}
		i, ok := at[image]
		if !ok {
			i = len(out)
			at[image] = i
			out = append(out, podImage{image: image})
		}
		label := name
		if init {
			label += " (init)"
		}
		out[i].containers = append(out[i].containers, label)
		r, ok := reports[name]
		if !ok || !r.Matches(image) {
			return
		}
		if cur := out[i].report; cur == nil || worseReport(r, *cur) {
			out[i].report = &r // two containers of one image: the worse report, then the newer
		}
	}
	for _, field := range []string{specInitContainers, specContainers} {
		for _, c := range model.NestedSlice(pod.Object, "spec", field) {
			if m, ok := c.(map[string]any); ok {
				add(m, field == specInitContainers)
			}
		}
	}
	return out
}

func worseReport(a, b model.VulnerabilityReport) bool {
	ra, rb := model.RatingRank(a.Counts.Rating(), a.Counts), model.RatingRank(b.Counts.Rating(), b.Counts)
	if ra != rb {
		return ra > rb
	}
	return a.Scanned.After(b.Scanned)
}

// probeTrivy is the Trivy integration's probe: whether the cluster serves the reports' kind (the
// custom kinds), and what the overlay's read of them met. It sends nothing.
func probeTrivy(_ *Session, b *Backend, on bool) (IntegrationState, string, string) {
	if !on {
		return IntegrationOff, "", ""
	}
	p, served := b.Profile(trivyReportsKind)
	if !served {
		set := b.customSet()
		switch {
		case set == nil:
			return IntegrationMissing, "", trivyNotServed
		case set.err != "":
			// Discovery names only the curated kinds (A90): without the definitions Cogpit cannot tell.
			state := IntegrationUnavailable
			if strings.Contains(strings.ToLower(set.err), "forbidden") {
				state = IntegrationForbidden
			}
			return state, "", "Cogpit cannot tell whether Trivy Operator's reports are served: the custom resource definitions cannot be listed (" + set.err + ")"
		case !set.synced:
			return IntegrationChecking, "", ""
		}
		return IntegrationMissing, "", trivyNotServed
	}
	where := p.Group + "/" + p.Version
	t, _ := overlayOf[*trivySource](b)
	if t == nil {
		return IntegrationChecking, where, ""
	}
	err := t.h.Err()
	switch {
	case refusedAll(t.h):
		return IntegrationForbidden, where, "list " + trivyReportsKind
	case t.h.HasSynced():
		n := t.reports()
		if n == 1 {
			return IntegrationFound, where, "1 vulnerability report"
		}
		return IntegrationFound, where, fmt.Sprintf("%d vulnerability reports", n)
	case err != nil:
		return IntegrationUnavailable, where, errText(err)
	case t.plan.pending:
		return IntegrationChecking, where, "waiting for the access reviews"
	}
	return IntegrationChecking, where, ""
}

// The drawer's "Image Ratings" (D-78): one section per image of the pod, under the group heading,
// titled by the image as the pod names it: the rating with its count ("Critical · 39"), the counts
// as chips, the containers that run it, when it was scanned, and the report as a link to its object
// (the reports' custom kind view). An image without a report says "Not scanned". A pod none of whose
// images has a report gets no section (the owner's "wenn verfügbar": a namespace Trivy Operator
// does not scan, a scan not done yet), and neither does a pod while the integration is off, the
// cluster does not serve the reports, or they are still being listed. The lead's decisions
// (2026-10-17). The sections come through the overlay (trivySource.sections), after the kind's own.

// groupImageRatings is the drawer's group heading of the image sections.
const groupImageRatings = "Image Ratings"

// PodImageRatings returns a pod's "Image Ratings" sections from Trivy Operator's report objects, as
// the overlay builds them from its index, with absolute times in ISO 8601 in now's zone (as
// BuildDetail writes them): the drawer's content without the cache, for tests of the UI.
func PodImageRatings(pod *unstructured.Unstructured, reports []*unstructured.Unstructured, now time.Time) []model.DetailSection {
	idx, _ := newTrivyReportIndex(reports)
	return model.TimeLayout{}.In(now.Location()).WriteSections(imageRatingSections(idx.podImages(pod), now))
}

// imageRatingSections are the drawer sections of a pod's images; none when no image has a report.
// Absolute times are marks (model.MarkTime).
func imageRatingSections(images []podImage, now time.Time) []model.DetailSection {
	scanned := false
	for _, im := range images {
		scanned = scanned || im.report != nil
	}
	if !scanned {
		return nil
	}
	out := make([]model.DetailSection, 0, len(images))
	for _, im := range images {
		rating, counts := im.rating()
		sec := model.DetailSection{Title: im.image, Group: groupImageRatings, Items: []model.DetailItem{
			{Name: "Rating", Value: model.RatingText(rating, counts), Status: rating.Status()},
		}}
		if chips := counts.Chips(); len(chips) > 0 {
			sec.Items = append(sec.Items, model.ChipsItem("Vulnerabilities", chips))
		}
		name := "Container"
		if len(im.containers) > 1 {
			name = "Containers"
		}
		sec.Items = append(sec.Items, model.DetailItem{Name: name, Value: strings.Join(im.containers, ", ")})
		if r := im.report; r != nil {
			if !r.Scanned.IsZero() {
				sec.Items = append(sec.Items, model.DetailItem{
					Name: "Scanned", Value: model.FormatAge(now.Sub(r.Scanned)) + " ago (" + model.MarkTime(r.Scanned) + ")",
				})
			}
			sec.Items = append(sec.Items, model.DetailItem{
				Name: "Report", Value: r.Name,
				Link: &model.ObjectRef{Kind: trivyReportsKind, Namespace: r.Namespace, Name: r.Name},
			})
		}
		out = append(out, sec)
	}
	return out
}
