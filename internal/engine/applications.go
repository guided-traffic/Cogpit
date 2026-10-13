package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"log/slog"
	"runtime/debug"
	"slices"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

// Applications (resource-parity › Cluster › Applications) are a virtual kind made of several
// kinds' cached objects: the workloads, grouped by their app.kubernetes.io/instance label
// (kinds.GroupApplications). A virtual kind's Derive sees one scope's objects only, so the
// applications get a cache of their own, as the local kinds do (kindscope.go): an informer over
// appsSource, which serves the grouped objects from a cache.Memory and keeps them current from the
// workloads' handles on the cluster's cache.
//
// The workloads are watched only while something reads the applications: the source holds their
// handles from the first list or watch of the applications' informer until its last watch stops
// (the view closed, the applications' warm period over), plus appsIdle. Nothing here asks the API
// server for anything: the handles share the cluster cache's informers with the workload views.

// appsInterval coalesces the workloads' events: the applications are grouped again at most this
// often, as a view's snapshot is built (architecture §5).
const appsInterval = 100 * time.Millisecond

// Tests shorten these.
var (
	// appsIdle keeps the workloads' handles across a watch the applications' reflector restarts.
	appsIdle = 2 * time.Second
	// appsWarmPeriod is the applications' informer's warm period; zero is the cache's default.
	appsWarmPeriod time.Duration
)

// errAppsClosed answers a list of the applications after their backend closed.
var errAppsClosed = errors.New("the cluster connection is closed")

// appsStore is a backend's Applications cache, made on first use and closed with the backend.
type appsStore struct {
	mu    sync.Mutex
	src   *appsSource
	cache *cache.Cache
	done  bool
}

func (a *appsStore) get(b *Backend) *cache.Cache {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cache == nil {
		a.src = &appsSource{b: b, mem: cache.NewMemory(), held: map[string]uint64{}}
		src := &cache.Held{Mem: a.src.mem, Use: a.src.use}
		a.cache = cache.New(src, cache.Options{Logger: b.logger(), WarmPeriod: appsWarmPeriod})
		if a.done {
			a.src.shutdown()
			a.cache.Close() // a closed backend hands out never-syncing handles
		}
	}
	return a.cache
}

// running reports whether the applications' informer runs, so that reading them costs nothing.
func (a *appsStore) running(scope cache.Scope) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.cache != nil && a.cache.Running(scope)
}

// wake has the running grouping look again at how its workload kinds are read (the session's
// replan: the access reviews answered, a read was refused).
func (a *appsStore) wake() {
	a.mu.Lock()
	src := a.src
	a.mu.Unlock()
	if src == nil {
		return
	}
	src.mu.Lock()
	r := src.run
	src.mu.Unlock()
	if r != nil {
		signal(r.wake)
	}
}

func (a *appsStore) close() {
	a.mu.Lock()
	c, src := a.cache, a.src
	a.done = true
	a.mu.Unlock()
	if src != nil {
		src.shutdown() // first: a list waiting for the workloads returns at once
	}
	if c != nil {
		c.Close()
	}
}

// isApplications reports whether a profile is the Applications view's.
func isApplications(p *model.KindProfile) bool {
	return p != nil && p.Key == kinds.ApplicationsKey && p.Group == model.LocalGroup
}

// withApplications adds the Applications view to a cluster's profiles when it serves a kind
// applications are made of.
func withApplications(profiles []*model.KindProfile) []*model.KindProfile {
	for _, p := range profiles {
		if slices.Contains(kinds.ApplicationWorkloadKinds, p.Key) {
			return append(slices.Clip(profiles), kinds.Applications())
		}
	}
	return profiles
}

// applicationsListable reports whether the user may list a kind applications are made of: any is
// enough, and one not known yet keeps the view shown (Backend.Listable).
func (b *Backend) applicationsListable(filter []string) (listable, known bool) {
	known = true
	for _, key := range kinds.ApplicationWorkloadKinds {
		p, ok := b.Profile(key)
		if !ok {
			continue
		}
		l, k := b.Listable(p, filter)
		if l && k {
			return true, true
		}
		if !k {
			known = false
		}
	}
	return !known, known
}

// appsSource fills the applications' Memory while an informer reads it (cache.Held): the running
// appsRun holds the workloads' handles and groups their objects.
type appsSource struct {
	b   *Backend
	mem *cache.Memory

	mu     sync.Mutex
	users  int      // lists in progress and open watches
	run    *appsRun // the workloads' handles while users > 0, and appsIdle after
	idle   *time.Timer
	closed bool

	buildMu sync.Mutex        // one grouping at a time, across runs
	held    map[string]uint64 // the Memory's objects by namespace/name: their content's hash
}

// use is cache.Held's Use: a list or watch of the applications' informer. Its wait blocks
// (bounded) until the run has grouped the listed workloads once.
func (s *appsSource) use() (wait func() error, done func(), err error) {
	r, err := s.acquire()
	if err != nil {
		return nil, nil, err
	}
	return func() error { return r.waitReady(syncTimeout) }, s.release, nil
}

// acquire starts the workloads' run if none is running and counts a user.
func (s *appsSource) acquire() (*appsRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, errAppsClosed
	}
	s.users++
	if s.idle != nil {
		s.idle.Stop()
		s.idle = nil
	}
	if s.run == nil {
		s.run = startAppsRun(s)
	}
	return s.run, nil
}

// release counts a user out; with the last, the run stops after appsIdle unless one comes back.
func (s *appsSource) release() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users--
	if s.users > 0 || s.closed || s.run == nil {
		return
	}
	if s.idle != nil {
		s.idle.Stop()
	}
	s.idle = time.AfterFunc(appsIdle, s.stopIfIdle)
}

func (s *appsSource) stopIfIdle() {
	s.mu.Lock()
	r := s.run
	if s.users > 0 || r == nil {
		s.mu.Unlock()
		return
	}
	s.run, s.idle = nil, nil
	s.mu.Unlock()
	r.stop()
}

// shutdown stops the run for good: the backend is closing.
func (s *appsSource) shutdown() {
	s.mu.Lock()
	r := s.run
	s.closed, s.run = true, nil
	if s.idle != nil {
		s.idle.Stop()
		s.idle = nil
	}
	s.mu.Unlock()
	if r != nil {
		r.stop()
	}
}

// appsRun holds a handle on each workload kind the cluster serves and groups their objects into
// the Memory after every change, coalesced.
type appsRun struct {
	s       *appsSource
	kinds   []appsKind // replaced only on the run's loop (followPlans)
	wake    chan struct{}
	quit    chan struct{}
	done    chan struct{}
	ready   chan struct{} // closed after the first grouping with every kind listed (or failed)
	readied sync.Once
	stopped sync.Once
	started time.Time

	statuses map[string]appsStatus // a workload's row status by UID, while its resourceVersion holds
}

type appsKind struct {
	profile *model.KindProfile
	handle  *cache.Handle
	key     string // how the handle reads the kind (Backend.readKey): taken again when that changes
	unsub   func()
}

type appsStatus struct {
	rv     string
	status model.Status
	seen   bool
}

func startAppsRun(s *appsSource) *appsRun {
	r := &appsRun{
		s: s, wake: make(chan struct{}, 1), quit: make(chan struct{}), done: make(chan struct{}),
		ready: make(chan struct{}), started: time.Now(), statuses: map[string]appsStatus{},
	}
	for _, key := range kinds.ApplicationWorkloadKinds {
		p, ok := s.b.Profile(key)
		if !ok {
			continue
		}
		r.kinds = append(r.kinds, r.acquireKind(p))
	}
	signal(r.wake)
	go r.loop()
	return r
}

// acquireKind takes a workload kind as the backend reads it now (in all namespaces, or in each
// namespace the user named when they may not list it everywhere), subscribed to its changes.
func (r *appsRun) acquireKind(p *model.KindProfile) appsKind {
	wake := r.wake
	k := appsKind{profile: p, key: r.s.b.readKey(p), handle: r.s.b.Acquire(p)}
	k.unsub = k.handle.Subscribe(func(cache.Event) { signal(wake) })
	return k
}

// followPlans takes a workload kind again when how the backend reads it changed: the access
// reviews answered, or the API server refused it in all namespaces (accessiblens_follow.go; the
// session's replan wakes the run, appsStore.wake). Before (2026-10-13) the run kept a refused read
// until the applications' informer gave up and started a new run. It runs on the run's loop.
func (r *appsRun) followPlans() {
	for i, k := range r.kinds {
		if r.s.b.readKey(k.profile) == k.key {
			continue
		}
		r.kinds[i] = r.acquireKind(k.profile)
		k.unsub()
		k.handle.Release()
	}
}

func (r *appsRun) loop() {
	defer close(r.done)
	defer func() {
		if p := recover(); p != nil {
			slog.Default().Error("engine: grouping the applications stopped: internal error", "panic", fmt.Sprint(p), "stack", string(debug.Stack()))
		}
	}()
	runCoalesced(r.quit, r.wake, appsInterval, r.unsettled, r.rebuild)
}

// stop ends the loop, waits for a grouping in progress, and releases the handles; the cluster's
// cache keeps the workloads' informers warm.
func (r *appsRun) stop() {
	r.stopped.Do(func() {
		close(r.quit)
		<-r.done
		for _, k := range r.kinds {
			k.unsub()
			k.handle.Release()
		}
	})
}

// settled reports whether every kind finished its initial list or failed it, and the first error
// when none listed.
func (r *appsRun) settled() (bool, error) {
	var firstErr error
	listed := 0
	for _, k := range r.kinds {
		switch err := k.handle.Err(); {
		case k.handle.HasSynced():
			listed++
		case err != nil:
			if firstErr == nil {
				firstErr = err
			}
		default:
			return false, nil
		}
	}
	if listed == 0 && firstErr != nil {
		return true, firstErr
	}
	return true, nil
}

// unsettled asks for a grouping while the first one waits for the kinds' initial lists, which
// arrive without an event the run sees.
func (r *appsRun) unsettled() bool {
	select {
	case <-r.ready:
		return false
	default:
		return true
	}
}

// waitReady waits (bounded) until the run has grouped the listed workloads once. It fails when
// no kind could be listed, with the first kind's error, so the view shows why.
func (r *appsRun) waitReady(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	select {
	case <-r.ready:
	case <-r.quit:
		return errAppsClosed
	case <-ctx.Done():
	}
	if _, err := r.settled(); err != nil {
		return err
	}
	return nil
}

// rebuild groups the cached workloads and brings the Memory in line: an application whose content
// changed is set again (a new resourceVersion), one that is gone is deleted.
func (r *appsRun) rebuild() {
	r.followPlans()
	settled, _ := r.settled()
	if !settled && time.Since(r.started) < syncTimeout {
		return
	}
	var ws []kinds.AppWorkload
	for _, k := range r.kinds {
		for _, obj := range k.handle.List() {
			ws = append(ws, kinds.AppWorkload{Kind: k.profile.Key, Object: obj, Status: r.status(k.profile, obj)})
		}
	}
	for uid, st := range r.statuses {
		if !st.seen {
			delete(r.statuses, uid)
			continue
		}
		st.seen = false
		r.statuses[uid] = st
	}
	apps := kinds.GroupApplications(ws)

	s := r.s
	s.buildMu.Lock()
	defer s.buildMu.Unlock()
	gvr := kinds.Applications().GVR()
	now := make(map[string]bool, len(apps))
	for _, app := range apps {
		key := app.GetNamespace() + "/" + app.GetName()
		now[key] = true
		sum := contentHash(app)
		if s.held[key] == sum {
			continue
		}
		s.mem.Set(gvr, app)
		s.held[key] = sum
	}
	for key := range s.held {
		if !now[key] {
			ns, name := splitKey(key)
			s.mem.Delete(gvr, ns, name)
			delete(s.held, key)
		}
	}
	r.readied.Do(func() { close(r.ready) })
}

// status is a workload's row status as its view colours it, kept while its resourceVersion holds.
func (r *appsRun) status(p *model.KindProfile, obj *unstructured.Unstructured) model.Status {
	uid, rv := p.Key+"/"+string(obj.GetUID()), obj.GetResourceVersion()
	if obj.GetUID() == "" {
		uid = p.Key + "/" + obj.GetNamespace() + "/" + obj.GetName()
	}
	if st, ok := r.statuses[uid]; ok && st.rv == rv && rv != "" {
		st.seen = true
		r.statuses[uid] = st
		return st.status
	}
	status := model.BuildRow(p, r.s.b.Context, obj).Status
	r.statuses[uid] = appsStatus{rv: rv, status: status, seen: true}
	return status
}

// contentHash identifies what an application shows; its resourceVersion moves only when it changes.
func contentHash(obj *unstructured.Unstructured) uint64 {
	b, err := json.Marshal(obj.Object)
	if err != nil {
		return 0
	}
	h := fnv.New64a()
	_, _ = h.Write(b)
	return h.Sum64()
}

func splitKey(key string) (namespace, name string) {
	for i := 0; i < len(key); i++ {
		if key[i] == '/' {
			return key[:i], key[i+1:]
		}
	}
	return "", key
}
