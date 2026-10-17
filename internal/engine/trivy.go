package engine

import (
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
// resource definition, through the custom kinds watch), the cluster's backend holds one handle on
// the reports (trivyReads, started and stopped by the follower below, as the UI state changes), read
// as an open page reads its kinds: in all namespaces, or in each namespace the user named for a
// user who may not list them everywhere, and nothing while the access reviews have not said which
// (acquireForPage, accessiblens_pages.go). The index over them is built from the cache on the first
// lookup after a change and maps a workload and a container to the report's summary. Switched off,
// the handle goes back: no request, no drawer section, nothing in ui_state.
//
// A pod's reports are its controller's, as Trivy Operator resolves a pod's workload (its
// ReportOwner, read in its source and seen on kind 2026-10-17): a ReplicaSet's (a Deployment's
// pods), a StatefulSet's, a DaemonSet's, a ReplicationController's, a Job's, the CronJob's of a
// Job a CronJob made, and the pod's own for a bare pod or one another controller made. The
// CronJob of a Job is found by the Job's name, as the CronJob controller names its Jobs
// ("<cronjob>-<scheduled minute>"), after the Job's own reports: the lead's decision (2026-10-17),
// so that a pod costs no read of its Job (the Jobs are not watched for it); a Job made by hand from
// a CronJob under another name has no rating.
//
// What Phase 2 adds (D-78): the Pods list's column, the worst rating of a pod's images, and an
// informer transform that keeps only each report's summary in the cache (the vulnerability lists
// are the reports' weight: 186 KB of JSON for nginx:1.16.1's 363, measured on kind).

// trivyReportsKind is the kind key of Trivy Operator's vulnerability reports.
const trivyReportsKind = "vulnerabilityreports.aquasecurity.github.io"

// trivyNotServed is the Trivy integration's detail on a cluster without the reports' kind.
const trivyNotServed = "Trivy Operator's reports are not served"

// The workload kinds a report names that a pod's lookup picks itself, and a pod spec's container lists.
const (
	trivyKindPod       = "Pod"
	trivyKindJob       = "Job"
	trivyKindCronJob   = "CronJob"
	specContainers     = "containers"
	specInitContainers = "initContainers"
)

func init() {
	integrationProbes[IntegrationTrivy] = probeTrivy
	integrationFollowers = append(integrationFollowers, followTrivy)
}

// trivyReads is a backend's read of Trivy Operator's reports: the index while the integration is
// on, nil otherwise. The backend's close stops it.
type trivyReads struct {
	mu     sync.Mutex
	ix     *trivyIndex
	closed bool
}

// current returns the running index, or nil.
func (t *trivyReads) current() *trivyIndex {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.ix
}

// stop ends the index for good: the backend closes.
func (t *trivyReads) stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.closed = true
	if t.ix != nil {
		t.ix.stop()
		t.ix = nil
	}
}

// follow starts the index (want, read as key says), starts it again when key changed, or stops it;
// it reports whether it did any of that.
func (t *trivyReads) follow(s *Session, b *Backend, p *model.KindProfile, want bool, key string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed || (t.ix == nil && !want) || (t.ix != nil && want && t.ix.key == key) {
		return false
	}
	if t.ix != nil {
		t.ix.stop()
		t.ix = nil
	}
	if want {
		t.ix = startTrivyIndex(s, b, p, key)
	}
	return true
}

// followTrivy is the Trivy integration's follower: it reads the reports of a connected cluster
// while the integration is on and the cluster serves them, as their namespace plan says, and stops
// when either ends. It runs while the engine reconciles (state.mu held): it does not wait.
func followTrivy(st *state, b *Backend, now UIState) {
	if b == nil || b.Cache == nil {
		return
	}
	p, served := b.Profile(trivyReportsKind)
	want := served && p.Namespaced && now.IntegrationOn(b.Context, IntegrationTrivy)
	key := ""
	if want {
		key = p.GVR().String() + "|" + b.pageReadKey([]string{trivyReportsKind}, false)
	}
	if !b.trivy.follow(st.s, b, p, want, key) {
		return
	}
	if d := st.drawer; d != nil && d.backend == b && isPodsKind(d.profile) {
		signal(d.wake) // its image ratings come or go
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

// trivyIndex is one read of the reports and the index over them.
type trivyIndex struct {
	s       *Session
	b       *Backend
	key     string // how it reads them (followTrivy): another key starts it again
	h       *cache.Handle
	pending bool // the reports' namespace plan was pending: h holds nothing until it is decided
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

func startTrivyIndex(s *Session, b *Backend, p *model.KindProfile, key string) *trivyIndex {
	ix := &trivyIndex{
		s: s, b: b, key: key, h: b.acquireForPage(p), pending: b.namespacePlan(p).pending,
		wake: make(chan struct{}, 1), quit: make(chan struct{}),
	}
	ix.unsub = ix.h.Subscribe(func(cache.Event) {
		ix.gen.Add(1)
		signal(ix.wake)
	})
	settled := ix.settled()
	poll := func() bool { return ix.settled() != settled }
	s.goSafe("trivy reports", func() {
		runCoalesced(ix.quit, ix.wake, s.opts.SnapshotInterval, poll, func() {
			if ix.stopped.Load() {
				return
			}
			s.wakePodDrawer(b) // the open pod's ratings may have changed
			if now := ix.settled(); now != settled {
				settled = now
				s.integrationsChanged(b.Context) // the reports were listed, or refused
			}
		})
	})
	return ix
}

// settled is what the integration's state reads of the handle: synced, and its error.
func (ix *trivyIndex) settled() string {
	return fmt.Sprint(ix.h.HasSynced(), errText(ix.h.Err()))
}

func (ix *trivyIndex) stop() {
	if !ix.stopped.CompareAndSwap(false, true) {
		return
	}
	close(ix.quit)
	ix.unsub()
	ix.h.Release()
}

// synced reports whether the reports' initial list is done.
func (ix *trivyIndex) synced() bool { return ix != nil && ix.h.HasSynced() }

// index returns the index, built from the cache first when the reports changed since.
func (ix *trivyIndex) index() (trivyReportIndex, int) {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	gen := ix.gen.Load() // before the list: an event during the build builds it again next time
	if ix.idx == nil || ix.built != gen {
		ix.idx, ix.count = newTrivyReportIndex(ix.h.List())
		ix.built = gen
	}
	return ix.idx, ix.count
}

// reports returns how many reports the index holds.
func (ix *trivyIndex) reports() int {
	_, n := ix.index()
	return n
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
	ra, rb := a.Counts.Rating(), b.Counts.Rating()
	if ra != rb {
		return ra > rb
	}
	return a.Scanned.After(b.Scanned)
}

// trivyIndexOf returns a context's index when the integration is on and its reports are listed,
// else nil: then nothing of it shows.
func (s *Session) trivyIndexOf(b *Backend) *trivyIndex {
	if b == nil {
		return nil
	}
	if u := s.UIState(); !u.IntegrationOn(b.Context, IntegrationTrivy) {
		return nil
	}
	if ix := b.trivy.current(); ix.synced() {
		return ix
	}
	return nil
}

// wakePodDrawer has the open drawer read its model again when it shows a pod of b.
func (s *Session) wakePodDrawer(b *Backend) {
	st := s.state
	st.mu.Lock()
	d := st.drawer
	st.mu.Unlock()
	if d != nil && d.backend == b && isPodsKind(d.profile) {
		signal(d.wake)
	}
}

// probeTrivy is the Trivy integration's probe: whether the cluster serves the reports' kind (the
// custom kinds), and what the index's read of them met. It sends nothing.
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
	ix := b.trivy.current()
	if ix == nil {
		return IntegrationChecking, where, ""
	}
	err := ix.h.Err()
	switch {
	case cache.IsRefused(err):
		return IntegrationForbidden, where, "list " + trivyReportsKind
	case ix.h.HasSynced():
		n := ix.reports()
		if n == 1 {
			return IntegrationFound, where, "1 vulnerability report"
		}
		return IntegrationFound, where, fmt.Sprintf("%d vulnerability reports", n)
	case err != nil:
		return IntegrationUnavailable, where, errText(err)
	case ix.pending:
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
// (2026-10-17).

// groupImageRatings is the drawer's group heading of the image sections.
const groupImageRatings = "Image Ratings"

// imageRatings adds a pod's image sections to its drawer model, from the index alone.
func (s *Session) imageRatings(b *Backend, p *model.KindProfile, obj *unstructured.Unstructured, m *DetailModel) {
	if !isPodsKind(p) || obj == nil {
		return
	}
	ix := s.trivyIndexOf(b)
	if ix == nil {
		return
	}
	idx, _ := ix.index()
	secs := imageRatingSections(idx.podImages(obj), s.displayNow())
	m.Sections = append(m.Sections, s.TimeFormat().WriteSections(secs)...)
}

// PodImageRatings returns a pod's "Image Ratings" sections from Trivy Operator's report objects, as
// the session builds them from its index, with absolute times in ISO 8601 in now's zone (as
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
