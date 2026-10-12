package metrics

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/guided-traffic/cogpit/internal/model"
)

// Kind is what a scope measures.
type Kind string

// Kinds.
const (
	KindPods  Kind = "pods"
	KindNodes Kind = "nodes"
)

// Scope names what a poll asks for. Everything that wants the same scope shares one poll.
//
//	{pods, "", ""}      every pod of every namespace (a list view)
//	{pods, "web", ""}   the pods of one namespace
//	{pods, "web", "a"}  one pod (a drawer)
//	{nodes, "", ""}     every node; {nodes, "", "n1"} one node
type Scope struct {
	Kind      Kind
	Namespace string
	Name      string
}

// IsList reports whether the scope is a list.
func (s Scope) IsList() bool { return s.Name == "" }

// Defaults of Options.
const (
	// DefaultListInterval is how often a visible list is polled: Lens's list refresh, 60 seconds.
	// A list can hold thousands of pods, so the response is large; the Metrics Server measures
	// every 15 seconds anyway, and a column of current values does not need every measurement.
	DefaultListInterval = 60 * time.Second
	// DefaultObjectInterval is how often an open drawer polls its object: 15 seconds, the Metrics
	// Server's own resolution (its --metric-resolution default), so every measurement reaches the
	// chart and none is asked for twice. The response is one small object.
	DefaultObjectInterval = 15 * time.Second
	// DefaultTTL is how long a result is served as current when nobody polls the scope: the
	// Metrics Server's resolution again. A subscription that starts within the TTL of the last
	// result does not poll at once; a one-off read (the agents' command) within it asks nothing.
	DefaultTTL = 15 * time.Second
	// DefaultSeriesPoints bounds one series: an hour at the object interval.
	DefaultSeriesPoints = 240
	// DefaultSeriesAge bounds a series in time.
	DefaultSeriesAge = time.Hour
	// DefaultDetectTTL is how long a found provider is trusted before discovery asks again.
	DefaultDetectTTL = 10 * time.Minute
	// DefaultRetryDetectTTL is how long a "not served", "forbidden" or "unavailable" is kept
	// before discovery asks again: a metrics-server installed meanwhile shows up within it.
	DefaultRetryDetectTTL = 2 * time.Minute
	// DefaultRequestTimeout bounds one poll.
	DefaultRequestTimeout = 20 * time.Second
	// DefaultRangeWindow is the time range of a chart of a provider with history: Lens's default,
	// one hour.
	DefaultRangeWindow = time.Hour
	// DefaultMinStep is the finest step of a range query: a Prometheus rarely scrapes faster.
	DefaultMinStep = 15 * time.Second
	// StepsPerWindow is how many points a chart gets across its time range (window / 120): a
	// chart a few hundred pixels wide gains nothing from more.
	StepsPerWindow = 120
	// DefaultMaxPoints bounds the points of one series however long the longest range is.
	DefaultMaxPoints = 1000
)

// Options tune a Hub; zero fields take the defaults above. Tests shrink the intervals.
type Options struct {
	ListInterval   time.Duration
	ObjectInterval time.Duration
	TTL            time.Duration
	SeriesPoints   int
	SeriesAge      time.Duration
	DetectTTL      time.Duration
	RetryDetectTTL time.Duration
	RequestTimeout time.Duration
	// RangeWindow is the time range a drawer's charts show unless a window was set with
	// Subscription.SetWindows; MinStep the finest step of a range query; MaxRangePoints the most
	// points a range query returns per series.
	RangeWindow    time.Duration
	MinStep        time.Duration
	MaxRangePoints int
	// Now is the clock; nil means time.Now.
	Now func() time.Time
	// Logger receives poll failures; nil means slog.Default().
	Logger *slog.Logger
}

func (o Options) withDefaults() Options {
	def := func(v *time.Duration, d time.Duration) {
		if *v <= 0 {
			*v = d
		}
	}
	def(&o.ListInterval, DefaultListInterval)
	def(&o.ObjectInterval, DefaultObjectInterval)
	def(&o.TTL, DefaultTTL)
	def(&o.SeriesAge, DefaultSeriesAge)
	def(&o.DetectTTL, DefaultDetectTTL)
	def(&o.RetryDetectTTL, DefaultRetryDetectTTL)
	def(&o.RequestTimeout, DefaultRequestTimeout)
	def(&o.RangeWindow, DefaultRangeWindow)
	def(&o.MinStep, DefaultMinStep)
	if o.MaxRangePoints <= 0 {
		o.MaxRangePoints = DefaultMaxPoints
	}
	if o.SeriesPoints <= 0 {
		o.SeriesPoints = DefaultSeriesPoints
	}
	if o.Now == nil {
		o.Now = time.Now
	}
	if o.Logger == nil {
		o.Logger = slog.Default()
	}
	return o
}

// Hub polls a cluster's provider for the scopes that are subscribed and keeps the latest results.
//
//   - Nothing is asked while nothing is subscribed: no poll, no detection.
//   - A scope is polled by one goroutine however many consumers subscribed to it; its interval is
//     the list or the object interval.
//   - A result is cached for the TTL, longer while the scope is polled (twice its interval, so a
//     poll that is late does not blank a column). A subscription that starts on a fresh result
//     waits for it to go stale.
//   - While an object scope is subscribed the hub keeps a bounded time series of the object (and
//     of each of a pod's containers), seeded from the list result if one is fresh. The series ends
//     with the last subscription.
//
// A Hub is safe for concurrent use.
type Hub struct {
	det  Detector
	opts Options

	detMu      sync.Mutex // serialises detection
	mu         sync.Mutex // guards everything below
	prov       Provider
	state      State
	detectedAt time.Time
	scopes     map[Scope]*scopeState
	closed     bool

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	gen atomic.Uint64
}

// NewHub returns a hub that asks det for the provider the first time something subscribes or
// reads.
func NewHub(det Detector, opts Options) *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	return &Hub{det: det, opts: opts.withDefaults(), scopes: map[Scope]*scopeState{}, ctx: ctx, cancel: cancel}
}

// Generation increases whenever a result or a state changed; a table snapshot built at an older
// generation may be stale.
func (h *Hub) Generation() uint64 { return h.gen.Load() }

// Close stops every poll and waits for them. Subscriptions made afterwards do nothing.
func (h *Hub) Close() {
	h.mu.Lock()
	h.closed = true
	h.mu.Unlock()
	h.cancel()
	h.wg.Wait()
}

type scopeState struct {
	scope     Scope
	subs      map[*Subscription]struct{}
	cancel    context.CancelFunc // set while the poll goroutine runs
	data      *dataSet
	state     State // the outcome of the last attempt
	attempted time.Time
	inflight  *flight
	series    map[string]*seriesBuf
	touched   time.Time     // the last use, for pruning
	kick      chan struct{} // wakes the poll goroutine early: a chart's range changed
}

// dataSet is one result: immutable once stored.
type dataSet struct {
	at    time.Time // when the hub got it
	pods  map[string]model.PodUsage
	nodes map[string]model.NodeUsage
	// series is the history of a provider that has some (Capabilities.History), over window at
	// step; nil for instant results.
	series       *ObjectSeries
	window, step time.Duration
}

type flight struct {
	done chan struct{}
}

func podKey(namespace, name string) string { return namespace + "/" + name }

// Subscription is one consumer's interest in a scope.
type Subscription struct {
	h        *Hub
	scope    Scope
	onChange func()
	once     sync.Once
	windows  []time.Duration // the time ranges its charts show (guarded by Hub.mu); empty: the default
}

// Subscribe starts polling scope (if nobody polls it yet) until the returned subscription is
// released. onChange, if set, is called after every poll of the scope, on the poll goroutine: it
// must not block. The first poll happens at once unless a result younger than the TTL is held.
func (h *Hub) Subscribe(scope Scope, onChange func()) *Subscription {
	sub := &Subscription{h: h, scope: scope, onChange: onChange}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return sub
	}
	st := h.scopeLocked(scope)
	st.subs[sub] = struct{}{}
	if st.cancel == nil {
		if !scope.IsList() {
			h.seedLocked(st)
		}
		ctx, cancel := context.WithCancel(h.ctx)
		st.cancel = cancel
		h.wg.Add(1)
		go h.loop(ctx, scope, st.kick, h.firstWaitLocked(st))
	}
	return sub
}

// Release ends the subscription (a nil subscription is a no-op); the last one of a scope stops its poll and drops its series. The
// result stays cached for the TTL. Releasing twice is harmless.
func (s *Subscription) Release() {
	if s == nil {
		return
	}
	s.once.Do(func() {
		h := s.h
		h.mu.Lock()
		defer h.mu.Unlock()
		st := h.scopes[s.scope]
		if st == nil {
			return
		}
		delete(st.subs, s)
		if len(st.subs) > 0 {
			return
		}
		if st.cancel != nil {
			st.cancel()
			st.cancel = nil
		}
		st.series = nil
		st.touched = h.opts.Now()
		h.pruneLocked()
	})
}

func (h *Hub) intervalOf(s Scope) time.Duration {
	if s.IsList() {
		return h.opts.ListInterval
	}
	return h.opts.ObjectInterval
}

// intervalLocked is how often a scope is polled: the list or object interval, and for a drawer on
// a provider with history the range query's step (a new point exists only once per step).
func (h *Hub) intervalLocked(st *scopeState) time.Duration {
	iv := h.intervalOf(st.scope)
	if !st.scope.IsList() && h.rangeProviderLocked() != nil {
		_, step := h.rangeOfLocked(st)
		iv = max(iv, step)
	}
	return iv
}

// rangeProviderLocked returns the detected provider if it has history.
func (h *Hub) rangeProviderLocked() RangeProvider {
	if rp, ok := h.prov.(RangeProvider); ok && h.prov.Capabilities().History {
		return rp
	}
	return nil
}

// rangeOfLocked is the time range one poll of an object scope asks for and its step. The window is
// the longest any subscribed chart shows (the default one hour is always among them, so a chart
// that was not told otherwise has data), the step follows the shortest: window / 120, at least
// MinStep, and coarse enough that the longest window stays within MaxRangePoints.
func (h *Hub) rangeOfLocked(st *scopeState) (window, step time.Duration) {
	window = h.opts.RangeWindow
	shortest := window
	for s := range st.subs {
		for _, w := range s.windows {
			if w <= 0 {
				continue
			}
			window = max(window, w)
			shortest = min(shortest, w)
		}
	}
	step = max(h.opts.MinStep, shortest/StepsPerWindow, window/time.Duration(h.opts.MaxRangePoints))
	if step >= time.Second {
		step = (step + time.Second - 1) / time.Second * time.Second
	}
	return window, step
}

// SetWindows tells the hub which time ranges the subscriber's charts show; with a provider that
// has history (Prometheus) the next poll asks for the longest, at the step of the shortest, and
// one poll serves every chart of the scope. A subscription that never calls it shows the default
// range. A change makes the scope poll at once. Providers without history ignore it.
func (s *Subscription) SetWindows(windows ...time.Duration) {
	if s == nil {
		return
	}
	h := s.h
	h.mu.Lock()
	defer h.mu.Unlock()
	if slices.Equal(s.windows, windows) {
		return
	}
	s.windows = slices.Clone(windows)
	if st := h.scopes[s.scope]; st != nil && st.cancel != nil && !s.scope.IsList() && h.rangeProviderLocked() != nil {
		select {
		case st.kick <- struct{}{}:
		default:
		}
	}
}

// Capabilities are the detected provider's: which chart variants it can draw. Zero until a
// provider was found.
func (h *Hub) Capabilities() Capabilities {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.prov == nil {
		return Capabilities{}
	}
	return h.prov.Capabilities()
}

// DefaultWindow is the time range a chart shows unless told otherwise.
func (h *Hub) DefaultWindow() time.Duration { return h.opts.RangeWindow }

// Forget makes the next poll detect the provider again: the user changed the cluster's metrics
// settings (the Prometheus override), or knows a provider was installed.
func (h *Hub) Forget() { h.forgetDetection() }

func (h *Hub) scopeLocked(s Scope) *scopeState {
	st := h.scopes[s]
	if st == nil {
		st = &scopeState{scope: s, subs: map[*Subscription]struct{}{}, kick: make(chan struct{}, 1)}
		h.scopes[s] = st
	}
	st.touched = h.opts.Now()
	return st
}

// pruneLocked forgets results nobody polls and nobody can still use.
func (h *Hub) pruneLocked() {
	now := h.opts.Now()
	for k, st := range h.scopes {
		if len(st.subs) == 0 && st.inflight == nil && now.Sub(st.touched) > 4*h.opts.TTL {
			delete(h.scopes, k)
		}
	}
}

// firstWaitLocked is how long a new poll goroutine waits before its first poll: until the held
// result goes stale, or not at all when there is none.
func (h *Hub) firstWaitLocked(st *scopeState) time.Duration {
	if st.data == nil {
		return 0
	}
	if st.data.series != nil {
		if w, step := h.rangeOfLocked(st); w != st.data.window || step != st.data.step {
			return 0 // the held history was asked for another range
		}
	}
	return max(h.opts.TTL-h.opts.Now().Sub(st.data.at), 0)
}

// seedLocked starts an object's series with the measurement a fresh list result holds for it.
func (h *Hub) seedLocked(st *scopeState) {
	sc := st.scope
	switch sc.Kind {
	case KindPods:
		for _, c := range []Scope{{KindPods, "", ""}, {KindPods, sc.Namespace, ""}} {
			if ls := h.scopes[c]; ls != nil && h.validLocked(ls) {
				if p, ok := ls.data.pods[podKey(sc.Namespace, sc.Name)]; ok {
					h.addPodSeriesLocked(st, p)
					return
				}
			}
		}
	case KindNodes:
		if ls := h.scopes[Scope{Kind: KindNodes}]; ls != nil && h.validLocked(ls) {
			if n, ok := ls.data.nodes[sc.Name]; ok {
				h.addNodeSeriesLocked(st, n)
			}
		}
	}
}

// validLocked reports whether a scope's result may still be served as current.
func (h *Hub) validLocked(st *scopeState) bool {
	if st.data == nil {
		return false
	}
	limit := h.opts.TTL
	if st.cancel != nil {
		limit = max(limit, 2*h.intervalLocked(st))
	}
	return h.opts.Now().Sub(st.data.at) <= limit
}

// loop is a scope's poll goroutine.
func (h *Hub) loop(ctx context.Context, sc Scope, kick <-chan struct{}, first time.Duration) {
	defer h.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			h.opts.Logger.Error("metrics: a poll panicked", "scope", fmt.Sprintf("%+v", sc), "panic", fmt.Sprint(r), "stack", string(debug.Stack()))
		}
	}()
	timer := time.NewTimer(first)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		case <-kick:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
		}
		pctx, cancel := context.WithTimeout(ctx, h.opts.RequestTimeout)
		_ = h.fetch(pctx, sc)
		cancel()
		if ctx.Err() != nil {
			return
		}
		h.notify(sc)
		timer.Reset(h.pollInterval(sc))
	}
}

// pollInterval is how long a scope waits after a poll.
func (h *Hub) pollInterval(sc Scope) time.Duration {
	h.mu.Lock()
	defer h.mu.Unlock()
	if st := h.scopes[sc]; st != nil {
		return h.intervalLocked(st)
	}
	return h.intervalOf(sc)
}

// notify calls the subscribers of a scope.
func (h *Hub) notify(sc Scope) {
	h.mu.Lock()
	var cbs []func()
	if st := h.scopes[sc]; st != nil {
		for s := range st.subs {
			if s.onChange != nil {
				cbs = append(cbs, s.onChange)
			}
		}
	}
	h.mu.Unlock()
	for _, cb := range cbs {
		func() {
			defer func() { _ = recover() }()
			cb()
		}()
	}
}

// provider returns the cluster's provider, asking the detector when its answer is older than the
// detection TTL (shorter while there is no provider).
func (h *Hub) provider(ctx context.Context) (Provider, State) {
	h.detMu.Lock()
	defer h.detMu.Unlock()
	h.mu.Lock()
	ttl := h.opts.DetectTTL
	if h.prov == nil {
		ttl = h.opts.RetryDetectTTL
	}
	if !h.detectedAt.IsZero() && h.opts.Now().Sub(h.detectedAt) < ttl {
		p, s := h.prov, h.state
		h.mu.Unlock()
		return p, s
	}
	h.mu.Unlock()

	var (
		p Provider
		s State
	)
	func() {
		defer func() {
			if r := recover(); r != nil {
				p, s = nil, State{Status: StatusUnavailable, Message: fmt.Sprint("detecting metrics failed: ", r)}
			}
		}()
		p, s = h.det.Detect(ctx)
	}()
	if ctx.Err() != nil { // cancelled: say nothing about the cluster
		return nil, State{Status: StatusUnavailable, Message: msgCancelled}
	}
	h.mu.Lock()
	h.prov, h.state, h.detectedAt = p, s, h.opts.Now()
	h.mu.Unlock()
	h.gen.Add(1)
	return p, s
}

// forgetDetection makes the next poll detect again: the API went away.
func (h *Hub) forgetDetection() {
	h.mu.Lock()
	h.detectedAt = time.Time{}
	h.mu.Unlock()
}

// fetch asks the provider for a scope once and stores the outcome. Concurrent fetches of one scope
// share one request. It returns the outcome's state error-free: failures are states.
func (h *Hub) fetch(ctx context.Context, sc Scope) State {
	h.mu.Lock()
	st := h.scopeLocked(sc)
	if f := st.inflight; f != nil {
		h.mu.Unlock()
		select {
		case <-f.done:
		case <-ctx.Done():
		}
		h.mu.Lock()
		defer h.mu.Unlock()
		return st.state
	}
	f := &flight{done: make(chan struct{})}
	st.inflight = f
	h.mu.Unlock()

	data, state := h.query(ctx, sc)

	h.mu.Lock()
	defer h.mu.Unlock()
	st.inflight = nil
	close(f.done)
	if ctx.Err() != nil && data == nil {
		return st.state // cancelled: keep what was known
	}
	st.attempted, st.state = h.opts.Now(), state
	if data != nil {
		st.data = data
		if st.cancel != nil && !sc.IsList() && data.series == nil { // series only while the object is watched
			h.recordLocked(st, data)
		}
	}
	h.gen.Add(1)
	h.pruneLocked()
	return state
}

// query makes the request of a scope. A failure is a state and, for an error that may be
// transient, leaves the previous result alone (data nil).
func (h *Hub) query(ctx context.Context, sc Scope) (*dataSet, State) {
	prov, state := h.provider(ctx)
	if prov == nil {
		return nil, state
	}
	ok := State{Status: StatusAvailable, Provider: prov.ID(), Label: prov.Label()}
	if rp, window, step := h.historyFor(prov, sc); rp != nil {
		return h.queryHistory(ctx, rp, sc, window, step, ok)
	}
	ds := &dataSet{at: h.opts.Now()}
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("provider panicked: %v", r)
			}
		}()
		switch sc.Kind {
		case KindPods:
			ds.pods = map[string]model.PodUsage{}
			if sc.IsList() {
				var l []model.PodUsage
				if l, err = prov.ListPods(ctx, sc.Namespace); err == nil {
					for _, p := range l {
						ds.pods[podKey(p.Namespace, p.Name)] = p
					}
				}
			} else {
				var p model.PodUsage
				if p, err = prov.GetPod(ctx, sc.Namespace, sc.Name); err == nil {
					if p.Namespace == "" {
						p.Namespace = sc.Namespace
					}
					ds.pods[podKey(p.Namespace, p.Name)] = p
				}
			}
		case KindNodes:
			ds.nodes = map[string]model.NodeUsage{}
			if sc.IsList() {
				var l []model.NodeUsage
				if l, err = prov.ListNodes(ctx); err == nil {
					for _, n := range l {
						ds.nodes[n.Name] = n
					}
				}
			} else {
				var n model.NodeUsage
				if n, err = prov.GetNode(ctx, sc.Name); err == nil {
					ds.nodes[n.Name] = n
				}
			}
		default:
			err = fmt.Errorf("unknown scope kind %q", sc.Kind)
		}
	}()
	switch {
	case err == nil:
		return ds, ok
	case errors.Is(err, ErrNoData):
		return ds, ok // an empty result: the object has no measurement yet
	}
	return nil, h.failure(ctx, sc, err)
}

// failure turns a provider's error into the scope's state. An error that carries its state
// (StateError) is taken as it is; a not-served state makes detection run again.
func (h *Hub) failure(ctx context.Context, sc Scope, err error) State {
	var s State
	var se *StateError
	if errors.As(err, &se) {
		s = se.State
	} else {
		s = StateFromError(err, "the cluster serves no metrics.k8s.io API (no metrics-server installed)")
	}
	if s.Status == StatusNotServed {
		h.forgetDetection()
	}
	if s.Status != StatusAvailable && ctx.Err() == nil {
		h.opts.Logger.Debug("metrics: a poll failed", "kind", sc.Kind, "namespace", sc.Namespace, "name", sc.Name, "status", s.Status)
	}
	return s
}

// historyFor returns the provider's range interface and the range to ask for when the scope is a
// watched object and the provider has history; nil otherwise (lists and one-off reads use the
// instant path).
func (h *Hub) historyFor(prov Provider, sc Scope) (RangeProvider, time.Duration, time.Duration) {
	rp, ok := prov.(RangeProvider)
	if !ok || !prov.Capabilities().History || sc.IsList() {
		return nil, 0, 0
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	st := h.scopes[sc]
	if st == nil || st.cancel == nil {
		return nil, 0, 0
	}
	window, step := h.rangeOfLocked(st)
	return rp, window, step
}

// alignDown rounds t down to a multiple of step since the Unix epoch, so that the grid of
// successive range queries is the same and a point does not move between polls.
func alignDown(t time.Time, step time.Duration) time.Time {
	if step < time.Second {
		return t
	}
	s := int64(step / time.Second)
	return time.Unix(t.Unix()/s*s, 0)
}

// queryHistory asks a provider with history for an object's series over the window.
func (h *Hub) queryHistory(ctx context.Context, rp RangeProvider, sc Scope, window, step time.Duration, ok State) (*dataSet, State) {
	now := h.opts.Now()
	q := RangeQuery{Kind: sc.Kind, Namespace: sc.Namespace, Name: sc.Name, From: alignDown(now.Add(-window), step), To: now, Step: step}
	var (
		series *ObjectSeries
		err    error
	)
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("provider panicked: %v", r)
			}
		}()
		series, err = rp.ObjectRange(ctx, q)
	}()
	if err != nil {
		return nil, h.failure(ctx, sc, err)
	}
	ds := &dataSet{at: now, series: series, window: window, step: step}
	switch sc.Kind {
	case KindPods:
		ds.pods = map[string]model.PodUsage{}
		if p, found := latestPod(sc, series); found {
			ds.pods[podKey(sc.Namespace, sc.Name)] = p
		}
	case KindNodes:
		ds.nodes = map[string]model.NodeUsage{}
		if t := series.Targets[""]; t != nil && len(t.Usage) > 0 {
			last := t.Usage[len(t.Usage)-1]
			ds.nodes[sc.Name] = model.NodeUsage{Name: sc.Name, Timestamp: last.At, Usage: last.Usage}
		}
	}
	return ds, ok
}

// latestPod is the newest sample of a pod's series as a measurement.
func latestPod(sc Scope, s *ObjectSeries) (model.PodUsage, bool) {
	whole := s.Targets[""]
	if whole == nil || len(whole.Usage) == 0 {
		return model.PodUsage{}, false
	}
	p := model.PodUsage{Namespace: sc.Namespace, Name: sc.Name, Timestamp: whole.Usage[len(whole.Usage)-1].At}
	names := make([]string, 0, len(s.Targets))
	for n := range s.Targets {
		if n != "" {
			names = append(names, n)
		}
	}
	slices.Sort(names)
	for _, n := range names {
		if u := s.Targets[n].Usage; len(u) > 0 {
			p.Containers = append(p.Containers, model.ContainerUsage{Name: n, Usage: u[len(u)-1].Usage})
		}
	}
	return p, true
}

// History returns the series a provider with history gave for an object scope that is polled,
// as of the last poll, or nil: lists, a scope nobody watches, a provider without history, a result
// that went stale. Without history the hub's own bounded Series applies.
func (h *Hub) History(sc Scope) *ObjectSeries {
	h.mu.Lock()
	defer h.mu.Unlock()
	st := h.scopes[sc]
	if st == nil || st.data == nil || st.data.series == nil || !h.validLocked(st) {
		return nil
	}
	return st.data.series
}

// Fetch returns the current result of a scope without subscribing: from a fresh result (the scope's
// own, or a list that covers it), else after one request. It is for one-off reads (the agents'
// command): nothing keeps polling afterwards.
func (h *Hub) Fetch(ctx context.Context, sc Scope) Result {
	if r, ok := h.cached(sc); ok {
		return r
	}
	if _, has := ctx.Deadline(); !has {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, h.opts.RequestTimeout)
		defer cancel()
	}
	state := h.fetch(ctx, sc)
	h.mu.Lock()
	defer h.mu.Unlock()
	st := h.scopes[sc]
	r := Result{State: state}
	if st != nil && st.data != nil {
		r.At, r.Pods, r.Nodes = st.data.at, sortedPods(st.data.pods, sc), sortedNodes(st.data.nodes)
	}
	return r
}

// Result is a scope's measurements and the state they came with.
type Result struct {
	State State
	// At is when the hub got the measurements; zero when there are none.
	At    time.Time
	Pods  []model.PodUsage
	Nodes []model.NodeUsage
}

// cached answers a scope from a fresh result: its own, or the all-namespaces list for a pod
// scope, or the node list for one node.
func (h *Hub) cached(sc Scope) (Result, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	cands := []Scope{sc}
	switch sc.Kind {
	case KindPods:
		if sc.Namespace != "" {
			cands = append(cands, Scope{Kind: KindPods, Namespace: sc.Namespace}, Scope{Kind: KindPods})
		}
	case KindNodes:
		if sc.Name != "" {
			cands = append(cands, Scope{Kind: KindNodes})
		}
	}
	for _, c := range cands {
		st := h.scopes[c]
		if st == nil || !h.validLocked(st) || !st.state.Usable() {
			continue
		}
		r := Result{State: st.state, At: st.data.at}
		switch sc.Kind {
		case KindPods:
			r.Pods = sortedPods(st.data.pods, sc)
		case KindNodes:
			for _, n := range sortedNodes(st.data.nodes) {
				if sc.Name == "" || n.Name == sc.Name {
					r.Nodes = append(r.Nodes, n)
				}
			}
		}
		if c != sc && sc.Name != "" && len(r.Pods)+len(r.Nodes) == 0 {
			continue // the covering list does not hold the object: ask for it
		}
		return r, true
	}
	return Result{}, false
}

// sortedPods returns the pods a scope asks for (a namespace's, or one pod) sorted by namespace and
// name.
func sortedPods(m map[string]model.PodUsage, sc Scope) []model.PodUsage {
	out := make([]model.PodUsage, 0, len(m))
	for _, p := range m {
		if (sc.Namespace == "" || p.Namespace == sc.Namespace) && (sc.Name == "" || p.Name == sc.Name) {
			out = append(out, p)
		}
	}
	slices.SortFunc(out, func(a, b model.PodUsage) int {
		return cmp.Or(cmp.Compare(a.Namespace, b.Namespace), cmp.Compare(a.Name, b.Name))
	})
	return out
}

func sortedNodes(m map[string]model.NodeUsage) []model.NodeUsage {
	out := make([]model.NodeUsage, 0, len(m))
	for _, n := range m {
		out = append(out, n)
	}
	slices.SortFunc(out, func(a, b model.NodeUsage) int { return cmp.Compare(a.Name, b.Name) })
	return out
}

// Pod returns the latest fresh measurement of a pod from any scope that holds it.
func (h *Hub) Pod(namespace, name string) (model.PodUsage, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	var best model.PodUsage
	found := false
	for _, c := range []Scope{{KindPods, "", ""}, {KindPods, namespace, ""}, {KindPods, namespace, name}} {
		st := h.scopes[c]
		if st == nil || !h.validLocked(st) {
			continue
		}
		if p, ok := st.data.pods[podKey(namespace, name)]; ok && (!found || p.Timestamp.After(best.Timestamp)) {
			best, found = p, true
		}
	}
	return best, found
}

// Node returns the latest fresh measurement of a node from any scope that holds it.
func (h *Hub) Node(name string) (model.NodeUsage, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	var best model.NodeUsage
	found := false
	for _, c := range []Scope{{Kind: KindNodes}, {Kind: KindNodes, Name: name}} {
		st := h.scopes[c]
		if st == nil || !h.validLocked(st) {
			continue
		}
		if n, ok := st.data.nodes[name]; ok && (!found || n.Timestamp.After(best.Timestamp)) {
			best, found = n, true
		}
	}
	return best, found
}

// State is what is known about the cluster's metrics for a scope: the outcome of its last poll,
// else what detection found, else unknown.
func (h *Hub) State(sc Scope) State {
	h.mu.Lock()
	defer h.mu.Unlock()
	if st := h.scopes[sc]; st != nil && st.state.Status != "" {
		return st.state
	}
	if h.state.Status != "" {
		return h.state
	}
	return State{Status: StatusUnknown}
}

// Series is one line of a chart: the whole object (Container ""), or one container of a pod.
type Series struct {
	Container string
	Samples   []model.UsageSample
}

// Series returns the time series the hub keeps for an object scope that is subscribed, oldest
// sample first: the object's own first, then its containers by name. It is empty for lists and for
// a scope nobody watches.
func (h *Hub) Series(sc Scope) []Series {
	h.mu.Lock()
	defer h.mu.Unlock()
	st := h.scopes[sc]
	if st == nil || len(st.series) == 0 {
		return nil
	}
	names := make([]string, 0, len(st.series))
	for n := range st.series {
		names = append(names, n)
	}
	slices.Sort(names) // "" sorts first
	out := make([]Series, 0, len(names))
	for _, n := range names {
		out = append(out, Series{Container: n, Samples: slices.Clone(st.series[n].pts)})
	}
	return out
}

// Subscribed reports how many scopes are being polled: zero when nothing visible shows metrics.
func (h *Hub) Subscribed() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	n := 0
	for _, st := range h.scopes {
		if st.cancel != nil {
			n++
		}
	}
	return n
}

// seriesBuf is one bounded series.
type seriesBuf struct{ pts []model.UsageSample }

// add appends a sample newer than the last; the series keeps at most max points and none older
// than age before the newest.
func (b *seriesBuf) add(s model.UsageSample, maxPoints int, age time.Duration) {
	if n := len(b.pts); n > 0 && !s.At.After(b.pts[n-1].At) {
		return
	}
	b.pts = append(b.pts, s)
	drop := 0
	if over := len(b.pts) - maxPoints; over > 0 {
		drop = over
	}
	for drop < len(b.pts)-1 && s.At.Sub(b.pts[drop].At) > age {
		drop++
	}
	if drop > 0 {
		b.pts = append(b.pts[:0], b.pts[drop:]...)
	}
}

func (h *Hub) bufLocked(st *scopeState, container string) *seriesBuf {
	if st.series == nil {
		st.series = map[string]*seriesBuf{}
	}
	b := st.series[container]
	if b == nil {
		b = &seriesBuf{}
		st.series[container] = b
	}
	return b
}

func (h *Hub) addPodSeriesLocked(st *scopeState, p model.PodUsage) {
	at := p.Timestamp
	if at.IsZero() {
		at = h.opts.Now()
	}
	h.bufLocked(st, "").add(model.UsageSample{At: at, Usage: p.Total()}, h.opts.SeriesPoints, h.opts.SeriesAge)
	for _, c := range p.Containers {
		h.bufLocked(st, c.Name).add(model.UsageSample{At: at, Usage: c.Usage}, h.opts.SeriesPoints, h.opts.SeriesAge)
	}
}

func (h *Hub) addNodeSeriesLocked(st *scopeState, n model.NodeUsage) {
	at := n.Timestamp
	if at.IsZero() {
		at = h.opts.Now()
	}
	h.bufLocked(st, "").add(model.UsageSample{At: at, Usage: n.Usage}, h.opts.SeriesPoints, h.opts.SeriesAge)
}

// recordLocked appends an object result to its scope's series.
func (h *Hub) recordLocked(st *scopeState, data *dataSet) {
	sc := st.scope
	switch sc.Kind {
	case KindPods:
		if p, ok := data.pods[podKey(sc.Namespace, sc.Name)]; ok {
			h.addPodSeriesLocked(st, p)
		}
	case KindNodes:
		if n, ok := data.nodes[sc.Name]; ok {
			h.addNodeSeriesLocked(st, n)
		}
	}
}
