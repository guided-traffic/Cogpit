package metrics

import (
	"context"
	"errors"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	apimetrics "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsclient "k8s.io/metrics/pkg/client/clientset/versioned/typed/metrics/v1beta1"

	"github.com/guided-traffic/cogpit/internal/model"
)

// ServerID and ServerLabel name the Kubernetes Metrics Server provider; the label is what Lens's
// drawer prints in "Displaying metrics from Kubernetes Metrics Server".
const (
	ServerID    = "metrics-server"
	ServerLabel = "Kubernetes Metrics Server"
)

// GroupVersion is the Metrics API's group and version.
const GroupVersion = "metrics.k8s.io/v1beta1"

// Server is the Kubernetes Metrics Server provider: the aggregated API metrics.k8s.io/v1beta1,
// which gives the current CPU and memory of every pod (per container) and node, measured by the
// kubelets about every 15 seconds. It also is the Detector for itself.
type Server struct {
	disco  discovery.DiscoveryInterface
	client metricsclient.MetricsV1beta1Interface
}

// NewServer returns the provider for a cluster: disco asks whether metrics.k8s.io is served,
// client reads it. Both go through the cluster's counting transport.
func NewServer(disco discovery.DiscoveryInterface, client metricsclient.MetricsV1beta1Interface) *Server {
	return &Server{disco: disco, client: client}
}

// ID implements Provider.
func (s *Server) ID() string { return ServerID }

// Label implements Provider.
func (s *Server) Label() string { return ServerLabel }

// Capabilities implements Provider: current CPU and memory only.
func (s *Server) Capabilities() Capabilities { return Capabilities{} }

// Detect implements Detector with one discovery request: is metrics.k8s.io/v1beta1 served with
// pods and nodes. A missing group is "not served"; a group that is registered but fails (the
// APIService is there, the metrics-server pod is not) is "unavailable".
func (s *Server) Detect(ctx context.Context) (Provider, State) {
	list, err := discovery.ToDiscoveryInterfaceWithContext(s.disco).ServerResourcesForGroupVersionWithContext(ctx, GroupVersion)
	if err != nil {
		return nil, StateFromError(err, "the cluster serves no metrics.k8s.io API (no metrics-server installed)")
	}
	pods, nodes := false, false
	for _, r := range list.APIResources {
		pods = pods || r.Name == "pods"
		nodes = nodes || r.Name == "nodes"
	}
	if !pods && !nodes {
		return nil, State{Status: StatusNotServed, Message: "metrics.k8s.io serves neither pods nor nodes"}
	}
	return s, State{Status: StatusAvailable, Provider: ServerID, Label: ServerLabel}
}

// StateFromError turns a failed request into the state it means: not found is "not served"
// (notFound is the message for it), forbidden and unauthorized are "forbidden", anything else
// "unavailable" with the error's text. The text is the API server's, never a credential.
func StateFromError(err error, notFound string) State {
	switch {
	case err == nil:
		return State{Status: StatusAvailable}
	case apierrors.IsForbidden(err), apierrors.IsUnauthorized(err):
		return State{Status: StatusForbidden, Message: "the current user may not read the Metrics API: " + errorText(err)}
	case apierrors.IsNotFound(err):
		return State{Status: StatusNotServed, Message: notFound}
	case apierrors.IsServiceUnavailable(err):
		return State{Status: StatusUnavailable, Message: "the Metrics API is registered but not answering (is the metrics-server running?): " + errorText(err)}
	case errors.Is(err, context.Canceled):
		return State{Status: StatusUnavailable, Message: msgCancelled}
	default:
		return State{Status: StatusUnavailable, Message: "the Metrics API failed: " + errorText(err)}
	}
}

func errorText(err error) string {
	var st apierrors.APIStatus
	if errors.As(err, &st) {
		if m := st.Status().Message; m != "" {
			return m
		}
	}
	return err.Error()
}

// ListPods implements Provider.
func (s *Server) ListPods(ctx context.Context, namespace string) ([]model.PodUsage, error) {
	l, err := s.client.PodMetricses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list pod metrics: %w", err)
	}
	out := make([]model.PodUsage, len(l.Items))
	for i := range l.Items {
		out[i] = podUsage(&l.Items[i])
	}
	return out, nil
}

// GetPod implements Provider. A pod without a measurement (not found) is ErrNoData.
func (s *Server) GetPod(ctx context.Context, namespace, name string) (model.PodUsage, error) {
	p, err := s.client.PodMetricses(namespace).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) && isObjectNotFound(err) {
		return model.PodUsage{}, ErrNoData
	}
	if err != nil {
		return model.PodUsage{}, fmt.Errorf("get pod metrics: %w", err)
	}
	return podUsage(p), nil
}

// ListNodes implements Provider.
func (s *Server) ListNodes(ctx context.Context) ([]model.NodeUsage, error) {
	l, err := s.client.NodeMetricses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list node metrics: %w", err)
	}
	out := make([]model.NodeUsage, len(l.Items))
	for i := range l.Items {
		out[i] = nodeUsage(&l.Items[i])
	}
	return out, nil
}

// GetNode implements Provider. A node without a measurement is ErrNoData.
func (s *Server) GetNode(ctx context.Context, name string) (model.NodeUsage, error) {
	n, err := s.client.NodeMetricses().Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) && isObjectNotFound(err) {
		return model.NodeUsage{}, ErrNoData
	}
	if err != nil {
		return model.NodeUsage{}, fmt.Errorf("get node metrics: %w", err)
	}
	return nodeUsage(n), nil
}

// isObjectNotFound tells "this pod or node has no metrics" from "the API is gone": the metrics
// server names the object in its 404 (details.name), the API server's 404 for a missing resource
// does not.
func isObjectNotFound(err error) bool {
	var st apierrors.APIStatus
	if !errors.As(err, &st) {
		return true // a fake client's error carries no details; the request named an object
	}
	d := st.Status().Details
	return d == nil || d.Name != ""
}

func podUsage(p *apimetrics.PodMetrics) model.PodUsage {
	out := model.PodUsage{
		Namespace: p.Namespace, Name: p.Name, Timestamp: p.Timestamp.Time,
		Containers: make([]model.ContainerUsage, len(p.Containers)),
	}
	for i, c := range p.Containers {
		out.Containers[i] = model.ContainerUsage{Name: c.Name, Usage: model.Usage{
			CPUMilli: c.Usage.Cpu().MilliValue(), MemoryBytes: c.Usage.Memory().Value(),
		}}
	}
	return out
}

func nodeUsage(n *apimetrics.NodeMetrics) model.NodeUsage {
	return model.NodeUsage{Name: n.Name, Timestamp: n.Timestamp.Time, Usage: model.Usage{
		CPUMilli: n.Usage.Cpu().MilliValue(), MemoryBytes: n.Usage.Memory().Value(),
	}}
}
