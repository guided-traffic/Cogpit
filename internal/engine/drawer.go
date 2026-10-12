package engine

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/metrics"
	"github.com/guided-traffic/cogpit/internal/model"
)

// eventsKind is the kind key of core v1 events, which the drawer shows for its object.
const eventsKind = "events"

// maxDetailEvents caps the events a detail model carries.
const maxDetailEvents = 50

// drawerWatch keeps the drawer current: while the drawer is open it holds a handle on the
// object's kind and one on the events in all namespaces, and emits EventDetail, coalesced to at
// most one per SnapshotInterval, when the object or its events change.
type drawerWatch struct {
	backend *Backend
	ref     model.ObjectRef
	profile *model.KindProfile
	obj     *cache.Handle
	events  *cache.Handle            // nil when the cluster serves no events
	related *relatedSet              // the indexes of the profile's Needs; nil without
	usage   *metrics.Subscription    // the pod's or node's metrics poll (metrics.go); nil without
	wmu     sync.Mutex               // guards windows
	windows map[string]time.Duration // the time range of each chart, by target name (metrics.go)
	unsubs  []func()
	wake    chan struct{}
	quit    chan struct{}
	stopped atomic.Bool

	// variants is what each chart shows, by target name; absent: CPU (metrics.go). Guarded by wmu.
	variants map[string]string
}

func (s *Session) startDrawerWatch(b *Backend, p *model.KindProfile, ref model.ObjectRef) *drawerWatch {
	d := &drawerWatch{
		backend: b, ref: ref, profile: p,
		obj:  b.Acquire(p),
		wake: make(chan struct{}, 1),
		quit: make(chan struct{}),
	}
	d.unsubs = append(d.unsubs, d.obj.Subscribe(func(ev cache.Event) {
		if ev.Object.GetName() == ref.Name && ev.Object.GetNamespace() == d.namespace() {
			signal(d.wake)
		}
	}))
	if ep, ok := b.Profile(eventsKind); ok && p.APIObject() { // no event involves a virtual or local kind's object
		d.events = b.Acquire(ep)
		d.unsubs = append(d.unsubs, d.events.Subscribe(func(ev cache.Event) {
			if d.involves(ev.Object) {
				signal(d.wake)
			}
		}))
	}
	// Woken only by changes to the keys the drawer's model looked up (a node's pods, not every
	// pod); Detail reads through this set while the drawer shows ref (drawerRelated).
	d.related = newRelatedSetTracking(b, p.Needs, func() { signal(d.wake) }, true)
	d.usage = s.startDrawerUsage(b, p, ref)
	objSynced, evSynced := d.synced()
	poll := func() bool {
		o, e := d.synced()
		changed := o != objSynced || e != evSynced
		objSynced, evSynced = o, e
		return changed
	}
	s.goSafe("detail drawer", func() {
		runCoalesced(d.quit, d.wake, s.opts.SnapshotInterval, poll, func() {
			if !d.stopped.Load() {
				s.emit(Event{Type: EventDetail, View: ref.Kind})
			}
		})
	})
	return d
}

func (d *drawerWatch) namespace() string {
	if d.profile.Namespaced {
		return d.ref.Namespace
	}
	return ""
}

func (d *drawerWatch) synced() (obj, events bool) {
	obj = d.obj.HasSynced() && d.related.synced()
	events = d.events == nil || d.events.HasSynced()
	return obj, events
}

// involves reports whether a core v1 event is about the drawer's object.
func (d *drawerWatch) involves(ev *unstructured.Unstructured) bool {
	var uid string
	if o, ok := KindGet(d.obj, d.profile, d.namespace(), d.ref.Name); ok {
		uid = string(o.GetUID())
	}
	return eventInvolves(ev, d.profile.Kind, d.namespace(), d.ref.Name, uid)
}

func (d *drawerWatch) stop() {
	if !d.stopped.CompareAndSwap(false, true) {
		return
	}
	close(d.quit)
	for _, u := range d.unsubs {
		u()
	}
	d.obj.Release()
	if d.events != nil {
		d.events.Release()
	}
	d.related.release()
	d.usage.Release()
}

// eventInvolves matches an event's involvedObject by UID when both are known, else by kind,
// namespace and name. Nodes match by kind and name: the kubelet reports node events with the
// node's name as the involved object's UID, not the Node object's UID.
func eventInvolves(ev *unstructured.Unstructured, kind, namespace, name, uid string) bool {
	inv, ok := ev.Object["involvedObject"].(map[string]any)
	if !ok {
		return false
	}
	if kind == "Node" {
		k, _ := inv["kind"].(string)
		n, _ := inv["name"].(string)
		return k == kind && n == name
	}
	if evUID, _ := inv["uid"].(string); evUID != "" && uid != "" {
		return evUID == uid
	}
	k, _ := inv["kind"].(string)
	ns, _ := inv["namespace"].(string)
	n, _ := inv["name"].(string)
	return k == kind && ns == namespace && n == name
}

// Detail builds the drawer model for ref from the cache, without waiting: an object whose kind is
// still being listed comes back with Loading set, one that is not in the cache with Missing set.
func (s *Session) Detail(ref model.ObjectRef) (*DetailModel, error) {
	return s.detail(context.Background(), ref, false)
}

// DetailWait is Detail after waiting, bounded by ctx and 15 seconds, for the object's kind and
// the events to finish their initial list. Agents and the CLI use it; the GUI uses Detail.
func (s *Session) DetailWait(ctx context.Context, ref model.ObjectRef) (*DetailModel, error) {
	return s.detail(ctx, ref, true)
}

func (s *Session) detail(ctx context.Context, ref model.ObjectRef, wait bool) (*DetailModel, error) {
	b, ref, p, err := s.resolve(ref)
	if err != nil {
		return nil, err
	}
	h := b.Acquire(p)
	defer h.Release()
	var evh *cache.Handle
	if ep, ok := b.Profile(eventsKind); ok && p.APIObject() {
		evh = b.Acquire(ep)
		defer evh.Release()
	}
	rs := s.drawerRelated(b, p, ref)
	if rs == nil {
		rs = newRelatedSet(b, p.Needs, nil)
	}
	defer rs.release()
	if wait {
		wctx, cancel := context.WithTimeout(ctx, syncTimeout)
		defer cancel()
		h.WaitForSync(wctx)
		if evh != nil {
			evh.WaitForSync(wctx)
		}
		rs.waitSynced(wctx)
	}

	obj, ok := KindGet(h, p, ref.Namespace, ref.Name)
	if !ok {
		m := &DetailModel{Ref: ref, Title: detailTitle(p, ref), Sections: []model.DetailSection{}, Events: []EventItem{}, NoEvents: !p.APIObject(), Plain: p.Derive != nil}
		if h.HasSynced() {
			m.Missing = true
		} else {
			m.Loading = true
		}
		return m, nil
	}
	var events []*unstructured.Unstructured
	if evh != nil {
		events = evh.List()
	}
	rel, _ := rs.view()
	m, perr := BuildDetailWith(p, ref, obj, events, s.Now(), rel)
	if perr != nil {
		s.log.Error("engine: kind profile detail panicked", "kind", p.Key, "panic", perr.Error())
	}
	if OriginOf(ctx).IsAgent() {
		m.Sections = model.WithholdPrivate(m.Sections) // object_get: agents never read private values
	}
	if m.YAML == "" {
		s.log.Warn("engine: object not representable as YAML", "ref", ref.String())
	}
	return m, nil
}

// drawerRelated returns the open drawer's related set, held, when the drawer shows ref through p
// on b, else nil. A fetch of the drawer's model reads the indexes the drawer keeps current instead
// of building its own, and its lookups tell the drawer which keys to wake for.
func (s *Session) drawerRelated(b *Backend, p *model.KindProfile, ref model.ObjectRef) *relatedSet {
	st := s.state
	st.mu.Lock()
	defer st.mu.Unlock()
	d := st.drawer
	if d == nil || d.backend != b || d.profile != p || !d.ref.Same(ref) || !d.related.hold() {
		return nil
	}
	return d.related
}

// BuildDetail assembles the drawer model of one object from its profile, the object and the
// cached events (of all objects; the object's own are picked out), with ages against now. It is
// the drawer's content without the cache, for the engine and for tests of the UI.
//
// The sections are Properties (the generic metadata, then the items of the kind's own Properties
// section) and the kind's other sections; the events follow, newest first. A profile whose Detail
// panics yields only the generic sections, and the panic as the error.
func BuildDetail(p *model.KindProfile, ref model.ObjectRef, obj *unstructured.Unstructured, events []*unstructured.Unstructured, now time.Time) (*DetailModel, error) {
	return BuildDetailWith(p, ref, obj, events, now, nil)
}

// BuildDetailWith is BuildDetail with the related objects of the profile's Needs; nil means none.
func BuildDetailWith(p *model.KindProfile, ref model.ObjectRef, obj *unstructured.Unstructured, events []*unstructured.Unstructured, now time.Time, rel model.Related) (*DetailModel, error) {
	m := &DetailModel{
		Ref: ref, Title: detailTitle(p, ref),
		Sections: []model.DetailSection{}, Events: []EventItem{},
	}
	m.Status = model.BuildRowWith(p, ref.Context, obj, rel).Status
	m.Metrics = p.Usage != ""
	m.NoEvents = p.Local // no event involves an object Cogpit makes itself (port forwards, Helm charts)
	kind, err := profileSections(p, obj, rel)
	if p.Derive != nil { // a virtual kind's drawer is the profile's alone (model.KindProfile.Detail)
		m.NoEvents, m.Plain = true, true
		if kind != nil {
			m.Sections = kind
		}
	} else {
		meta := obj
		if isSecrets(p) {
			// The cache drops a Secret's last-applied annotation, which repeats its values; should
			// one arrive another way, its Annotations show it redacted.
			meta = model.Redact(obj)
		}
		m.Sections = model.MergeProperties(model.PropertiesSection(meta, now), kind)
		if isSecrets(p) {
			m.Sections = append(m.Sections, secretDataSection(ref, obj)) // keys and sizes, values masked
		}
	}
	if len(events) > 0 && !m.NoEvents {
		m.Events = objectEvents(events, p.Kind, ref.Namespace, ref.Name, string(obj.GetUID()))
	}
	if y, yerr := yaml.Marshal(model.Redact(obj).Object); yerr == nil {
		m.YAML = string(y)
	}
	if p.Key == "pods" && p.Group == "" {
		m.Containers = ContainerNames(obj, false)
		m.InitContainers = containerNames(obj, "initContainers")
	}
	return m, err
}

// detailTitle is the drawer's title: "Pod: web-0", "Release: web" (a profile's DetailTitle).
func detailTitle(p *model.KindProfile, ref model.ObjectRef) string {
	if p.DetailTitle != "" {
		return p.DetailTitle + ": " + ref.Name
	}
	return p.Kind + ": " + ref.Name
}

// profileSections returns the kind's own drawer sections; a panicking profile yields none and
// the panic as an error.
func profileSections(p *model.KindProfile, obj *unstructured.Unstructured, rel model.Related) (out []model.DetailSection, err error) {
	if p.Detail == nil && p.DetailWith == nil {
		return nil, nil
	}
	defer func() {
		if r := recover(); r != nil {
			out, err = nil, fmt.Errorf("%v", r)
		}
	}()
	return p.DetailSections(obj, rel), nil
}

// objectEvents returns the events about one object, newest first, at most maxDetailEvents.
func objectEvents(all []*unstructured.Unstructured, kind, namespace, name, uid string) []EventItem {
	out := []EventItem{}
	for _, ev := range all {
		if eventInvolves(ev, kind, namespace, name, uid) {
			out = append(out, eventItem(ev))
		}
	}
	slices.SortStableFunc(out, func(a, b EventItem) int {
		if a.LastSeen != b.LastSeen {
			if a.LastSeen > b.LastSeen {
				return -1
			}
			return 1
		}
		return strings.Compare(a.Reason, b.Reason)
	})
	if len(out) > maxDetailEvents {
		out = out[:maxDetailEvents]
	}
	return out
}

func eventItem(ev *unstructured.Unstructured) EventItem {
	it := EventItem{
		Type:    nestedString(ev.Object, "type"),
		Reason:  nestedString(ev.Object, "reason"),
		Message: nestedString(ev.Object, "message"),
	}
	src := nestedString(ev.Object, "source", "component")
	if src == "" {
		src = nestedString(ev.Object, "reportingComponent")
	}
	if host := nestedString(ev.Object, "source", "host"); host != "" && src != "" {
		src += " " + host
	}
	it.Source = src
	it.Count = nestedInt(ev.Object, "count")
	if it.Count == 0 {
		it.Count = nestedInt(ev.Object, "series", "count")
	}
	for _, path := range [][]string{
		{"series", "lastObservedTime"}, {"lastTimestamp"}, {"eventTime"}, {"firstTimestamp"},
		{"metadata", "creationTimestamp"},
	} {
		if t, err := time.Parse(time.RFC3339Nano, nestedString(ev.Object, path...)); err == nil {
			it.LastSeen = t.Unix()
			break
		}
	}
	switch it.Type {
	case "Warning":
		it.Status = model.StatusWarning
	case "Normal":
		it.Status = model.StatusOK
	}
	return it
}

// ContainerNames returns the container names of a pod: spec.containers, then, with init, the
// init containers. Other shapes yield none.
func ContainerNames(obj *unstructured.Unstructured, init bool) []string {
	out := containerNames(obj, "containers")
	if init {
		out = append(out, containerNames(obj, "initContainers")...)
	}
	return out
}

// containerNames returns the names in a pod's spec.<field> list.
func containerNames(obj *unstructured.Unstructured, field string) []string {
	v, _, _ := unstructured.NestedFieldNoCopy(obj.Object, "spec", field)
	list, _ := v.([]any)
	out := make([]string, 0, len(list))
	for _, c := range list {
		if m, ok := c.(map[string]any); ok {
			if n, _ := m["name"].(string); n != "" {
				out = append(out, n)
			}
		}
	}
	return out
}

// nestedString reads a string field without copying and without failing on other shapes.
func nestedString(obj map[string]any, fields ...string) string {
	v, _, _ := unstructured.NestedFieldNoCopy(obj, fields...)
	s, _ := v.(string)
	return s
}

// nestedInt reads an integer field that JSON decoding may have produced as int64 or float64.
func nestedInt(obj map[string]any, fields ...string) int64 {
	v, _, _ := unstructured.NestedFieldNoCopy(obj, fields...)
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case int32:
		return int64(n)
	case float64:
		return int64(n)
	}
	return 0
}
