package cache

import (
	"strconv"
	"sync"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	toolscache "k8s.io/client-go/tools/cache"
)

// memoryHistory bounds the events a Memory keeps for watches that resume from an older
// resourceVersion; a watch from before them gets 410 Gone and its reflector lists again.
const memoryHistory = 1024

// memoryWatchBuffer is the event buffer of one watch. A watcher that falls this far behind is
// closed and resumes (or relists) instead of blocking writers.
const memoryWatchBuffer = 1024

// Memory is a Source over objects Cogpit makes itself rather than reads from a cluster: the local
// kinds (port forwards). Writers Set and Delete objects; informers over it list and watch them
// like a cluster's. It never talks to an API server. It is safe for concurrent use.
type Memory struct {
	mu       sync.Mutex
	rv       int64
	objs     map[schema.GroupVersionResource]map[string]*unstructured.Unstructured
	history  map[schema.GroupVersionResource][]memoryEvent
	watchers map[schema.GroupVersionResource]map[*memoryWatch]struct{}
}

type memoryEvent struct {
	rv  int64
	typ watch.EventType
	obj *unstructured.Unstructured
}

// NewMemory returns an empty Memory.
func NewMemory() *Memory {
	return &Memory{
		objs:     map[schema.GroupVersionResource]map[string]*unstructured.Unstructured{},
		history:  map[schema.GroupVersionResource][]memoryEvent{},
		watchers: map[schema.GroupVersionResource]map[*memoryWatch]struct{}{},
	}
}

func memoryKey(namespace, name string) string {
	if namespace == "" {
		return name
	}
	return namespace + "/" + name
}

// Set adds or replaces an object (a copy of obj) and sends Added or Modified to the watches. It
// assigns the resourceVersion.
func (m *Memory) Set(gvr schema.GroupVersionResource, obj *unstructured.Unstructured) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rv++
	obj = obj.DeepCopy()
	obj.SetResourceVersion(strconv.FormatInt(m.rv, 10))
	byKey := m.objs[gvr]
	if byKey == nil {
		byKey = map[string]*unstructured.Unstructured{}
		m.objs[gvr] = byKey
	}
	key := memoryKey(obj.GetNamespace(), obj.GetName())
	typ := watch.Added
	if _, ok := byKey[key]; ok {
		typ = watch.Modified
	}
	byKey[key] = obj
	m.publishLocked(gvr, memoryEvent{rv: m.rv, typ: typ, obj: obj})
}

// Delete removes an object and sends Deleted to the watches; a missing object is a no-op.
func (m *Memory) Delete(gvr schema.GroupVersionResource, namespace, name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := memoryKey(namespace, name)
	obj, ok := m.objs[gvr][key]
	if !ok {
		return
	}
	delete(m.objs[gvr], key)
	m.rv++
	obj = obj.DeepCopy()
	obj.SetResourceVersion(strconv.FormatInt(m.rv, 10))
	m.publishLocked(gvr, memoryEvent{rv: m.rv, typ: watch.Deleted, obj: obj})
}

// Get returns a copy of one object.
func (m *Memory) Get(gvr schema.GroupVersionResource, namespace, name string) (*unstructured.Unstructured, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	obj, ok := m.objs[gvr][memoryKey(namespace, name)]
	if !ok {
		return nil, false
	}
	return obj.DeepCopy(), true
}

func (m *Memory) publishLocked(gvr schema.GroupVersionResource, ev memoryEvent) {
	h := append(m.history[gvr], ev)
	if len(h) > memoryHistory {
		h = append(h[:0:0], h[len(h)-memoryHistory:]...)
	}
	m.history[gvr] = h
	for w := range m.watchers[gvr] {
		w.send(ev)
	}
}

// ListerWatcher implements Source.
func (m *Memory) ListerWatcher(scope Scope) toolscache.ListerWatcher {
	return &memoryLW{m: m, scope: scope}
}

type memoryLW struct {
	m     *Memory
	scope Scope
}

// IsWatchListSemanticsUnSupported tells client-go's reflector to list, then watch.
func (lw *memoryLW) IsWatchListSemanticsUnSupported() bool { return true }

func (lw *memoryLW) List(_ metav1.ListOptions) (runtime.Object, error) {
	lw.m.mu.Lock()
	defer lw.m.mu.Unlock()
	list := listObject()
	for _, o := range lw.m.objs[lw.scope.GVR] {
		if lw.scope.Namespace != "" && o.GetNamespace() != lw.scope.Namespace {
			continue
		}
		if f := FilterScope(lw.scope, o); f != nil {
			list.Items = append(list.Items, *f.DeepCopy())
		}
	}
	list.SetResourceVersion(strconv.FormatInt(lw.m.rv, 10))
	return list, nil
}

func (lw *memoryLW) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	from, _ := strconv.ParseInt(opts.ResourceVersion, 10, 64)
	lw.m.mu.Lock()
	defer lw.m.mu.Unlock()
	h := lw.m.history[lw.scope.GVR]
	if from > 0 && len(h) > 0 && h[0].rv > from+1 {
		return nil, apierrors.NewResourceExpired("the requested resourceVersion is too old")
	}
	w := &memoryWatch{
		ch: make(chan watch.Event, memoryWatchBuffer), done: make(chan struct{}), scope: lw.scope,
	}
	for _, ev := range h {
		if ev.rv > from {
			w.send(ev)
		}
	}
	set := lw.m.watchers[lw.scope.GVR]
	if set == nil {
		set = map[*memoryWatch]struct{}{}
		lw.m.watchers[lw.scope.GVR] = set
	}
	set[w] = struct{}{}
	gvr := lw.scope.GVR
	w.remove = func() {
		lw.m.mu.Lock()
		delete(lw.m.watchers[gvr], w)
		lw.m.mu.Unlock()
	}
	return w, nil
}

// memoryWatch is one watch. send runs with Memory.mu held and never blocks: a full buffer closes
// the watch, and the reflector resumes from its last resourceVersion.
type memoryWatch struct {
	ch     chan watch.Event
	done   chan struct{}
	once   sync.Once
	closed bool // guarded by Memory.mu
	scope  Scope
	remove func()
}

func (w *memoryWatch) send(ev memoryEvent) {
	if w.closed {
		return
	}
	if w.scope.Namespace != "" && ev.obj.GetNamespace() != w.scope.Namespace {
		return
	}
	obj := FilterScope(w.scope, ev.obj)
	if obj == nil {
		return
	}
	select {
	case <-w.done:
		w.closed = true
	case w.ch <- watch.Event{Type: ev.typ, Object: obj.DeepCopy()}:
	default:
		w.closed = true
		close(w.ch) // the reflector sees the end of the watch and resumes
		go w.Stop()
	}
}

func (w *memoryWatch) ResultChan() <-chan watch.Event { return w.ch }

func (w *memoryWatch) Stop() {
	w.once.Do(func() {
		close(w.done)
		if w.remove != nil {
			go w.remove() // Stop may be called while Memory holds its lock
		}
	})
}

// compile-time check that the list-watcher has the context-free methods client-go needs.
var _ toolscache.ListerWatcher = (*memoryLW)(nil)
