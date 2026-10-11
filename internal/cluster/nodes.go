package cluster

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes"
	policyv1client "k8s.io/client-go/kubernetes/typed/policy/v1"
	"k8s.io/kubectl/pkg/drain"
)

// DrainOptions are a drain's options: kubectl drain's flags.
type DrainOptions struct {
	// IgnoreDaemonSets leaves DaemonSet-managed pods running (--ignore-daemonsets). Without it a
	// node that runs any is not drained.
	IgnoreDaemonSets bool
	// DeleteEmptyDirData evicts pods with emptyDir volumes, whose data is lost
	// (--delete-emptydir-data). Without it a node that runs any is not drained.
	DeleteEmptyDirData bool
	// Force evicts pods that no controller manages, which nothing recreates (--force). Without it a
	// node that runs any is not drained.
	Force bool
	// GracePeriodSeconds overrides each evicted pod's termination grace period; negative uses the
	// pod's own (kubectl's --grace-period=-1, the default). Zero would end the pods at once.
	GracePeriodSeconds int
	// Timeout bounds the whole drain; zero leaves the bound to ctx.
	Timeout time.Duration
}

// DrainStep says what a DrainProgress reports.
type DrainStep string

// Drain steps, in the order a drain reports them.
const (
	// DrainCordoned: the node is unschedulable.
	DrainCordoned DrainStep = "cordoned"
	// DrainListed: the pods to evict are known; Total counts them, Message lists the pods left
	// running (DaemonSet pods, mirror pods) when there are any.
	DrainListed DrainStep = "listed"
	// DrainEvicting: the eviction of Pod was requested.
	DrainEvicting DrainStep = "evicting"
	// DrainBlocked: the API server refused Pod's eviction for now (a PodDisruptionBudget allows
	// no disruption; Message says which); the drain retries until it times out.
	DrainBlocked DrainStep = "blocked"
	// DrainEvicted: Pod is gone.
	DrainEvicted DrainStep = "evicted"
)

// DrainProgress is one step of a drain, reported while it runs.
type DrainProgress struct {
	Step    DrainStep `json:"step"`
	Node    string    `json:"node"`
	Pod     string    `json:"pod,omitempty"` // namespace/name
	Message string    `json:"message,omitempty"`
	Done    int       `json:"done"`  // pods evicted so far
	Total   int       `json:"total"` // pods to evict; known from DrainListed on
}

// Tunables of the drain; tests shorten them.
var (
	// evictRetryDelay is the pause after an eviction the API server refused for now (HTTP 429),
	// as kubectl drain's default.
	evictRetryDelay = 5 * time.Second
	// drainChunkSize pages the pod list, as kubectl's --chunk-size default.
	drainChunkSize int64 = 500
)

// SetNodeUnschedulable cordons (true) or uncordons (false) a node with the strategic merge patch
// kubectl cordon and uncordon send: spec.unschedulable true, or null to remove it.
func (c *Conn) SetNodeUnschedulable(ctx context.Context, name string, unschedulable bool) error {
	var value any // null removes the field, as kubectl uncordon's patch does
	verb := "uncordon"
	if unschedulable {
		value, verb = true, "cordon"
	}
	patch, err := json.Marshal(map[string]any{"spec": map[string]any{"unschedulable": value}})
	if err != nil {
		return fmt.Errorf("%s node %s: %w", verb, name, err)
	}
	if _, err := c.kube.CoreV1().Nodes().Patch(ctx, name, types.StrategicMergePatchType, patch,
		metav1.PatchOptions{FieldManager: FieldManager}); err != nil {
		return fmt.Errorf("%s node %s: %w", verb, name, err)
	}
	return nil
}

// DrainNode drains a node as kubectl drain does, with kubectl's drain library: it cordons the
// node, lists the node's pods, refuses (before evicting anything) if a pod may not be evicted
// under opts, then evicts every evictable pod through the eviction API, which honours
// PodDisruptionBudgets, retrying evictions a budget blocks, and waits until the pods are gone.
//
// API requests: patch nodes; get on the discovery document /api/v1 (eviction support); list
// pods (field selector spec.nodeName, all namespaces); get daemonsets for DaemonSet-managed pods;
// create pods/eviction per pod; get pods while retrying and waiting. Unlike the library, it never
// falls back to deleting pods: a server without the eviction API is refused.
//
// A drain that fails, times out or is cancelled leaves the node cordoned and partly drained, as
// kubectl's does; the caller records that (the engine's drain journal). progress, if not nil, is
// called for each step, one call at a time, from the library's goroutines.
func (c *Conn) DrainNode(ctx context.Context, name string, opts DrainOptions, progress func(DrainProgress)) error {
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}
	rep := &drainReporter{node: name, fn: progress, started: map[string]bool{}, blocked: map[string]string{}}

	if err := c.SetNodeUnschedulable(ctx, name, true); err != nil {
		return fmt.Errorf("drain node %s: %w", name, err)
	}
	rep.report(DrainProgress{Step: DrainCordoned})

	gv, err := drain.CheckEvictionSupport(c.kube)
	if err != nil {
		return fmt.Errorf("drain node %s: checking for the eviction API: %w", name, err)
	}
	if gv.Empty() {
		return fmt.Errorf("drain node %s: the API server offers no eviction API; Cogpit drains only by eviction, "+
			"which honours PodDisruptionBudgets", name)
	}

	h := &drain.Helper{
		Ctx:                  ctx,
		Client:               evictionRecorder{Interface: c.kube, rep: rep},
		Force:                opts.Force,
		GracePeriodSeconds:   opts.GracePeriodSeconds,
		IgnoreAllDaemonSets:  opts.IgnoreDaemonSets,
		DeleteEmptyDirData:   opts.DeleteEmptyDirData,
		Timeout:              opts.Timeout,
		ChunkSize:            drainChunkSize,
		EvictErrorRetryDelay: evictRetryDelay,
		Out:                  io.Discard, // progress comes through the callbacks
		ErrOut:               io.Discard,
		OnPodDeletionOrEvictionStarted: func(pod *corev1.Pod, _ bool) {
			rep.evicting(pod.Namespace + "/" + pod.Name)
		},
		OnPodDeletionOrEvictionFinished: func(pod *corev1.Pod, _ bool, err error) {
			if err == nil {
				rep.evicted(pod.Namespace + "/" + pod.Name)
			}
		},
	}
	list, errs := h.GetPodsForDeletion(name)
	if len(errs) > 0 {
		return fmt.Errorf("drain node %s: %w", name, errors.New(cogpitFlags(utilerrors.NewAggregate(errs).Error())))
	}
	pods := list.Pods()
	rep.listed(len(pods), cogpitFlags(list.Warnings()))
	if err := h.DeleteOrEvictPods(pods); err != nil {
		return fmt.Errorf("drain node %s: %s%w", name, drainCause(ctx, opts.Timeout), rep.explain(err))
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("drain node %s: %s%w", name, drainCause(ctx, opts.Timeout), err)
	}
	return nil
}

// drainCause names why a drain's context ended, if it did: the timeout or a cancellation.
func drainCause(ctx context.Context, timeout time.Duration) string {
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded) && timeout > 0:
		return fmt.Sprintf("timed out after %s: ", timeout)
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return "timed out: "
	case errors.Is(ctx.Err(), context.Canceled):
		return "cancelled: "
	}
	return ""
}

// cogpitFlags names the parameters of Cogpit's node.drain where the drain library's messages name
// kubectl's flags.
func cogpitFlags(msg string) string {
	return flagHints.Replace(msg)
}

var flagHints = strings.NewReplacer(
	"(use --ignore-daemonsets to ignore)", "(set ignoreDaemonSets to leave them running)",
	"(use --delete-emptydir-data to override)", "(set deleteEmptyDirData to evict them; their emptyDir data is lost)",
	"(use --force to override)", "(set force to evict them; nothing recreates them)",
)

// drainReporter turns the drain library's callbacks into DrainProgress calls, one at a time, and
// remembers why evictions were blocked, for the error of a drain that does not finish.
type drainReporter struct {
	mu      sync.Mutex
	node    string
	fn      func(DrainProgress)
	total   int
	done    int
	started map[string]bool   // pods whose eviction was requested
	blocked map[string]string // pod → the last reason its eviction was refused
}

// report calls fn with the node and the counters filled in. A panicking fn is the caller's bug; it
// must not end a drain halfway, so it is recovered.
func (r *drainReporter) report(p DrainProgress) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reportLocked(p)
}

func (r *drainReporter) reportLocked(p DrainProgress) {
	if r.fn == nil {
		return
	}
	p.Node, p.Done, p.Total = r.node, r.done, r.total
	defer func() { _ = recover() }()
	r.fn(p)
}

func (r *drainReporter) listed(total int, warnings string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.total = total
	r.reportLocked(DrainProgress{Step: DrainListed, Message: warnings})
}

func (r *drainReporter) evicting(pod string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.started[pod] { // the library calls this again for every retry
		return
	}
	r.started[pod] = true
	r.reportLocked(DrainProgress{Step: DrainEvicting, Pod: pod})
}

func (r *drainReporter) evicted(pod string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.done++
	delete(r.blocked, pod)
	r.reportLocked(DrainProgress{Step: DrainEvicted, Pod: pod})
}

// observe records an eviction's outcome: a refusal because of a PodDisruptionBudget is reported
// once per reason and kept for the drain's error.
func (r *drainReporter) observe(pod string, err error) {
	var blocked *EvictionBlockedError
	if !errors.As(explainEviction(pod, err), &blocked) {
		return
	}
	reason := blocked.reason()
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.blocked[pod] == reason {
		return
	}
	r.blocked[pod] = reason
	r.reportLocked(DrainProgress{Step: DrainBlocked, Pod: pod, Message: reason})
}

// explain adds the blocked evictions' reasons to the error of a drain that did not finish: the
// library's own error says only that the timeout was reached.
func (r *drainReporter) explain(err error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.blocked) == 0 {
		return err
	}
	pods := make([]string, 0, len(r.blocked))
	for p := range r.blocked {
		pods = append(pods, p)
	}
	sort.Strings(pods)
	reasons := make([]string, 0, len(pods))
	for _, p := range pods {
		reasons = append(reasons, "pod "+p+": "+r.blocked[p])
	}
	return fmt.Errorf("%w; evictions blocked: %s", err, strings.Join(reasons, "; "))
}

// evictionRecorder is the clientset the drain library gets: every policy/v1 eviction's outcome is
// observed, so that a drain can say which PodDisruptionBudget held it up. Everything else is the
// connection's clientset.
type evictionRecorder struct {
	kubernetes.Interface
	rep *drainReporter
}

// PolicyV1 returns the policy/v1 client with recorded evictions.
func (c evictionRecorder) PolicyV1() policyv1client.PolicyV1Interface {
	return recordingPolicyV1{PolicyV1Interface: c.Interface.PolicyV1(), rep: c.rep}
}

type recordingPolicyV1 struct {
	policyv1client.PolicyV1Interface
	rep *drainReporter
}

// Evictions returns the namespace's eviction client with recorded outcomes.
func (p recordingPolicyV1) Evictions(namespace string) policyv1client.EvictionInterface {
	return recordingEvictions{EvictionInterface: p.PolicyV1Interface.Evictions(namespace), rep: p.rep}
}

type recordingEvictions struct {
	policyv1client.EvictionInterface
	rep *drainReporter
}

// Evict evicts and records the outcome.
func (e recordingEvictions) Evict(ctx context.Context, eviction *policyv1.Eviction) error {
	err := e.EvictionInterface.Evict(ctx, eviction)
	if err != nil && eviction != nil {
		e.rep.observe(eviction.Namespace+"/"+eviction.Name, err)
	}
	return err
}
