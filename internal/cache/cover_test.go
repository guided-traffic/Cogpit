package cache_test

import (
	"context"
	"fmt"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/resourceversion"
	toolscache "k8s.io/client-go/tools/cache"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// The covering scopes of architecture §5: a label-selected scope is served from the running
// unselected scope of the same resource, namespace and metadata flag.

var (
	allSecrets  = cache.Scope{GVR: testkit.SecretsGVR}
	helmSecrets = cache.Scope{GVR: testkit.SecretsGVR, LabelSelector: "owner=helm"}
)

// release is a Helm release Secret, labelled owner=helm.
func release(ns, name string, rev int) *unstructured.Unstructured {
	return testkit.Secret(ns, "sh.helm.release.v1."+name+".v"+strconv.Itoa(rev),
		map[string]string{"owner": "helm", "name": name}, map[string]string{"release": name + "-" + strconv.Itoa(rev)})
}

// recorder checks the events a handle's subscriber gets: no add of an object it holds, no update
// or delete of one it does not, no version older than or equal to the last one it got for that
// object; and it rebuilds the objects from them.
type recorder struct {
	mu     sync.Mutex
	held   map[string]*unstructured.Unstructured
	lastRV map[string]string
	errs   []string
	counts map[cache.EventType]int
	hist   map[string][]string
}

// record subscribes to h and starts from what h holds; call it while nothing changes.
func record(h *cache.Handle) (*recorder, func()) {
	r := &recorder{held: map[string]*unstructured.Unstructured{}, lastRV: map[string]string{}, counts: map[cache.EventType]int{}, hist: map[string][]string{}}
	r.mu.Lock()
	defer r.mu.Unlock()
	cancel := h.Subscribe(r.on)
	for _, o := range h.List() {
		k := o.GetNamespace() + "/" + o.GetName()
		r.held[k], r.lastRV[k] = o, o.GetResourceVersion()
	}
	return r, cancel
}

func newer(a, b string) bool {
	n, err := resourceversion.CompareResourceVersion(a, b)
	return err == nil && n > 0
}

func (r *recorder) on(ev cache.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := ev.Object.GetNamespace() + "/" + ev.Object.GetName()
	rv := ev.Object.GetResourceVersion()
	r.counts[ev.Type]++
	last, seen := r.lastRV[k]
	r.hist[k] = append(r.hist[k], string(ev.Type)+"@"+rv+fmt.Sprint(ev.Object.GetLabels()))
	switch ev.Type {
	case cache.Added:
		if r.held[k] != nil {
			r.errs = append(r.errs, fmt.Sprintf("added %s@%s while held at %s", k, rv, r.held[k].GetResourceVersion()))
		}
		if seen && !newer(rv, last) {
			r.errs = append(r.errs, fmt.Sprintf("added %s@%s after an event at %s: %v", k, rv, last, r.hist[k]))
		}
		r.held[k] = ev.Object
	case cache.Updated:
		if r.held[k] == nil {
			r.errs = append(r.errs, fmt.Sprintf("updated %s@%s, not held", k, rv))
		} else if !newer(rv, r.held[k].GetResourceVersion()) {
			r.errs = append(r.errs, fmt.Sprintf("updated %s@%s, held at %s", k, rv, r.held[k].GetResourceVersion()))
		}
		r.held[k] = ev.Object
	case cache.Deleted:
		if r.held[k] == nil {
			r.errs = append(r.errs, fmt.Sprintf("deleted %s@%s, not held", k, rv))
		}
		delete(r.held, k)
	}
	if !seen || newer(rv, last) {
		r.lastRV[k] = rv
	}
}

// state returns the objects rebuilt from the events, as "ns/name@rv", sorted.
func (r *recorder) state() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return versions(func(yield func(*unstructured.Unstructured)) {
		for _, o := range r.held {
			yield(o)
		}
	})
}

func (r *recorder) problems() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.errs)
}

func versions(each func(func(*unstructured.Unstructured))) []string {
	var out []string
	each(func(o *unstructured.Unstructured) {
		out = append(out, o.GetNamespace()+"/"+o.GetName()+"@"+o.GetResourceVersion())
	})
	slices.Sort(out)
	return out
}

func listed(h *cache.Handle) []string {
	return versions(func(yield func(*unstructured.Unstructured)) {
		for _, o := range h.List() {
			yield(o)
		}
	})
}

// truth lists scope on a fresh cache over src: what the server holds now.
func truth(t *testing.T, src cache.Source, scope cache.Scope) []string {
	t.Helper()
	c := cache.New(src, cache.Options{})
	defer c.Close()
	h := c.Acquire(scope)
	defer h.Release()
	waitSync(t, h)
	return listed(h)
}

func stat(c *cache.Cache, scope cache.Scope) (cache.InformerStats, bool) {
	for _, st := range c.Stats() {
		if st.Scope == scope {
			return st, true
		}
	}
	return cache.InformerStats{}, false
}

func waitUntil(t *testing.T, what string, cond func() bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if !testkit.WaitFor(ctx, cond) {
		t.Fatalf("timed out waiting for %s", what)
	}
}

// While the unselected scope runs, a label-selected scope of the same resource, namespace and
// metadata flag reads from it: no list and no watch of its own, the selector applied to the
// objects and to the events, an object whose labels leave the selection arrives as a delete and
// one whose labels enter it as an add. Another namespace or another metadata flag is not covered.
func TestCoveredScopeCostsNoWatch(t *testing.T) {
	src := testkit.NewMemorySource()
	src.Set(testkit.SecretsGVR, testkit.Secret("apps", "db", map[string]string{"app": "db"}, map[string]string{"password": "hunter2"}))
	src.Set(testkit.SecretsGVR, release("apps", "web", 1))
	src.Set(testkit.SecretsGVR, release("ops", "mon", 1))
	c := cache.New(src, cache.Options{WarmPeriod: time.Minute})
	defer c.Close()

	full := c.Acquire(allSecrets)
	defer full.Release()
	waitSync(t, full)
	if !c.Running(helmSecrets) {
		t.Error("a label-selected scope whose cover runs is not reported running")
	}
	if src.Lists() != 1 {
		t.Fatalf("Running started a list: %d", src.Lists())
	}
	helm := c.Acquire(helmSecrets)
	defer helm.Release()
	waitSync(t, helm)
	if got := names(helm.List()); fmt.Sprint(got) != "[sh.helm.release.v1.mon.v1 sh.helm.release.v1.web.v1]" {
		t.Fatalf("covered scope lists %v", got)
	}
	if got := names(helm.ListNamespace("ops")); fmt.Sprint(got) != "[sh.helm.release.v1.mon.v1]" {
		t.Errorf("ListNamespace(ops): %v", got)
	}
	if o, ok := helm.Get("apps", "sh.helm.release.v1.web.v1"); !ok || o.Object["data"] == nil {
		t.Errorf("Get: %v %v", o, ok)
	}
	if _, ok := helm.Get("apps", "db"); ok {
		t.Error("Get found an object outside the selection")
	}
	if src.Lists() != 1 || src.Watches() != 1 {
		t.Errorf("the covered scope cost %d lists and %d watches in all, want 1 and 1", src.Lists(), src.Watches())
	}
	if st, _ := stat(c, helmSecrets); st.CoveredBy != "secrets" || !st.Synced || st.Objects != 2 || st.Handles != 1 {
		t.Errorf("stats %+v", st)
	}
	if got := cache.Runners(c); fmt.Sprint(got) != "[secrets]" {
		t.Errorf("informers %v", got)
	}

	rec, cancel := record(helm)
	defer cancel()
	src.Set(testkit.SecretsGVR, release("apps", "web", 2))                                                                 // added
	src.Set(testkit.SecretsGVR, testkit.Secret("apps", "other", map[string]string{"app": "x"}, nil))                       // not selected
	src.Set(testkit.SecretsGVR, testkit.Secret("apps", "sh.helm.release.v1.web.v1", map[string]string{"owner": "x"}, nil)) // leaves
	src.Set(testkit.SecretsGVR, testkit.Secret("apps", "other", map[string]string{"owner": "helm"}, nil))                  // enters
	src.Set(testkit.SecretsGVR, release("ops", "mon", 1))                                                                  // updated
	src.Delete(testkit.SecretsGVR, "apps", "other")                                                                        // deleted
	want := []string{"apps/sh.helm.release.v1.web.v2", "ops/sh.helm.release.v1.mon.v1"}
	waitUntil(t, "the events", func() bool {
		rec.mu.Lock()
		defer rec.mu.Unlock()
		return rec.counts[cache.Added] == 2 && rec.counts[cache.Deleted] == 2 && rec.counts[cache.Updated] == 1
	})
	if p := rec.problems(); len(p) > 0 {
		t.Errorf("events: %v", p)
	}
	if got := rec.state(); len(got) != 2 || !strings.HasPrefix(got[0], want[0]+"@") || !strings.HasPrefix(got[1], want[1]+"@") {
		t.Errorf("rebuilt from events: %v", got)
	}
	if fmt.Sprint(rec.state()) != fmt.Sprint(listed(helm)) || fmt.Sprint(listed(helm)) != fmt.Sprint(truth(t, src, helmSecrets)) {
		t.Errorf("events %v, list %v, server %v", rec.state(), listed(helm), truth(t, src, helmSecrets))
	}

	// Not covered: another metadata flag, another namespace.
	lists := src.Lists()
	for _, scope := range []cache.Scope{
		{GVR: testkit.SecretsGVR, LabelSelector: "owner=helm", MetadataOnly: true},
		{GVR: testkit.SecretsGVR, LabelSelector: "owner=helm", Namespace: "apps"},
	} {
		h := c.Acquire(scope)
		waitSync(t, h)
		if st, _ := stat(c, scope); st.CoveredBy != "" {
			t.Errorf("%s covered by %s", scope, st.CoveredBy)
		}
		h.Release()
	}
	if src.Lists() != lists+2 {
		t.Errorf("uncovered scopes: %d lists, want 2", src.Lists()-lists)
	}
}

// A label-selected scope with its own informer moves onto a cover that starts later, once the
// cover has synced: its own informer and watch stop, and its subscribers see no event twice and
// miss none.
func TestScopeMovesOntoACoverThatStartsLater(t *testing.T) {
	src := testkit.NewMemorySource()
	for i := range 5 {
		src.Set(testkit.SecretsGVR, release("apps", "r"+strconv.Itoa(i), 1))
		src.Set(testkit.SecretsGVR, testkit.Secret("apps", "plain-"+strconv.Itoa(i), nil, map[string]string{"k": "v"}))
	}
	c := cache.New(src, cache.Options{WarmPeriod: time.Minute})
	defer c.Close()
	helm := c.Acquire(helmSecrets)
	defer helm.Release()
	waitSync(t, helm)
	if st, _ := stat(c, helmSecrets); st.CoveredBy != "" || st.Objects != 5 {
		t.Fatalf("own informer: %+v", st)
	}
	rec, cancel := record(helm)
	defer cancel()

	full := c.Acquire(allSecrets)
	defer full.Release()
	waitUntil(t, "the move onto the cover", func() bool {
		st, _ := stat(c, helmSecrets)
		return st.CoveredBy == "secrets" && fmt.Sprint(cache.Runners(c)) == "[secrets]"
	})
	waitUntil(t, "the own watch to close", func() bool { return src.OpenWatches(testkit.SecretsGVR) == 1 })
	src.Set(testkit.SecretsGVR, release("apps", "r0", 2))
	src.Delete(testkit.SecretsGVR, "apps", "sh.helm.release.v1.r1.v1")
	lists := src.Lists()
	want := truth(t, src, helmSecrets)
	waitUntil(t, "the changes", func() bool { return fmt.Sprint(rec.state()) == fmt.Sprint(want) })
	if p := rec.problems(); len(p) > 0 {
		t.Errorf("events: %v", p)
	}
	if fmt.Sprint(listed(helm)) != fmt.Sprint(want) {
		t.Errorf("list %v, want %v", listed(helm), want)
	}
	if lists != 2 {
		t.Errorf("%d lists, want 2 (the own informer's, then the cover's)", lists)
	}
}

// A covered scope never keeps its cover alive: when the cover's warm period ends while covered
// handles are open, the scope starts its own informer, and the cover's stops once that one has
// synced; the subscribers miss nothing and see nothing twice.
func TestCoverExpiryHandsOverToOwnInformer(t *testing.T) {
	src := testkit.NewMemorySource()
	src.Set(testkit.SecretsGVR, release("apps", "web", 1))
	src.Set(testkit.SecretsGVR, testkit.Secret("apps", "db", nil, map[string]string{"password": "hunter2"}))
	c := cache.New(src, cache.Options{WarmPeriod: 50 * time.Millisecond})
	defer c.Close()
	full := c.Acquire(allSecrets)
	waitSync(t, full)
	helm := c.Acquire(helmSecrets)
	defer helm.Release()
	waitSync(t, helm)
	rec, cancel := record(helm)
	defer cancel()

	full.Release()
	waitUntil(t, "the handover to the own informer", func() bool {
		st, ok := stat(c, helmSecrets)
		return ok && st.CoveredBy == "" && st.Synced && fmt.Sprint(cache.Runners(c)) == "[secrets with owner=helm]"
	})
	if c.Running(allSecrets) {
		t.Error("the cover still runs")
	}
	waitUntil(t, "the cover's watch to close", func() bool { return src.OpenWatches(testkit.SecretsGVR) == 1 })
	if _, ok := stat(c, allSecrets); ok {
		t.Error("the cover is still listed")
	}
	src.Set(testkit.SecretsGVR, release("apps", "web", 2))
	src.Set(testkit.SecretsGVR, testkit.Secret("apps", "db", map[string]string{"owner": "helm"}, nil))
	lists := src.Lists()
	want := truth(t, src, helmSecrets)
	waitUntil(t, "the changes through the own informer", func() bool { return fmt.Sprint(rec.state()) == fmt.Sprint(want) })
	if p := rec.problems(); len(p) > 0 {
		t.Errorf("events: %v", p)
	}
	if fmt.Sprint(listed(helm)) != fmt.Sprint(want) {
		t.Errorf("list %v, want %v", listed(helm), want)
	}
	if lists != 2 {
		t.Errorf("%d lists, want 2 (the cover's, then the own informer's)", lists)
	}
}

// A covered scope that is only warm (no handle) ends with its cover: it starts no informer of its
// own just to stay warm.
func TestWarmCoveredScopeEndsWithItsCover(t *testing.T) {
	src := testkit.NewMemorySource()
	src.Set(testkit.SecretsGVR, release("apps", "web", 1))
	c := cache.New(src, cache.Options{WarmPeriod: 50 * time.Millisecond})
	defer c.Close()
	full := c.Acquire(allSecrets)
	waitSync(t, full)
	helm := c.Acquire(helmSecrets)
	waitSync(t, helm)
	helm.Release()
	full.Release()
	waitUntil(t, "both to end", func() bool { return len(c.Stats()) == 0 && len(cache.Runners(c)) == 0 })
	if src.Lists() != 1 {
		t.Errorf("%d lists, want 1", src.Lists())
	}
	if c.Running(helmSecrets) {
		t.Error("still running")
	}
}

// selectiveGate holds the lists of label-selected scopes until it opens.
type selectiveGate struct {
	*testkit.MemorySource
	open chan struct{}
}

func (g *selectiveGate) ListerWatcher(scope cache.Scope) toolscache.ListerWatcher {
	if scope.LabelSelector == "" {
		return g.MemorySource.ListerWatcher(scope)
	}
	return &gatedLW{gate: g.open, inner: g.MemorySource.ListerWatcher(scope)}
}

// The cover stays only until the scope's own informer has synced, and never longer than the
// handover timeout: an own informer that cannot list does not keep every Secret in memory.
func TestCoverStopsAfterTheHandoverTimeout(t *testing.T) {
	src := &selectiveGate{MemorySource: testkit.NewMemorySource(), open: make(chan struct{})}
	src.Set(testkit.SecretsGVR, release("apps", "web", 1))
	c := cache.New(src, cache.Options{WarmPeriod: 20 * time.Millisecond, HandoverTimeout: 300 * time.Millisecond})
	defer func() {
		close(src.open)
		c.Close()
	}()
	full := c.Acquire(allSecrets)
	waitSync(t, full)
	helm := c.Acquire(helmSecrets)
	defer helm.Release()
	waitSync(t, helm)
	full.Release()
	waitUntil(t, "the own informer to start", func() bool {
		return fmt.Sprint(cache.Runners(c)) == "[secrets secrets with owner=helm]"
	})
	if st, _ := stat(c, helmSecrets); st.CoveredBy != "secrets" || len(helm.List()) != 1 {
		t.Errorf("while the own informer lists, the cover serves: %+v", st)
	}
	start := time.Now()
	waitUntil(t, "the cover to stop", func() bool { return fmt.Sprint(cache.Runners(c)) == "[secrets with owner=helm]" })
	if d := time.Since(start); d > 5*time.Second {
		t.Errorf("the cover stopped after %v", d)
	}
	if helm.HasSynced() {
		t.Error("synced although its own informer never listed")
	}
	if len(helm.List()) != 0 {
		t.Errorf("after the timeout the scope shows what its own informer holds (nothing yet): %v", names(helm.List()))
	}
}

// Many rounds of a cover starting and stopping under churn, with the selected scope's handle
// open throughout: every round its subscriber's events stay exact (no event twice, none missing,
// never an older version) and List agrees with them and with the server. Run it with -race.
func TestCoverHandoversUnderChurn(t *testing.T) {
	gate := make(chan struct{})
	src := &gatedSource{MemorySource: testkit.NewMemorySource(), gate: gate}
	for i := range 20 {
		src.Set(testkit.SecretsGVR, release("apps", "r"+strconv.Itoa(i), 1))
	}
	c := cache.New(src, cache.Options{WarmPeriod: 5 * time.Millisecond, HandoverTimeout: 2 * time.Second})
	defer c.Close()
	helm := c.Acquire(helmSecrets)
	defer helm.Release()
	rec, cancel := record(helm) // before the first list: every object arrives as an event
	defer cancel()
	close(gate)
	waitSync(t, helm)

	stop := make(chan struct{})
	var churn sync.WaitGroup
	var changes atomic.Int64
	churn.Go(func() {
		rng := rand.New(rand.NewPCG(1, 2))
		for {
			select {
			case <-stop:
				return
			default:
			}
			name := "r" + strconv.Itoa(rng.IntN(30))
			obj := "sh.helm.release.v1." + name + ".v1"
			switch rng.IntN(5) {
			case 0:
				src.Delete(testkit.SecretsGVR, "apps", obj)
			case 1: // the labels leave the selection
				src.Set(testkit.SecretsGVR, testkit.Secret("apps", obj, map[string]string{"owner": "someone"}, nil))
			default:
				src.Set(testkit.SecretsGVR, release("apps", name, 1))
			}
			changes.Add(1)
			time.Sleep(time.Duration(rng.IntN(300)) * time.Microsecond)
		}
	})
	for round := range 15 {
		full := c.Acquire(allSecrets)
		waitUntil(t, "round "+strconv.Itoa(round)+": covered", func() bool {
			st, _ := stat(c, helmSecrets)
			return st.CoveredBy == "secrets"
		})
		time.Sleep(time.Duration(round%4) * time.Millisecond)
		full.Release()
		waitUntil(t, "round "+strconv.Itoa(round)+": own informer", func() bool {
			st, _ := stat(c, helmSecrets)
			return st.CoveredBy == "" && st.Synced && !c.Running(allSecrets)
		})
	}
	close(stop)
	churn.Wait()

	want := truth(t, src, helmSecrets)
	waitUntil(t, "the events to settle", func() bool {
		return fmt.Sprint(rec.state()) == fmt.Sprint(want) && fmt.Sprint(listed(helm)) == fmt.Sprint(want)
	})
	if p := rec.problems(); len(p) > 0 {
		t.Fatalf("%d problems in the events, first: %v", len(p), p[:min(5, len(p))])
	}
	if changes.Load() < 50 {
		t.Logf("only %d changes during the handovers", changes.Load())
	}
}

// Many goroutines acquire and release covers and covered scopes at once while the source
// changes: no race, no deadlock, and every informer stops once nothing holds it.
func TestConcurrentCoversAndCoveredScopes(t *testing.T) {
	src := testkit.NewMemorySource()
	c := cache.New(src, cache.Options{WarmPeriod: 10 * time.Millisecond, HandoverTimeout: time.Second})
	scopes := []cache.Scope{
		allSecrets, helmSecrets,
		{GVR: testkit.SecretsGVR, LabelSelector: "name=web"},
		{GVR: testkit.SecretsGVR, LabelSelector: "owner=helm,name=web"},
	}
	stop := make(chan struct{})
	var writer sync.WaitGroup
	writer.Go(func() {
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}
			src.Set(testkit.SecretsGVR, release("apps", []string{"web", "db", "mon"}[i%3], i%7))
			if i%5 == 0 {
				src.Delete(testkit.SecretsGVR, "apps", "sh.helm.release.v1.web.v"+strconv.Itoa(i%7))
			}
			time.Sleep(200 * time.Microsecond)
		}
	})
	var workers sync.WaitGroup
	for w := range 12 {
		workers.Go(func() {
			for i := range 40 {
				h := c.Acquire(scopes[(w+i)%len(scopes)])
				unsub := h.Subscribe(func(ev cache.Event) { _ = ev.Object.GetName() })
				_ = h.List()
				_ = h.ListNamespace("apps")
				_, _ = h.Get("apps", "sh.helm.release.v1.web.v1")
				_ = h.Err()
				_ = c.Stats()
				_ = c.Running(scopes[i%len(scopes)])
				if i%5 == 0 {
					ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
					h.WaitForSync(ctx)
					cancel()
				}
				unsub()
				h.Release()
				if i%8 == 0 {
					time.Sleep(15 * time.Millisecond)
				}
			}
		})
	}
	workers.Wait()
	close(stop)
	writer.Wait()
	waitUntil(t, "every informer to stop", func() bool { return len(c.Stats()) == 0 && len(cache.Runners(c)) == 0 })
	c.Close()
}
