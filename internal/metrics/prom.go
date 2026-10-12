package metrics

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"

	"github.com/guided-traffic/cogpit/internal/model"
)

// PrometheusID names the Prometheus provider.
const PrometheusID = "prometheus"

// Bounds of what a Prometheus may send back or take.
const (
	// DefaultMaxResponseBytes caps one response body. A list of 10,000 pods is a few MiB; more
	// than this is a Prometheus answering something that is not meant for a desktop.
	DefaultMaxResponseBytes = 16 << 20
	// DefaultQueryTimeout bounds one query (the hub's poll timeout bounds all of a poll's).
	DefaultQueryTimeout = 15 * time.Second
	// maxPointsPerSeries rejects a series longer than Prometheus's own limit for a range query.
	maxPointsPerSeries = 11000
)

// Target is a Prometheus reached through the API server's service proxy:
// /api/v1/namespaces/<Namespace>/services/<Service>:<Port>/proxy<Prefix>/api/v1/...
type Target struct {
	Namespace string
	Service   string
	// Port is the service port's number or name.
	Port string
	// Prefix is the route prefix Prometheus serves under (--web.route-prefix), "" or "/"
	// for the root.
	Prefix string
}

// String is "namespace/service", the name the source line shows.
func (t Target) String() string { return t.Namespace + "/" + t.Service }

func (t Target) prefix() string {
	p := strings.Trim(t.Prefix, "/")
	if p == "" {
		return ""
	}
	return "/" + p
}

// proxy sends a GET to the Prometheus API through the API server and returns the body, at most
// limit bytes.
type proxy interface {
	get(ctx context.Context, t Target, path string, params map[string]string, limit int64) ([]byte, error)
}

// kubeProxy is the production proxy: client-go's service ProxyGet, on the connection's own
// clients, so the request is counted, rate limited and authenticated like every other.
type kubeProxy struct{ kube kubernetes.Interface }

func (k kubeProxy) get(ctx context.Context, t Target, path string, params map[string]string, limit int64) ([]byte, error) {
	body, err := k.kube.CoreV1().Services(t.Namespace).ProxyGet("", t.Service, t.Port, t.prefix()+path, params).Stream(ctx)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	b, err := io.ReadAll(io.LimitReader(body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, fmt.Errorf("the answer is larger than %d bytes", limit)
	}
	return b, nil
}

// PrometheusOptions tune the provider; zero fields take the defaults.
type PrometheusOptions struct {
	MaxResponseBytes int64
	QueryTimeout     time.Duration
	Logger           *slog.Logger
}

// Prometheus is the provider that reads a cluster's Prometheus through the API server's service
// proxy. It answers range queries (it has history, network and filesystem) and, for the lists, the
// current values by instant queries. It only ever sends GET requests. Safe for concurrent use.
type Prometheus struct {
	px     proxy
	target Target
	opts   PrometheusOptions

	// fsMode is which filesystem metric this Prometheus has: fsUnknown until a query found one.
	fsMode atomic.Int32
}

// Filesystem metric modes.
const (
	fsUnknown int32 = iota
	fsUsage
	fsIO
)

// NewPrometheus returns the provider for target, reached through kube's service proxy.
func NewPrometheus(kube kubernetes.Interface, target Target, opts PrometheusOptions) *Prometheus {
	return newPrometheus(kubeProxy{kube}, target, opts)
}

func newPrometheus(px proxy, target Target, opts PrometheusOptions) *Prometheus {
	if opts.MaxResponseBytes <= 0 {
		opts.MaxResponseBytes = DefaultMaxResponseBytes
	}
	if opts.QueryTimeout <= 0 {
		opts.QueryTimeout = DefaultQueryTimeout
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	return &Prometheus{px: px, target: target, opts: opts}
}

// ID implements Provider.
func (p *Prometheus) ID() string { return PrometheusID }

// Label implements Provider: "Prometheus (namespace/service)", which the drawer prints after
// "Displaying metrics from".
func (p *Prometheus) Label() string { return "Prometheus (" + p.target.String() + ")" }

// Target returns the service the provider reads.
func (p *Prometheus) Target() Target { return p.target }

// Capabilities implements Provider: history, network and filesystem.
func (p *Prometheus) Capabilities() Capabilities {
	return Capabilities{History: true, Network: true, Filesystem: true}
}

// FilesystemSource says which filesystem metric this Prometheus turned out to have: "usage",
// "io", or "" while unknown.
func (p *Prometheus) FilesystemSource() string {
	switch p.fsMode.Load() {
	case fsUsage:
		return FilesystemUsage
	case fsIO:
		return FilesystemIO
	}
	return ""
}

// point is one sample, the time in Unix milliseconds.
type point struct {
	t int64
	v float64
}

// promSeries is one series of a result: its labels and its points (one for an instant query).
type promSeries struct {
	labels map[string]string
	points []point
}

// samplePair is the wire form of a sample: [unix seconds as a number, "value" as a string].
type samplePair struct{ p point }

func (s *samplePair) UnmarshalJSON(b []byte) error {
	var raw [2]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	var ts float64
	if err := json.Unmarshal(raw[0], &ts); err != nil {
		return err
	}
	var str string
	if err := json.Unmarshal(raw[1], &str); err != nil {
		return err
	}
	v, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return err
	}
	s.p = point{t: int64(math.Round(ts * 1000)), v: v}
	return nil
}

type apiResponse struct {
	Status    string `json:"status"`
	ErrorType string `json:"errorType"`
	Error     string `json:"error"`
	Data      struct {
		ResultType string          `json:"resultType"`
		Result     json.RawMessage `json:"result"`
	} `json:"data"`
}

type resultItem struct {
	Metric map[string]string `json:"metric"`
	Value  *samplePair       `json:"value"`
	Values []samplePair      `json:"values"`
}

// parseResult parses the body of /api/v1/query or /api/v1/query_range into series. NaN and
// infinite samples (a rate over a counter reset, a division by zero) are dropped. A "status":
// "error" body is an error carrying Prometheus's own text.
func parseResult(body []byte) ([]promSeries, error) {
	var r apiResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("not a Prometheus answer: %w", err)
	}
	if r.Status != "success" {
		return nil, fmt.Errorf("prometheus: %s: %s", r.ErrorType, r.Error)
	}
	var items []resultItem
	switch r.Data.ResultType {
	case "matrix", "vector":
		if err := json.Unmarshal(r.Data.Result, &items); err != nil {
			return nil, fmt.Errorf("unreadable %s result: %w", r.Data.ResultType, err)
		}
	case "scalar", "string":
		return nil, nil
	default:
		return nil, fmt.Errorf("unexpected result type %q", r.Data.ResultType)
	}
	out := make([]promSeries, 0, len(items))
	for _, it := range items {
		if len(it.Values) > maxPointsPerSeries {
			return nil, fmt.Errorf("a series of %d points is more than expected", len(it.Values))
		}
		s := promSeries{labels: it.Metric}
		if s.labels == nil {
			s.labels = map[string]string{}
		}
		if it.Value != nil {
			s.points = appendFinite(s.points, it.Value.p)
		}
		for _, v := range it.Values {
			s.points = appendFinite(s.points, v.p)
		}
		out = append(out, s)
	}
	return out, nil
}

func appendFinite(pts []point, p point) []point {
	if math.IsNaN(p.v) || math.IsInf(p.v, 0) {
		return pts
	}
	return append(pts, p)
}

// queryRange runs a range query.
func (p *Prometheus) queryRange(ctx context.Context, expr string, from, to time.Time, step time.Duration) ([]promSeries, error) {
	return p.run(ctx, "/api/v1/query_range", map[string]string{
		"query": expr,
		"start": formatTime(from),
		"end":   formatTime(to),
		"step":  formatSeconds(step),
	})
}

// queryInstant runs an instant query at the server's now.
func (p *Prometheus) queryInstant(ctx context.Context, expr string) ([]promSeries, error) {
	return p.run(ctx, "/api/v1/query", map[string]string{"query": expr})
}

func formatTime(t time.Time) string {
	return strconv.FormatFloat(float64(t.UnixMilli())/1000, 'f', -1, 64)
}

func formatSeconds(d time.Duration) string { return strconv.FormatFloat(d.Seconds(), 'f', -1, 64) }

func (p *Prometheus) run(ctx context.Context, path string, params map[string]string) ([]promSeries, error) {
	ctx, cancel := context.WithTimeout(ctx, p.opts.QueryTimeout)
	defer cancel()
	body, err := p.px.get(ctx, p.target, path, params, p.opts.MaxResponseBytes)
	if err != nil {
		return nil, p.classify(ctx, err)
	}
	s, err := parseResult(body)
	if err != nil {
		return nil, &StateError{State: State{Status: StatusUnavailable, Message: "Prometheus " + p.target.String() + " sent an answer Cogpit cannot read: " + err.Error()}, Err: err}
	}
	return s, nil
}

// classify turns a failed proxy request into an error the hub reads as a state.
func (p *Prometheus) classify(ctx context.Context, err error) error {
	where := p.target.String()
	switch {
	case apierrors.IsForbidden(err), apierrors.IsUnauthorized(err):
		return &StateError{State: State{Status: StatusForbidden, Message: "the current user may not use the service proxy of " + where + ": " + errorText(err)}, Err: err}
	case apierrors.IsNotFound(err):
		return &StateError{State: State{Status: StatusNotServed, Message: where + " does not answer the Prometheus API (the service is gone or serves another path): " + errorText(err)}, Err: err}
	case errors.Is(err, context.Canceled) && ctx.Err() != nil:
		return err
	case apierrors.IsServiceUnavailable(err), errors.Is(err, context.DeadlineExceeded), apierrors.IsTimeout(err), apierrors.IsServerTimeout(err):
		return &StateError{State: State{Status: StatusUnavailable, Message: where + " does not answer (is Prometheus running?): " + errorText(err)}, Err: err}
	}
	return &StateError{State: State{Status: StatusUnavailable, Message: "Prometheus " + where + " failed: " + errorText(err)}, Err: err}
}

// --- current values (the lists, a one-off read) ---

func seconds2milli(v float64) int64 { return int64(math.Round(v * 1000)) }

// ListPods implements Provider with two instant queries: CPU and memory per container.
func (p *Prometheus) ListPods(ctx context.Context, namespace string) ([]model.PodUsage, error) {
	cpu, mem, err := p.instantPair(ctx, podListRate(metricCPU, namespace), podListGauge(metricMemory, namespace))
	if err != nil {
		return nil, err
	}
	return podUsages(cpu, mem), nil
}

// GetPod implements Provider.
func (p *Prometheus) GetPod(ctx context.Context, namespace, name string) (model.PodUsage, error) {
	cpu, mem, err := p.instantPair(ctx, podRate(metricCPU, namespace, name), podGauge(metricMemory, namespace, name))
	if err != nil {
		return model.PodUsage{}, err
	}
	for _, s := range append(cpu, mem...) {
		s.labels["namespace"], s.labels["pod"] = namespace, name
	}
	for _, u := range podUsages(cpu, mem) {
		if u.Namespace == namespace && u.Name == name {
			return u, nil
		}
	}
	return model.PodUsage{}, ErrNoData
}

// ListNodes implements Provider through the root cgroup of each node's cAdvisor series.
func (p *Prometheus) ListNodes(ctx context.Context) ([]model.NodeUsage, error) {
	cpu, mem, err := p.instantPair(ctx, nodeListSum(metricCPU, true), nodeListSum(metricMemory, false))
	if err != nil {
		return nil, err
	}
	return nodeUsages(cpu, mem), nil
}

// GetNode implements Provider.
func (p *Prometheus) GetNode(ctx context.Context, name string) (model.NodeUsage, error) {
	cpu, mem, err := p.instantPair(ctx, nodeSum(metricCPU, name, true), nodeSum(metricMemory, name, false))
	if err != nil {
		return model.NodeUsage{}, err
	}
	for _, s := range append(cpu, mem...) {
		s.labels["node"] = name
	}
	for _, u := range nodeUsages(cpu, mem) {
		if u.Name == name {
			return u, nil
		}
	}
	return model.NodeUsage{}, ErrNoData
}

// instantPair runs the CPU and the memory query of a list at once.
func (p *Prometheus) instantPair(ctx context.Context, cpuExpr, memExpr string) (cpu, mem []promSeries, err error) {
	var cerr, merr error
	runParallel(
		func() { cpu, cerr = p.queryInstant(ctx, cpuExpr) },
		func() { mem, merr = p.queryInstant(ctx, memExpr) },
	)
	if err := errors.Join(cerr, merr); err != nil {
		return nil, nil, firstStateError(cerr, merr)
	}
	return cpu, mem, nil
}

// firstStateError prefers an error that carries a state, so a forbidden proxy is not reported as
// the joined text of two failures.
func firstStateError(errs ...error) error {
	var se *StateError
	for _, e := range errs {
		if errors.As(e, &se) {
			return e
		}
	}
	for _, e := range errs {
		if e != nil {
			return e
		}
	}
	return nil
}

// runParallel runs fns at once and waits; a panic in one is recovered at its boundary, so a bug
// in a parser cannot take the process down from a cluster I/O goroutine.
func runParallel(fns ...func()) {
	var wg sync.WaitGroup
	for _, fn := range fns {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { _ = recover() }()
			fn()
		}()
	}
	wg.Wait()
}

func podUsages(cpu, mem []promSeries) []model.PodUsage {
	type key struct{ ns, pod string }
	pods := map[key]*model.PodUsage{}
	cont := map[key]map[string]*model.ContainerUsage{}
	get := func(s promSeries) (*model.ContainerUsage, bool) {
		if len(s.points) == 0 || s.labels["pod"] == "" {
			return nil, false
		}
		k := key{s.labels["namespace"], s.labels["pod"]}
		pu := pods[k]
		if pu == nil {
			pu = &model.PodUsage{Namespace: k.ns, Name: k.pod}
			pods[k] = pu
			cont[k] = map[string]*model.ContainerUsage{}
		}
		name := s.labels["container"]
		cu := cont[k][name]
		if cu == nil {
			cu = &model.ContainerUsage{Name: name}
			cont[k][name] = cu
		}
		if ts := time.UnixMilli(s.points[0].t); ts.After(pu.Timestamp) {
			pu.Timestamp = ts
		}
		return cu, true
	}
	for _, s := range cpu {
		if cu, ok := get(s); ok {
			cu.CPUMilli = seconds2milli(s.points[0].v)
		}
	}
	for _, s := range mem {
		if cu, ok := get(s); ok {
			cu.MemoryBytes = int64(s.points[0].v)
		}
	}
	out := make([]model.PodUsage, 0, len(pods))
	for k, pu := range pods {
		names := make([]string, 0, len(cont[k]))
		for n := range cont[k] {
			names = append(names, n)
		}
		slices.Sort(names)
		for _, n := range names {
			pu.Containers = append(pu.Containers, *cont[k][n])
		}
		out = append(out, *pu)
	}
	slices.SortFunc(out, func(a, b model.PodUsage) int {
		return strings.Compare(a.Namespace+"/"+a.Name, b.Namespace+"/"+b.Name)
	})
	return out
}

// nodeName is a node series' name: the node label (kube-prometheus-stack), else the instance
// without its port (the Prometheus chart's cAdvisor job names the instance after the node).
func nodeName(l map[string]string) string {
	if n := l["node"]; n != "" {
		return n
	}
	inst := l["instance"]
	if i := strings.LastIndexByte(inst, ':'); i > 0 && !strings.Contains(inst[i:], "]") {
		inst = inst[:i]
	}
	return inst
}

func nodeUsages(cpu, mem []promSeries) []model.NodeUsage {
	nodes := map[string]*model.NodeUsage{}
	get := func(s promSeries) *model.NodeUsage {
		n := nodeName(s.labels)
		if len(s.points) == 0 || n == "" {
			return nil
		}
		nu := nodes[n]
		if nu == nil {
			nu = &model.NodeUsage{Name: n}
			nodes[n] = nu
		}
		if ts := time.UnixMilli(s.points[0].t); ts.After(nu.Timestamp) {
			nu.Timestamp = ts
		}
		return nu
	}
	for _, s := range cpu {
		if nu := get(s); nu != nil {
			nu.CPUMilli = seconds2milli(s.points[0].v)
		}
	}
	for _, s := range mem {
		if nu := get(s); nu != nil {
			nu.MemoryBytes = int64(s.points[0].v)
		}
	}
	out := make([]model.NodeUsage, 0, len(nodes))
	for _, nu := range nodes {
		out = append(out, *nu)
	}
	slices.SortFunc(out, func(a, b model.NodeUsage) int { return strings.Compare(a.Name, b.Name) })
	return out
}
