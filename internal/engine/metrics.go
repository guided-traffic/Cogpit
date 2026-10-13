package engine

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/metrics"
	"github.com/guided-traffic/cogpit/internal/model"
)

// The engine's side of metrics (architecture §9): what a visible view or an open drawer subscribes
// to, how the Pods and Nodes lists get their CPU and Memory cells, what the drawer's Metrics
// section reads, and the read command for agents. The polling itself, the TTL cache and the series
// are package metrics'.

// usageFeed is a view's subscription to the metrics of its kind (a Pods list polls every pod's
// usage, the Nodes list every node's) for as long as the view is open. It tells the view worker
// when new numbers came, and answers the lookups of the row cache.
type usageFeed struct {
	hub  *metrics.Hub
	kind model.UsageKind
	sub  *metrics.Subscription
	gen  atomic.Uint64
}

// startUsageFeed subscribes to the metrics a profile's CPU and Memory columns show; nil for a kind
// without such columns or a backend without metrics. onChange runs after each poll.
func (b *Backend) startUsageFeed(p *model.KindProfile, onChange func()) *usageFeed {
	if p.Usage == "" || b.Metrics == nil {
		return nil
	}
	f := &usageFeed{hub: b.Metrics, kind: p.Usage}
	f.sub = b.Metrics.Subscribe(usageScope(p.Usage, "", ""), func() {
		f.gen.Add(1)
		onChange()
	})
	return f
}

// usageScope is the metrics scope of a usage kind: a list, or with a name one object.
func usageScope(k model.UsageKind, namespace, name string) metrics.Scope {
	if k == model.UsageNodes {
		return metrics.Scope{Kind: metrics.KindNodes, Name: name}
	}
	return metrics.Scope{Kind: metrics.KindPods, Namespace: namespace, Name: name}
}

// generation changes whenever new numbers arrived; rows patched at an older one are patched again.
func (f *usageFeed) generation() uint64 {
	if f == nil {
		return 0
	}
	return f.gen.Load()
}

// lookup returns the usage of the object a row stands for.
func (f *usageFeed) lookup(namespace, name string) (model.Usage, bool) {
	if f == nil {
		return model.Usage{}, false
	}
	if f.kind == model.UsageNodes {
		n, ok := f.hub.Node(name)
		return n.Usage, ok
	}
	p, ok := f.hub.Pod(namespace, name)
	return p.Total(), ok
}

func (f *usageFeed) release() {
	if f != nil {
		f.sub.Release()
	}
}

// withUsage returns the row with its CPU and Memory cells filled from u. The row is shared and
// immutable: the cells are a copy.
func withUsage(p *model.KindProfile, row model.Row, obj *unstructured.Unstructured, u model.Usage, cpuCol, memCol int) (out model.Row) {
	defer func() {
		if recover() != nil {
			out = row
		}
	}()
	cpu, mem := p.UsageCells(obj, u)
	cells := make([]model.Cell, len(row.Cells))
	copy(cells, row.Cells)
	if cpuCol >= 0 && cpuCol < len(cells) {
		cells[cpuCol] = cpu
	}
	if memCol >= 0 && memCol < len(cells) {
		cells[memCol] = mem
	}
	row.Cells = cells
	return row
}

// startDrawerUsage subscribes to the open drawer's object: one poll of that pod or node every 15
// seconds, which also feeds the chart's series, for as long as the drawer shows it.
func (s *Session) startDrawerUsage(b *Backend, p *model.KindProfile, ref model.ObjectRef) *metrics.Subscription {
	if p.Usage == "" || b.Metrics == nil {
		return nil
	}
	return b.Metrics.Subscribe(usageScope(p.Usage, ref.Namespace, ref.Name), func() {
		s.emit(Event{Type: EventMetrics, View: ref.Kind})
	})
}

// MetricsTarget is one chart's worth of an object's metrics: the object itself (Name ""), or one
// of a pod's containers.
type MetricsTarget struct {
	// Name is the container's name; empty for the pod or node as a whole.
	Name string `json:"name,omitempty"`
	// Current is the latest measurement; nil while there is none.
	Current *model.Usage `json:"current,omitempty"`
	// At is the time of that measurement.
	At time.Time `json:"at,omitempty"`
	// Requests and Limits are the container's (a pod's: the sum, see model.PodTotals), 0 for not
	// set. Allocatable and Capacity are a node's. The chart draws them as horizontal lines.
	Requests    model.Usage `json:"requests"`
	Limits      model.Usage `json:"limits"`
	Allocatable model.Usage `json:"allocatable"`
	Capacity    model.Usage `json:"capacity"`
	// Samples are the CPU and memory measurements, oldest first. With the Metrics Server: since
	// the drawer opened, bounded to an hour. With Prometheus: the chart's time range, from its
	// history.
	Samples []model.UsageSample `json:"samples,omitempty"`
	// Network is the traffic in bytes per second, from a provider with the Network capability
	// (Prometheus). The network belongs to the pod: a container's target carries the pod's samples
	// and says so with NetworkOfPod.
	Network      []model.NetworkSample `json:"network,omitempty"`
	NetworkOfPod bool                  `json:"networkOfPod,omitempty"`
	// Filesystem is from a provider with the Filesystem capability: bytes in use, or read and
	// write rates when ObjectMetrics.FilesystemSource says "io".
	Filesystem []model.FilesystemSample `json:"filesystem,omitempty"`
	// Window is the chart's time range as the engine holds it: the default hour unless
	// ui.setMetricsRange (Session.SetMetricsRange) set another for this chart of the open drawer.
	// The window's range control shows it, whoever set it. Range is the same as ui.setMetricsRange
	// writes it ("1h", "7d").
	Window time.Duration `json:"-"`
	Range  string        `json:"range"`
	// Variant is what the chart shows as the engine holds it: "cpu" (the default), "memory",
	// "network" or "filesystem" (ui.setMetricsVariant). A variant the providers do not have (the
	// network without Prometheus) reads as "cpu". The network and the filesystem are asked of
	// Prometheus only while a chart shows them.
	Variant string `json:"variant"`
}

// The variants of a chart (MetricsTarget.Variant, ui.setMetricsVariant).
const (
	VariantCPU        = "cpu"
	VariantMemory     = "memory"
	VariantNetwork    = "network"
	VariantFilesystem = "filesystem"
)

// MetricsVariants are the variants a chart can show, in the order of its buttons.
var MetricsVariants = []string{VariantCPU, VariantMemory, VariantNetwork, VariantFilesystem}

// ObjectMetrics is the drawer's Metrics section for a pod or a node: where the numbers come from,
// or why there are none, and for each target its latest measurement, its requests and limits and
// the series since the drawer opened.
type ObjectMetrics struct {
	Ref model.ObjectRef `json:"ref"`
	// Status is the metrics state: "available", "notServed" (no metrics-server), "forbidden",
	// "unavailable", or "unknown" before the first poll answered.
	Status metrics.Status `json:"status"`
	// Source is the provider as Lens's source line names it ("Kubernetes Metrics Server") when
	// Status is available.
	Source string `json:"source,omitempty"`
	// Message says why Status is not available.
	Message string `json:"message,omitempty"`
	// Capabilities say which variants the provider can draw (network and filesystem need
	// Prometheus).
	Capabilities metrics.Capabilities `json:"capabilities"`
	// Variants are the variants the charts offer, in the buttons' order: those the cluster's
	// settings do not hide (hidemetrics.go). With all of them hidden, Status is "hidden" and there
	// are no targets.
	Variants []string `json:"variants,omitempty"`
	// FilesystemSource says which filesystem metric the Filesystem series are: "usage" (bytes in
	// use) or "io" (read and write rates); empty when there are none.
	FilesystemSource string `json:"filesystemSource,omitempty"`
	// Providers is the status line under the charts: each of the cluster's providers (Prometheus,
	// the Metrics Server), whether Cogpit reached it and it answered, its last good answer and last
	// failure (a class, never a response body), and what it serves now (history, live values).
	Providers []metrics.ProviderStatus `json:"providers,omitempty"`
	// Targets: the object first, then a pod's containers in spec order.
	Targets []MetricsTarget `json:"targets"`
}

// ObjectMetrics returns the Metrics section of a pod or node, or nil, nil for a kind without one.
// It reads the hub's latest results and the cached object and sends no request; the open drawer
// keeps the numbers coming.
func (s *Session) ObjectMetrics(ref model.ObjectRef) (*ObjectMetrics, error) {
	b, ref, p, err := s.resolve(ref)
	if err != nil {
		return nil, err
	}
	if p.Usage == "" {
		return nil, nil
	}
	out := &ObjectMetrics{Ref: ref, Status: metrics.StatusNotServed, Message: "this cluster has no metrics provider", Targets: []MetricsTarget{}}
	hidden := s.hiddenMetrics(b.Context)
	visible := hidden.drawerVariants(p.Usage)
	if len(visible) == 0 {
		out.Status, out.Message = StatusHidden, msgHidden
		return out, nil
	}
	out.Variants = visible
	if b.Metrics == nil {
		return out, nil
	}
	scope := usageScope(p.Usage, ref.Namespace, ref.Name)
	st := b.Metrics.State(scope)
	out.Status, out.Message, out.Source = st.Status, st.Message, st.Label
	out.Providers = b.Metrics.Providers()
	caps := b.Metrics.Capabilities()
	if st.Usable() {
		out.Message = ""
		out.Capabilities = caps
	}
	h := b.Acquire(p)
	defer h.Release()
	obj, _ := KindGet(h, p, ref.Namespace, ref.Name)
	switch p.Usage {
	case model.UsagePods:
		out.Targets = podTargets(b.Metrics, ref, obj)
		if hidden&HideContainers != 0 {
			out.Targets = out.Targets[:1] // the containers' charts are hidden: the pod's alone
		}
	case model.UsageNodes:
		out.Targets = nodeTargets(b.Metrics, ref, obj)
	}
	setWindows(out.Targets, s.chartWindows(ref), b.Metrics.DefaultWindow())
	setVariants(out.Targets, s.chartVariants(ref), caps, visible)
	if hist := b.Metrics.History(scope); hist != nil && st.Usable() {
		out.FilesystemSource = hist.FilesystemSource
		applyHistory(out.Targets, hist)
	}
	return out, nil
}

// setVariants gives each target its chart's variant: the one set for it if it is visible and the
// providers have it, else the default (CPU, or the first visible variant while CPU is hidden).
func setVariants(targets []MetricsTarget, variants map[string]string, caps metrics.Capabilities, visible []string) {
	shown := shownVariants(visible, caps)
	def := defaultVariant(visible, caps)
	for i := range targets {
		t := &targets[i]
		t.Variant = def
		if v := variants[t.Name]; v != "" && slices.Contains(shown, v) {
			t.Variant = v
		}
	}
}

// setWindows gives each target its chart's time range: the one set for it, else the default.
func setWindows(targets []MetricsTarget, windows map[string]time.Duration, def time.Duration) {
	for i := range targets {
		t := &targets[i]
		t.Window = def
		if w, ok := windows[t.Name]; ok {
			t.Window = w
		}
		t.Range = FormatChartRange(t.Window)
	}
}

// applyHistory fills the targets' series from a provider with history, each cut to its chart's
// time range (counted back from the end of the query). The hub fetched the longest range any
// chart asked for; a chart that shows less gets the tail.
func applyHistory(targets []MetricsTarget, hist *metrics.ObjectSeries) {
	whole := hist.Targets[""]
	for i := range targets {
		t := &targets[i]
		series := hist.Targets[t.Name]
		if series == nil {
			continue
		}
		cutoff := hist.To.Add(-t.Window)
		t.Samples = tail(series.Usage, cutoff, func(x model.UsageSample) time.Time { return x.At })
		t.Filesystem = tail(series.Filesystem, cutoff, func(x model.FilesystemSample) time.Time { return x.At })
		if whole != nil && len(whole.Network) > 0 {
			t.Network = tail(whole.Network, cutoff, func(x model.NetworkSample) time.Time { return x.At })
			t.NetworkOfPod = t.Name != ""
		}
	}
}

// tail returns the points at or after cutoff; the points are shared and must not be modified.
func tail[T any](pts []T, cutoff time.Time, at func(T) time.Time) []T {
	i := slices.IndexFunc(pts, func(p T) bool { return !at(p).Before(cutoff) })
	if i < 0 {
		return nil
	}
	return pts[i:]
}

// MaxChartWindow bounds a chart's time range: Prometheus's default retention is 15 days.
const MaxChartWindow = 30 * 24 * time.Hour

// MinChartWindow is the shortest range a chart can show.
const MinChartWindow = time.Minute

// DefaultChartRange names the default time range in ui.setMetricsRange.
const DefaultChartRange = "default"

// ParseChartRange reads a chart's time range as ui.setMetricsRange takes it: a Go duration ("15m",
// "1h", "90m") or whole days ("7d"); "" or "default" is the default range (0).
func ParseChartRange(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == DefaultChartRange {
		return 0, nil
	}
	if days, ok := strings.CutSuffix(s, "d"); ok {
		n, err := strconv.Atoi(days)
		if err != nil || n <= 0 {
			return 0, fmt.Errorf("time range %q: days are a whole number, as in 7d", s)
		}
		return time.Duration(n) * 24 * time.Hour, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("time range %q: use minutes, hours or days, as in 15m, 6h, 7d", s)
	}
	return d, nil
}

// FormatChartRange writes a time range the way ParseChartRange reads it, in its largest whole
// unit: "5m", "1h", "7d"; anything else as a Go duration.
func FormatChartRange(d time.Duration) string {
	switch {
	case d <= 0:
		return ""
	case d%(24*time.Hour) == 0:
		return strconv.FormatInt(int64(d/(24*time.Hour)), 10) + "d"
	case d%time.Hour == 0:
		return strconv.FormatInt(int64(d/time.Hour), 10) + "h"
	case d%time.Minute == 0:
		return strconv.FormatInt(int64(d/time.Minute), 10) + "m"
	}
	return d.String()
}

// SetMetricsRange sets the time range of one chart of the open drawer: target is "" for the pod's
// or node's chart, or a container's name. A provider with history (Prometheus) is then asked once
// for the longest range any chart shows, at the step of the shortest, one query set for all of
// them, at once. Window 0 puts the chart back to the default range (one hour). The Metrics Server
// has no history: the range is remembered and changes nothing the engine returns. The charts'
// ranges are the drawer's: they end when it closes or shows another object.
func (s *Session) SetMetricsRange(ref model.ObjectRef, target string, window time.Duration) error {
	if window < 0 || window > 0 && window < MinChartWindow || window > MaxChartWindow {
		return fmt.Errorf("a chart's time range is between %s and %s", FormatChartRange(MinChartWindow), FormatChartRange(MaxChartWindow))
	}
	st := s.state
	st.mu.Lock()
	d := st.drawer
	if d == nil || !d.ref.Same(ref) {
		st.mu.Unlock()
		return ErrDrawerNotOpen
	}
	st.mu.Unlock()
	d.wmu.Lock()
	if d.windows == nil {
		d.windows = map[string]time.Duration{}
	}
	if window == 0 {
		delete(d.windows, target)
	} else {
		d.windows[target] = window
	}
	list := d.windowListLocked()
	sub := d.usage
	d.wmu.Unlock()
	sub.SetWindows(list...)
	s.emit(Event{Type: EventMetrics, View: ref.Kind})
	return nil
}

// ErrDrawerNotOpen refuses a drawer-scoped call for an object the drawer does not show.
var ErrDrawerNotOpen = errors.New("the drawer does not show this object")

// chartWindows returns the time ranges set for the charts of the open drawer if it shows ref.
func (s *Session) chartWindows(ref model.ObjectRef) map[string]time.Duration {
	st := s.state
	st.mu.Lock()
	d := st.drawer
	if d == nil || !d.ref.Same(ref) {
		st.mu.Unlock()
		return nil
	}
	st.mu.Unlock()
	d.wmu.Lock()
	defer d.wmu.Unlock()
	return maps.Clone(d.windows)
}

// SetMetricsVariant sets what one chart of the open drawer shows: target is "" for the pod's or
// node's chart, or a container's name; variant is "cpu", "memory", "network" or "filesystem" ("" is
// "cpu"). The network and the filesystem exist only in Prometheus: they are asked of it only while
// a chart shows them (once over the chart's range, then at its step), so this is what starts and
// stops those requests. A variant the cluster's providers do not have is refused.
func (s *Session) SetMetricsVariant(ref model.ObjectRef, target, variant string) error {
	reset := variant == "" // the chart's default: CPU, or the first visible variant while CPU is hidden
	if reset {
		variant = VariantCPU
	}
	if !slices.Contains(MetricsVariants, variant) {
		return fmt.Errorf("a chart shows %s, not %q", strings.Join(MetricsVariants, ", "), variant)
	}
	st := s.state
	st.mu.Lock()
	d := st.drawer
	if d == nil || !d.ref.Same(ref) {
		st.mu.Unlock()
		return ErrDrawerNotOpen
	}
	st.mu.Unlock()
	if !reset && !slices.Contains(s.hiddenMetrics(d.backend.Context).drawerVariants(d.profile.Usage), variant) {
		return fmt.Errorf("%w: the %s chart is %s", ErrNotApplicable, variant, msgHidden)
	}
	if b := d.backend; variant == VariantNetwork || variant == VariantFilesystem {
		caps := metrics.Capabilities{}
		if b.Metrics != nil {
			caps = b.Metrics.Capabilities()
		}
		if variant == VariantNetwork && !caps.Network || variant == VariantFilesystem && !caps.Filesystem {
			return fmt.Errorf("%w: the %s chart needs Prometheus, and this cluster's metrics come without it", ErrNotApplicable, variant)
		}
	}
	d.wmu.Lock()
	if d.variants == nil {
		d.variants = map[string]string{}
	}
	if variant == VariantCPU {
		delete(d.variants, target)
	} else {
		d.variants[target] = variant
	}
	want := d.wantsLocked()
	sub := d.usage
	d.wmu.Unlock()
	sub.SetWants(want)
	s.emit(Event{Type: EventMetrics, View: ref.Kind})
	return nil
}

// chartVariants returns the variants set for the charts of the open drawer if it shows ref.
func (s *Session) chartVariants(ref model.ObjectRef) map[string]string {
	st := s.state
	st.mu.Lock()
	d := st.drawer
	if d == nil || !d.ref.Same(ref) {
		st.mu.Unlock()
		return nil
	}
	st.mu.Unlock()
	d.wmu.Lock()
	defer d.wmu.Unlock()
	return maps.Clone(d.variants)
}

// promServices lists the cluster's Services for Prometheus detection, from the cache: the Services
// kind's informer, shared with the Services view and kept warm for the warm period, so detection
// sends no list of its own. A cluster that does not serve Services has no candidates; a user who
// may not list them gets a state that says so, and detection relies on the override.
func (b *Backend) promServices(ctx context.Context) ([]metrics.Service, error) {
	p, ok := b.Profile("services")
	if !ok {
		return nil, nil
	}
	// A user who is known not to be allowed to list Services gets no watch that would be refused.
	// The access reviews that answer it start with the connection and take a moment: wait for them
	// briefly, and go ahead when they do not answer (the API server decides).
	for range accessWaitTries {
		listable, known := b.Listable(p, nil)
		if known && !listable {
			return nil, errors.New("the current user may not list services, so Prometheus cannot be looked for (name it in the cluster settings)")
		}
		if known || ctx.Err() != nil {
			break
		}
		select {
		case <-ctx.Done():
		case <-time.After(accessWaitStep):
		}
	}
	h := b.Acquire(p)
	defer h.Release()
	wctx, cancel := context.WithTimeout(ctx, servicesSyncTimeout)
	defer cancel()
	if !h.WaitForSync(wctx) {
		if err := h.Err(); err != nil {
			return nil, fmt.Errorf("the cluster's services cannot be read: %w", err)
		}
		return nil, errors.New("the cluster's services are not loaded yet")
	}
	return servicesOf(KindList(h, p)), nil
}

// accessWaitTries and accessWaitStep bound how long detection waits for the access reviews that
// say whether the user may list Services: 3 seconds.
const (
	accessWaitTries = 30
	accessWaitStep  = 100 * time.Millisecond
)

// servicesSyncTimeout is how long detection waits for the Services informer's first list.
const servicesSyncTimeout = 8 * time.Second

// servicesOf reads the names and ports of cached Services.
func servicesOf(objs []*unstructured.Unstructured) []metrics.Service {
	out := make([]metrics.Service, 0, len(objs))
	for _, o := range objs {
		svc := metrics.Service{Namespace: o.GetNamespace(), Name: o.GetName()}
		ports, _, _ := unstructured.NestedSlice(o.Object, "spec", "ports")
		for _, raw := range ports {
			m, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			name, _ := m["name"].(string)
			var num int32
			switch v := m["port"].(type) {
			case int64:
				num = int32(v)
			case float64:
				num = int32(v)
			case int32:
				num = v
			case int:
				num = int32(v)
			}
			svc.Ports = append(svc.Ports, metrics.ServicePort{Name: name, Port: num})
		}
		out = append(out, svc)
	}
	return out
}

func podTargets(hub *metrics.Hub, ref model.ObjectRef, obj *unstructured.Unstructured) []MetricsTarget {
	pu, have := hub.Pod(ref.Namespace, ref.Name)
	var res []model.ContainerResources
	if obj != nil {
		res = model.PodResources(obj)
	}
	whole := MetricsTarget{}
	whole.Requests, whole.Limits = model.PodTotals(res)
	if have {
		total := pu.Total()
		whole.Current, whole.At = &total, pu.Timestamp
	}
	targets := make([]MetricsTarget, 1, 1+len(res))
	targets[0] = whole
	for _, r := range res {
		t := MetricsTarget{Name: r.Name, Requests: r.Requests, Limits: r.Limits}
		for _, c := range pu.Containers {
			if c.Name == r.Name && have {
				u := c.Usage
				t.Current, t.At = &u, pu.Timestamp
			}
		}
		targets = append(targets, t)
	}
	return targets
}

func nodeTargets(hub *metrics.Hub, ref model.ObjectRef, obj *unstructured.Unstructured) []MetricsTarget {
	t := MetricsTarget{}
	if obj != nil {
		t.Allocatable, t.Capacity = model.NodeCapacity(obj)
	}
	if nu, ok := hub.Node(ref.Name); ok {
		u := nu.Usage
		t.Current, t.At = &u, nu.Timestamp
	}
	return []MetricsTarget{t}
}

// PodUsageItem is one pod in MetricsResult.
type PodUsageItem struct {
	Namespace string    `json:"namespace"`
	Name      string    `json:"name"`
	Timestamp time.Time `json:"timestamp"`
	model.Usage
	Containers []model.ContainerUsage `json:"containers"`
}

// MetricsResult is metrics.get's result: the current usage of one pod or node, of the pods of a
// namespace or of all, or of the nodes.
type MetricsResult struct {
	Status  metrics.Status `json:"status"`
	Source  string         `json:"source,omitempty"`
	Message string         `json:"message,omitempty"`
	// At is when Cogpit got the numbers; the measurements' own timestamps are on the items.
	At    time.Time         `json:"at,omitempty"`
	Pods  []PodUsageItem    `json:"pods,omitempty"`
	Nodes []model.NodeUsage `json:"nodes,omitempty"`
	// Total counts the matching objects before Limit cut the list.
	Total     int  `json:"total"`
	Truncated bool `json:"truncated,omitempty"`
	// Providers is what Cogpit knows of each of the cluster's providers (the drawer's status
	// line): reached and answering or not, the last good answer, the last failure's class, what it
	// serves now. Prometheus is looked for only when a drawer opens or the Metrics Server fails,
	// so it may be "unknown".
	Providers []metrics.ProviderStatus `json:"providers,omitempty"`
}

// DefaultMetricsLimit is how many pods metrics.get returns unless asked for fewer or more.
const DefaultMetricsLimit = 500

// MetricsGet reads the current metrics of pods or nodes for an agent. A pod or a node by name, or
// the pods of a namespace, or all pods, or all nodes; a fresh result (within 15 seconds) is
// served without a request, else one `get` or `list` is sent. Nothing keeps polling afterwards.
// No metrics-server is a result with a status, not an error.
func (s *Session) MetricsGet(ctx context.Context, kind, namespace, name string, limit int) (*MetricsResult, error) {
	b := s.Backend()
	if b == nil {
		return nil, ErrNotConnected
	}
	if limit <= 0 {
		limit = DefaultMetricsLimit
	}
	var scope metrics.Scope
	switch metrics.Kind(kind) {
	case metrics.KindPods:
		if name != "" && namespace == "" {
			return nil, errors.New("a pod is named with its namespace")
		}
		scope = metrics.Scope{Kind: metrics.KindPods, Namespace: namespace, Name: name}
	case metrics.KindNodes:
		scope = metrics.Scope{Kind: metrics.KindNodes, Name: name}
	default:
		return nil, fmt.Errorf("kind %q has no metrics: use pods or nodes", kind)
	}
	if b.Metrics == nil {
		return &MetricsResult{Status: metrics.StatusNotServed, Message: "this cluster has no metrics provider"}, nil
	}
	r := b.Metrics.Fetch(ctx, scope)
	out := &MetricsResult{Status: r.State.Status, Source: r.State.Label, Message: r.State.Message, At: r.At, Providers: b.Metrics.Providers()}
	if r.State.Usable() {
		out.Message = ""
	}
	out.Total = len(r.Pods) + len(r.Nodes)
	if r.State.Usable() && out.Total == 0 && name != "" {
		out.Message = "no metrics for " + kind + "/" + name + " yet (it may have started a moment ago)"
	}
	for i, p := range r.Pods {
		if i >= limit {
			out.Truncated = true
			break
		}
		out.Pods = append(out.Pods, PodUsageItem{Namespace: p.Namespace, Name: p.Name, Timestamp: p.Timestamp, Usage: p.Total(), Containers: p.Containers})
	}
	for i, n := range r.Nodes {
		if i >= limit {
			out.Truncated = true
			break
		}
		out.Nodes = append(out.Nodes, n)
	}
	return out, nil
}
