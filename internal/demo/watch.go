package demo

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"slices"
	"strconv"
	"sync"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	toolscache "k8s.io/client-go/tools/cache"
)

// defaultQueueLimit is how many events a watcher may have pending before it is closed: five
// seconds at the default rate. A reflector drains far faster; a watcher this far behind belongs
// to a consumer that is stuck, and closing it bounds the memory its queue holds.
const defaultQueueLimit = 5_000

// resultBuffer is the capacity of a watcher's result channel.
const resultBuffer = 256

// source implements cache.Source over the cluster.
type source struct{ c *Cluster }

// ListerWatcher implements cache.Source.
func (s source) ListerWatcher(gvr schema.GroupVersionResource, namespace string) toolscache.ListerWatcher {
	k, ok := kindOf(gvr)
	return &listerWatcher{c: s.c, gvr: gvr, kind: k, known: ok, namespace: namespace}
}

// listerWatcher lists and watches one resource, in one namespace or (empty) in all.
type listerWatcher struct {
	c         *Cluster
	gvr       schema.GroupVersionResource
	kind      kind
	known     bool
	namespace string
}

// IsWatchListSemanticsUnSupported tells client-go's reflector to list, then watch: the demo does
// not stream initial lists.
func (lw *listerWatcher) IsWatchListSemanticsUnSupported() bool { return true }

// List implements toolscache.Lister.
func (lw *listerWatcher) List(opts metav1.ListOptions) (runtime.Object, error) {
	return lw.ListWithContext(context.Background(), opts)
}

// Watch implements toolscache.Watcher.
func (lw *listerWatcher) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return lw.WatchWithContext(context.Background(), opts)
}

// ListWithContext returns the current objects sorted by namespace and name, with the cluster's
// resourceVersion. It ignores selectors, limits and the requested resourceVersion: the snapshot
// is always the newest, which satisfies every resourceVersion a reflector asks for.
func (lw *listerWatcher) ListWithContext(_ context.Context, _ metav1.ListOptions) (runtime.Object, error) {
	c := lw.c
	c.lists.Add(1)
	if !lw.known {
		return nil, apierrors.NewNotFound(lw.gvr.GroupResource(), "")
	}
	c.mu.Lock()
	items := make([]*unstructured.Unstructured, 0, len(c.store[lw.kind]))
	for _, o := range c.store[lw.kind] {
		if lw.namespace == "" || o.GetNamespace() == lw.namespace {
			items = append(items, o)
		}
	}
	rv := c.rv
	c.mu.Unlock()

	// Published objects are immutable, so copying happens outside the lock.
	slices.SortFunc(items, func(a, b *unstructured.Unstructured) int {
		return cmp.Or(cmp.Compare(a.GetNamespace(), b.GetNamespace()), cmp.Compare(a.GetName(), b.GetName()))
	})
	info := kindInfo[lw.kind]
	list := &unstructured.UnstructuredList{Object: map[string]any{}}
	list.SetAPIVersion(info.apiVersion)
	list.SetKind(info.kind + "List")
	list.Items = make([]unstructured.Unstructured, len(items))
	for i, o := range items {
		o.DeepCopyInto(&list.Items[i])
	}
	list.SetResourceVersion(strconv.FormatInt(rv, 10))
	return list, nil
}

// WatchWithContext starts a watch. From "" or "0" it begins with a synthetic Added event per
// current object; from a resourceVersion in the history it replays the compacted changes since
// (see the package comment); from an older one it answers 410 Gone.
func (lw *listerWatcher) WatchWithContext(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	c := lw.c
	c.watches.Add(1)
	if !lw.known {
		return nil, apierrors.NewNotFound(lw.gvr.GroupResource(), "")
	}
	from := int64(-1)
	if opts.ResourceVersion != "" && opts.ResourceVersion != "0" {
		v, err := strconv.ParseInt(opts.ResourceVersion, 10, 64)
		if err != nil || v < 0 {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid resourceVersion %q", opts.ResourceVersion))
		}
		from = v
	}
	w := &watcher{
		c:         c,
		kind:      lw.kind,
		namespace: lw.namespace,
		bookmarks: opts.AllowWatchBookmarks,
		result:    make(chan watch.Event, resultBuffer),
		wake:      make(chan struct{}, 1),
		done:      make(chan struct{}),
	}

	w.stopWith(ctx, opts.TimeoutSeconds)

	c.mu.Lock()
	switch {
	case from < 0:
		w.queue = c.initialEvents(lw.kind, lw.namespace)
	case from < c.compacted:
		compacted := c.compacted
		c.mu.Unlock()
		c.expired.Add(1)
		w.Stop()
		return expiredWatch(from, compacted), nil
	default:
		w.queue = c.replay(lw.kind, lw.namespace, from)
	}
	w.queueLimit = c.queueLimit
	w.limit = len(w.queue) + w.queueLimit
	c.watchers[lw.kind][w] = struct{}{}
	c.mu.Unlock()

	go w.pump()
	return w, nil
}

// stopWith stops the watcher when ctx ends or after the watch's timeout: the API server ends a
// watch after its timeout, and the reflector then resumes it.
func (w *watcher) stopWith(ctx context.Context, timeoutSeconds *int64) {
	stopCtx := context.AfterFunc(ctx, w.Stop)
	var timer *time.Timer
	if timeoutSeconds != nil && *timeoutSeconds > 0 {
		timer = time.AfterFunc(time.Duration(*timeoutSeconds)*time.Second, w.Stop)
	}
	w.mu.Lock()
	w.timer, w.stopCtx = timer, stopCtx
	w.mu.Unlock()
	select {
	case <-w.done: // stopped before the fields were set: release them here
		stopCtx()
		if timer != nil {
			timer.Stop()
		}
	default:
	}
}

// initialEvents returns an Added change per current object, sorted by resourceVersion. Called
// with c.mu held.
func (c *Cluster) initialEvents(k kind, namespace string) []change {
	out := make([]change, 0, len(c.store[k]))
	for key, o := range c.store[k] {
		ns := o.GetNamespace()
		if namespace != "" && ns != namespace {
			continue
		}
		rv, _ := strconv.ParseInt(o.GetResourceVersion(), 10, 64)
		out = append(out, change{rv: rv, kind: k, typ: watch.Added, namespace: ns, key: key, obj: o})
	}
	slices.SortFunc(out, func(a, b change) int { return cmp.Compare(a.rv, b.rv) })
	return out
}

// replay returns the compacted changes of kind k since rv: per object its latest change, with
// the current state, in resourceVersion order. An object created after rv is Added, one that
// existed at rv is Modified, one deleted since is Deleted. Called with c.mu held.
func (c *Cluster) replay(k kind, namespace string, rv int64) []change {
	type span struct {
		first watch.EventType
		last  int64
	}
	spans := map[string]span{}
	c.history.since(rv, func(ch *change) {
		if ch.kind != k || (namespace != "" && ch.namespace != namespace) {
			return
		}
		s, seen := spans[ch.key]
		if !seen {
			s.first = ch.typ
		}
		s.last = ch.rv
		spans[ch.key] = s
	})
	out := make([]change, 0, len(spans))
	c.history.since(rv, func(ch *change) {
		s, ok := spans[ch.key]
		if ch.kind != k || !ok || s.last != ch.rv {
			return
		}
		ev := *ch
		switch {
		case ch.typ == watch.Deleted:
			// ch.obj is the last state.
		case s.first == watch.Added:
			ev.typ, ev.obj = watch.Added, c.store[k][ch.key]
		default:
			ev.typ, ev.obj = watch.Modified, c.store[k][ch.key]
		}
		if ev.obj != nil {
			out = append(out, ev)
		}
	})
	return out
}

// sendBookmarks queues a bookmark at the current resourceVersion for every watcher that asked
// for them. Called with c.mu held.
func (c *Cluster) sendBookmarks() {
	for k := range numKinds {
		for w := range c.watchers[k] {
			if w.bookmarks {
				w.enqueue(change{rv: c.rv, kind: k, typ: watch.Bookmark})
			}
		}
	}
}

// watcher is one open watch. The generator appends to its queue under the cluster lock and never
// blocks; the pump goroutine deep-copies the objects and sends them on the result channel.
type watcher struct {
	c         *Cluster
	kind      kind
	namespace string
	bookmarks bool
	result    chan watch.Event
	wake      chan struct{} // capacity 1: "the queue has entries"
	done      chan struct{} // closed by Stop
	stopOnce  sync.Once

	queueLimit int // pending events allowed once the initial or replayed ones are taken

	mu      sync.Mutex
	queue   []change    // guarded by mu
	limit   int         // guarded by mu
	timer   *time.Timer // guarded by mu
	stopCtx func() bool // guarded by mu
}

// enqueue adds a change, or closes the watcher when its consumer has fallen too far behind.
// Called with c.mu held; it never blocks.
func (w *watcher) enqueue(ch change) {
	if ch.typ != watch.Bookmark && w.namespace != "" && ch.namespace != w.namespace {
		return
	}
	select {
	case <-w.done:
		return
	default:
	}
	w.mu.Lock()
	if len(w.queue) >= w.limit {
		w.mu.Unlock()
		w.c.dropped.Add(1)
		w.Stop()
		return
	}
	w.queue = append(w.queue, ch)
	w.mu.Unlock()
	select {
	case w.wake <- struct{}{}:
	default:
	}
}

// pump delivers the queue until the watcher stops, then leaves the cluster's watcher set and
// closes the result channel, which ends the consumer's watch. Pending events are dropped; the
// reflector resumes from the last resourceVersion it processed.
func (w *watcher) pump() {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("demo: watch stopped: internal error", "resource", kindInfo[w.kind].gvr.Resource, "panic", r, "stack", string(debug.Stack()))
		}
		w.Stop()
		w.c.mu.Lock()
		delete(w.c.watchers[w.kind], w)
		w.c.mu.Unlock()
		close(w.result)
	}()
	// Two buffers alternate: the generator appends to one while the pump sends from the other.
	var spare []change
	for {
		w.mu.Lock()
		batch := w.queue
		w.queue = spare[:0]
		w.limit = w.queueLimit // the initial or replayed events, if any, are taken
		w.mu.Unlock()
		spare = batch // whatever happens, batch's array is the one the queue does not use
		if len(batch) == 0 {
			select {
			case <-w.wake:
				continue
			case <-w.done:
				return
			}
		}
		for i := range batch {
			ev := w.event(&batch[i])
			select {
			case w.result <- ev:
			case <-w.done:
				return
			}
		}
		clear(batch) // drop the object references
	}
}

// event turns a change into the watch event the consumer owns.
func (w *watcher) event(ch *change) watch.Event {
	if ch.typ == watch.Bookmark {
		info := kindInfo[w.kind]
		b := &unstructured.Unstructured{Object: map[string]any{}}
		b.SetAPIVersion(info.apiVersion)
		b.SetKind(info.kind)
		b.SetResourceVersion(strconv.FormatInt(ch.rv, 10))
		return watch.Event{Type: watch.Bookmark, Object: b}
	}
	return watch.Event{Type: ch.typ, Object: ch.obj.DeepCopy()}
}

// Stop implements watch.Interface. It is idempotent and safe from any goroutine.
func (w *watcher) Stop() {
	w.stopOnce.Do(func() {
		close(w.done)
		w.mu.Lock()
		timer, stopCtx := w.timer, w.stopCtx
		w.mu.Unlock()
		if timer != nil {
			timer.Stop()
		}
		if stopCtx != nil {
			stopCtx()
		}
	})
}

// ResultChan implements watch.Interface.
func (w *watcher) ResultChan() <-chan watch.Event { return w.result }

// expiredWatch is the API server's answer to a watch from a resourceVersion it no longer has: one
// ERROR event with 410 Gone, then the end of the watch.
func expiredWatch(from, compacted int64) watch.Interface {
	err := apierrors.NewResourceExpired(fmt.Sprintf("too old resource version: %d (%d)", from, compacted))
	ch := make(chan watch.Event, 1)
	ch <- watch.Event{Type: watch.Error, Object: &err.ErrStatus}
	close(ch)
	return closedWatch{ch: ch}
}

// closedWatch is a watch whose events are all in its channel already.
type closedWatch struct{ ch chan watch.Event }

// Stop implements watch.Interface.
func (closedWatch) Stop() {}

// ResultChan implements watch.Interface.
func (w closedWatch) ResultChan() <-chan watch.Event { return w.ch }
