package metrics

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
)

// Service is what detection needs to know about a cached Service: where it is and which ports it
// has. The engine fills it from the cache's Services; no request is made for it.
type Service struct {
	Namespace string
	Name      string
	Ports     []ServicePort
}

// ServicePort is one port of a Service.
type ServicePort struct {
	Name string
	Port int32
}

// ServiceSource lists the cluster's Services, from the cache: detection never lists Services
// itself. An error is a state ("unavailable"), not a failure.
type ServiceSource interface {
	Services(ctx context.Context) ([]Service, error)
}

// ServiceSourceFunc adapts a function to a ServiceSource.
type ServiceSourceFunc func(ctx context.Context) ([]Service, error)

// Services calls f.
func (f ServiceSourceFunc) Services(ctx context.Context) ([]Service, error) { return f(ctx) }

// Override names the Prometheus to use instead of detecting one: set in a cluster's settings. Port
// may be a number or a port name; Prefix is the route prefix, if Prometheus runs under one.
type Override struct {
	Namespace string `json:"namespace"`
	Service   string `json:"service"`
	Port      string `json:"port"`
	Prefix    string `json:"prefix,omitempty"`
}

// Valid reports whether the override names a service and a port.
func (o Override) Valid() bool { return o.Namespace != "" && o.Service != "" && o.Port != "" }

// Candidate is a Service that may be a Prometheus, with the rank detection gave it.
type Candidate struct {
	Target Target
	Score  int
}

// Detection of Prometheus by well-known Service names, observed from the documentation of the
// charts and of Lens's metrics providers (clean-room, D-12):
//
//	score 100  <release>-kube-prometheus-prometheus      kube-prometheus-stack
//	            kube-prometheus-stack-prometheus
//	score  95  prometheus-k8s                            kube-prometheus manifests
//	score  90  prometheus-operated                       the Prometheus Operator's governing service
//	score  80  prometheus-server                         the prometheus Helm chart
//	score  70  prometheus in monitoring, prometheus, lens-metrics (Lens's own install, a
//	           hand-written install)
//	score  40  prometheus in any other namespace
//
// Ties go to the namespaces named in preferredNamespaces, then by name. A name that is not one of
// these is never tried: probing every Service of a cluster would send a request each.
var promNames = []struct {
	re    *regexp.Regexp
	score int
}{
	{regexp.MustCompile(`^(.+-)?kube-prometheus-prometheus$`), 100},
	{regexp.MustCompile(`^kube-prometheus-stack-prometheus$`), 100},
	{regexp.MustCompile(`^prometheus-k8s$`), 95},
	{regexp.MustCompile(`^prometheus-operated$`), 90},
	{regexp.MustCompile(`^prometheus-server$`), 80},
}

const (
	nsMonitoring = "monitoring"
	nsPrometheus = "prometheus"
	nsLens       = "lens-metrics"
	promService  = "prometheus"
)

var preferredNamespaces = []string{nsMonitoring, nsPrometheus, nsLens}

// MaxProbes bounds how many candidates one detection probes.
const MaxProbes = 3

// webPortNames are the names the charts give the HTTP port, best first.
var webPortNames = []string{"http-web", "web", "http", "http-prometheus", "prometheus", "tcp-prometheus"}

// Candidates ranks the Services that may be a Prometheus, best first. It is pure.
func Candidates(services []Service) []Candidate {
	var out []Candidate
	for _, s := range services {
		score := scoreName(s.Namespace, s.Name)
		if score == 0 {
			continue
		}
		port, ok := pickPort(s.Ports)
		if !ok {
			continue
		}
		out = append(out, Candidate{Target: Target{Namespace: s.Namespace, Service: s.Name, Port: port}, Score: score})
	}
	slices.SortFunc(out, func(a, b Candidate) int {
		if a.Score != b.Score {
			return b.Score - a.Score
		}
		if pa, pb := nsRank(a.Target.Namespace), nsRank(b.Target.Namespace); pa != pb {
			return pa - pb
		}
		return strings.Compare(a.Target.String(), b.Target.String())
	})
	return out
}

func scoreName(namespace, name string) int {
	for _, n := range promNames {
		if n.re.MatchString(name) {
			return n.score
		}
	}
	if name == promService {
		if slices.Contains(preferredNamespaces, namespace) {
			return 70
		}
		return 40
	}
	return 0
}

func nsRank(ns string) int {
	if i := slices.Index(preferredNamespaces, ns); i >= 0 {
		return i
	}
	return len(preferredNamespaces)
}

// pickPort chooses the port Prometheus's web UI and API listen on: one named like the charts name
// it, else 9090, else 80, else the first. Ports that belong to sidecars (the config reloader, the
// Thanos sidecar's gRPC) are skipped by name. The port is returned as its name when it has one
// of the known names (the service proxy resolves names), else as its number.
func pickPort(ports []ServicePort) (string, bool) {
	var usable []ServicePort
	for _, p := range ports {
		n := strings.ToLower(p.Name)
		if strings.Contains(n, "reloader") || strings.Contains(n, "grpc") || strings.Contains(n, "thanos") {
			continue
		}
		usable = append(usable, p)
	}
	for _, want := range webPortNames {
		for _, p := range usable {
			if strings.EqualFold(p.Name, want) {
				return p.Name, true
			}
		}
	}
	for _, want := range []int32{9090, 80} {
		for _, p := range usable {
			if p.Port == want {
				return strconv.Itoa(int(p.Port)), true
			}
		}
	}
	if len(usable) > 0 {
		return strconv.Itoa(int(usable[0].Port)), true
	}
	return "", false
}

// PrometheusDetector finds a Prometheus: the override if one is set, else the best candidates among
// the cached Services, each probed with one `GET .../api/v1/status/buildinfo` through the service
// proxy. It is the Detector for Prometheus and answers "not served" when there is none, "forbidden"
// when the user may not use the service proxy, and "unavailable" when a candidate exists but does
// not answer.
type PrometheusDetector struct {
	px       proxy
	services ServiceSource
	opts     PrometheusOptions

	mu       sync.Mutex
	override *Override
}

// NewPrometheusDetector returns the detector for a cluster: services is the cache's list of
// Services, kube reaches the service proxy through the connection's counting transport.
func NewPrometheusDetector(kube kubernetes.Interface, services ServiceSource, opts PrometheusOptions) *PrometheusDetector {
	return &PrometheusDetector{px: kubeProxy{kube}, services: services, opts: opts}
}

// PrometheusLabel names Prometheus in the status line before one is found.
const PrometheusLabel = "Prometheus"

// ID implements Identified: the detector looks for "prometheus".
func (d *PrometheusDetector) ID() string { return PrometheusID }

// Label implements Identified.
func (d *PrometheusDetector) Label() string { return PrometheusLabel }

// SetOverride sets (or, with nil, clears) the Prometheus to use instead of detection. It takes
// effect at the next detection: call Hub.Forget to run it at once.
func (d *PrometheusDetector) SetOverride(o *Override) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if o != nil && !o.Valid() {
		o = nil
	}
	if o != nil {
		c := *o
		o = &c
	}
	d.override = o
}

// Override returns the override in effect, or nil when Prometheus is detected.
func (d *PrometheusDetector) Override() *Override {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.override == nil {
		return nil
	}
	c := *d.override
	return &c
}

// probe is the request that tells a Prometheus from any other service.
const probePath = "/api/v1/status/buildinfo"

// Detect implements Detector.
func (d *PrometheusDetector) Detect(ctx context.Context) (Provider, State) {
	d.mu.Lock()
	ov := d.override
	d.mu.Unlock()

	var cands []Candidate
	if ov != nil {
		cands = []Candidate{{Target: Target{Namespace: ov.Namespace, Service: ov.Service, Port: ov.Port, Prefix: ov.Prefix}, Score: 1000}}
	} else {
		if d.services == nil {
			return nil, State{Status: StatusNotServed, Message: "no Prometheus found"}
		}
		svcs, err := d.services.Services(ctx)
		if err != nil {
			return nil, State{Status: StatusUnavailable, Message: "could not read the cluster's services: " + err.Error()}
		}
		cands = Candidates(svcs)
	}
	if len(cands) == 0 {
		return nil, State{Status: StatusNotServed, Message: "no Prometheus found (no service named like one in the cluster)"}
	}
	if len(cands) > MaxProbes {
		cands = cands[:MaxProbes]
	}
	var lastErr error
	var lastTarget Target
	for _, c := range cands {
		err := d.probe(ctx, c.Target)
		if err == nil {
			prov := newPrometheus(d.px, c.Target, d.opts)
			return prov, State{Status: StatusAvailable, Provider: PrometheusID, Label: prov.Label()}
		}
		if ctx.Err() != nil {
			return nil, State{Status: StatusUnavailable, Message: msgCancelled}
		}
		if apierrors.IsForbidden(err) || apierrors.IsUnauthorized(err) {
			return nil, State{Status: StatusForbidden, Message: "the current user may not use the service proxy of " + c.Target.String() + " (get services/proxy), so Prometheus cannot be read: " + errorText(err), Reason: errorClass(err)}
		}
		if !errors.Is(err, errNotPrometheus) && !apierrors.IsNotFound(err) {
			lastErr, lastTarget = err, c.Target
		}
	}
	if lastErr != nil {
		return nil, State{Status: StatusUnavailable, Message: fmt.Sprintf("%s looks like a Prometheus but does not answer: %s", lastTarget, errorText(lastErr)), Reason: errorClass(lastErr)}
	}
	return nil, State{Status: StatusNotServed, Message: "no Prometheus found (the candidates do not serve the Prometheus API)"}
}

var errNotPrometheus = errors.New("not a Prometheus")

// probe checks one candidate with buildinfo, which every Prometheus since 2.14 serves.
func (d *PrometheusDetector) probe(ctx context.Context, t Target) error {
	ctx, cancel := context.WithTimeout(ctx, DefaultQueryTimeout)
	defer cancel()
	body, err := d.px.get(ctx, t, probePath, nil, 1<<20)
	if err != nil {
		return err
	}
	if !isBuildInfo(body) {
		return errNotPrometheus
	}
	return nil
}
