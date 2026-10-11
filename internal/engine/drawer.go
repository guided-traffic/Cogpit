package engine

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"

	"github.com/guided-traffic/cogpit/internal/cache"
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
	events  *cache.Handle // nil when the cluster serves no events
	unsubs  []func()
	wake    chan struct{}
	quit    chan struct{}
	stopped atomic.Bool
}

func (s *Session) startDrawerWatch(b *Backend, p *model.KindProfile, ref model.ObjectRef) *drawerWatch {
	d := &drawerWatch{
		backend: b, ref: ref, profile: p,
		obj:  b.Cache.Acquire(cache.Scope{GVR: p.GVR()}),
		wake: make(chan struct{}, 1),
		quit: make(chan struct{}),
	}
	d.unsubs = append(d.unsubs, d.obj.Subscribe(func(ev cache.Event) {
		if ev.Object.GetName() == ref.Name && ev.Object.GetNamespace() == d.namespace() {
			signal(d.wake)
		}
	}))
	if ep, ok := b.Profile(eventsKind); ok {
		d.events = b.Cache.Acquire(cache.Scope{GVR: ep.GVR()})
		d.unsubs = append(d.unsubs, d.events.Subscribe(func(ev cache.Event) {
			if d.involves(ev.Object) {
				signal(d.wake)
			}
		}))
	}
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
	obj = d.obj.HasSynced()
	events = d.events == nil || d.events.HasSynced()
	return obj, events
}

// involves reports whether a core v1 event is about the drawer's object.
func (d *drawerWatch) involves(ev *unstructured.Unstructured) bool {
	var uid string
	if o, ok := d.obj.Get(d.namespace(), d.ref.Name); ok {
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
}

// eventInvolves matches an event's involvedObject by UID when both are known, else by kind,
// namespace and name.
func eventInvolves(ev *unstructured.Unstructured, kind, namespace, name, uid string) bool {
	inv, ok := ev.Object["involvedObject"].(map[string]any)
	if !ok {
		return false
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
	h := b.Cache.Acquire(cache.Scope{GVR: p.GVR()})
	defer h.Release()
	var evh *cache.Handle
	if ep, ok := b.Profile(eventsKind); ok {
		evh = b.Cache.Acquire(cache.Scope{GVR: ep.GVR()})
		defer evh.Release()
	}
	if wait {
		wctx, cancel := context.WithTimeout(ctx, syncTimeout)
		defer cancel()
		h.WaitForSync(wctx)
		if evh != nil {
			evh.WaitForSync(wctx)
		}
	}

	m := &DetailModel{
		Ref: ref, Title: p.Kind + ": " + ref.Name,
		Sections: []model.DetailSection{}, Events: []EventItem{},
	}
	obj, ok := h.Get(ref.Namespace, ref.Name)
	if !ok {
		if h.HasSynced() {
			m.Missing = true
		} else {
			m.Loading = true
		}
		return m, nil
	}
	m.Status = model.BuildRow(p, b.Context, obj).Status
	m.Sections = append(m.Sections, model.MetadataSection(obj, s.Now()))
	m.Sections = append(m.Sections, s.profileSections(p, obj)...)
	if evh != nil {
		m.Events = objectEvents(evh.List(), p.Kind, ref.Namespace, ref.Name, string(obj.GetUID()))
	}
	if y, err := yaml.Marshal(model.Redact(obj).Object); err == nil {
		m.YAML = string(y)
	} else {
		s.log.Warn("engine: object not representable as YAML", "ref", ref.String(), "err", err)
	}
	if p.Key == "pods" && p.Group == "" {
		m.Containers = ContainerNames(obj, false)
	}
	return m, nil
}

// profileSections returns the kind's own drawer sections; a panicking profile yields none.
func (s *Session) profileSections(p *model.KindProfile, obj *unstructured.Unstructured) (out []model.DetailSection) {
	if p.Detail == nil {
		return nil
	}
	defer func() {
		if r := recover(); r != nil {
			s.log.Error("engine: kind profile detail panicked", "kind", p.Key, "panic", fmt.Sprint(r))
			out = nil
		}
	}()
	return p.Detail(obj)
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
