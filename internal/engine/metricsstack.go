package engine

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/helm"
	"github.com/guided-traffic/cogpit/internal/metrics"
	"github.com/guided-traffic/cogpit/internal/model"
)

// Cogpit Metrics (Cluster Settings › Cogpit Metrics, D-61; Lens: Cluster Settings › Lens Metrics,
// three switches and Apply, captured): the metrics stack Cogpit bundles (internal/helm's
// metricsstack.go: prometheus-community's prometheus chart at a pinned version, its digest
// checked), installed on request as one Helm release per switch in Cogpit's own namespace
// cogpit-metrics. Nothing here writes: the state is read from the cache (the releases, the
// namespace, the StorageClasses), and a plan says what an Apply does. The command
// (commands/metricsstack.go) runs the plan through the Helm SDK, each release recorded in the Helm
// journal first (D-11), and the engine's metrics detection finds the Prometheus it installed
// (metrics.CogpitPrometheusService, ranked first).

// Labels of the namespace Cogpit creates for the stack: it deletes the namespace with the last
// component only when it carries them, never one the user made.
const (
	labelManagedBy        = "app.kubernetes.io/managed-by"
	labelPartOf           = "app.kubernetes.io/part-of"
	metricsStackManagedBy = "cogpit"
	metricsStackPartOf    = "cogpit-metrics"
)

// annotations that mark a cluster's default StorageClass (Kubernetes honours the beta one still).
const (
	defaultClassAnnotation     = "storageclass.kubernetes.io/is-default-class"
	defaultClassAnnotationBeta = "storageclass.beta.kubernetes.io/is-default-class"
)

// MetricsStackComponent is one switch of the page: its release in cogpit-metrics and what the cache
// shows of it.
type MetricsStackComponent struct {
	// ID is the component (helm.MetricsComponents): prometheus, kube-state-metrics, node-exporter.
	ID      string `json:"id"`
	Label   string `json:"label"`
	Release string `json:"release"`
	// Installed: a release of that name is in the namespace, in whatever state (Status).
	Installed bool `json:"installed"`
	// Status is the release's Helm status (deployed, pending-install, failed, uninstalling …).
	Status       string `json:"status,omitempty"`
	Revision     int    `json:"revision,omitempty"`
	ChartVersion string `json:"chartVersion,omitempty"`
	// Running: a Helm action of this Cogpit runs on the release now.
	Running bool `json:"running,omitempty"`
}

// MetricsStackState is the page's state.
type MetricsStackState struct {
	Context   string `json:"context"`
	Namespace string `json:"namespace"`
	// Chart is the bundle: "prometheus 29.36.1", from MetricsChartRepository; ChartDigest its SHA-256.
	Chart       string                  `json:"chart"`
	Repository  string                  `json:"repository"`
	ChartDigest string                  `json:"chartDigest"`
	Components  []MetricsStackComponent `json:"components"`
	// NamespaceExists and NamespaceCogpits: the namespace is there, and Cogpit made it (its labels):
	// it goes with the last component.
	NamespaceExists  bool `json:"namespaceExists"`
	NamespaceCogpits bool `json:"namespaceCogpits"`
	// Storage is where an install's Prometheus would keep its history: the cluster's default
	// StorageClass, or an emptyDir; StorageText says it for the page.
	Storage     helm.MetricsStorage `json:"storage"`
	StorageText string              `json:"storageText"`
	// Known: the cache could read the releases and the namespace; else Reason says why, and the
	// switches show nothing reliable.
	Known  bool   `json:"known"`
	Reason string `json:"reason,omitempty"`
}

// Component returns the state of one component.
func (st MetricsStackState) Component(c helm.MetricsComponent) MetricsStackComponent {
	for _, x := range st.Components {
		if x.ID == string(c) {
			return x
		}
	}
	return MetricsStackComponent{ID: string(c), Label: c.Label(), Release: c.Release()}
}

// ErrNoMetricsStack refuses the stack where Cogpit cannot install it: the demo cluster.
var ErrNoMetricsStack = fmt.Errorf("%w: this cluster connection cannot install charts (the demo cluster's metrics are its own)", ErrNotApplicable)

// MetricsStack returns the stack's state from the cache, waiting for the releases', the
// namespace's and the StorageClasses' first lists (bounded). It sends nothing but the cache's own
// lists and watches.
func (s *Session) MetricsStack(ctx context.Context, contextID string) (MetricsStackState, error) {
	b, err := s.TargetBackend(contextID) // the context's cluster, "" the shown one (D-69)
	if err != nil {
		return MetricsStackState{}, err
	}
	if _, err := b.HelmInstaller(); err != nil {
		return MetricsStackState{}, ErrNoMetricsStack
	}
	st := MetricsStackState{Context: b.Context, Namespace: helm.MetricsNamespace, Known: true,
		Chart: helm.MetricsChartName + " " + helm.MetricsChartVersion, Repository: helm.MetricsChartRepository, ChartDigest: helm.MetricsChartSHA256}
	unknown := func(err error) {
		if st.Known {
			st.Known, st.Reason = false, err.Error()
		}
	}
	for _, c := range helm.MetricsComponents {
		x := MetricsStackComponent{ID: string(c), Label: c.Label(), Release: c.Release(),
			Running: s.HelmOps().Running(b.Context, helm.MetricsNamespace, c.Release())}
		switch obj, err := s.Object(ctx, HelmReleaseRef(b.Context, helm.MetricsNamespace, c.Release())); {
		case err == nil:
			x.Installed, x.Status, x.Revision = true, helm.ReleaseStatus(obj), helm.ReleaseRevision(obj)
			_, x.ChartVersion = helm.ReleaseChart(obj)
		case !errors.Is(err, ErrNotFound):
			unknown(err)
		}
		st.Components = append(st.Components, x)
	}
	switch ns, err := s.Object(ctx, metricsNamespaceRef(b.Context)); {
	case err == nil:
		st.NamespaceExists, st.NamespaceCogpits = true, CogpitsMetricsNamespace(ns)
	case !errors.Is(err, ErrNotFound):
		unknown(err)
	}
	st.Storage = s.metricsStorage(ctx, b.Context)
	st.StorageText = st.Storage.String()
	return st, nil
}

// metricsNamespaceRef is the stack's namespace.
func metricsNamespaceRef(context string) model.ObjectRef {
	return model.ObjectRef{Context: context, Kind: namespacesKind, Name: helm.MetricsNamespace}
}

// CogpitsMetricsNamespace reports whether a namespace carries the labels Cogpit gives the one it
// creates for the stack.
func CogpitsMetricsNamespace(ns *unstructured.Unstructured) bool {
	l := ns.GetLabels()
	return l[labelManagedBy] == metricsStackManagedBy && l[labelPartOf] == metricsStackPartOf
}

// namespaceObjectKind is a namespace's kind.
const namespaceObjectKind = "Namespace"

// MetricsNamespaceObject is the namespace Cogpit creates for the stack, labelled as Cogpit's.
func MetricsNamespaceObject() *unstructured.Unstructured {
	ns := &unstructured.Unstructured{}
	ns.SetAPIVersion("v1")
	ns.SetKind(namespaceObjectKind)
	ns.SetName(helm.MetricsNamespace)
	ns.SetLabels(map[string]string{labelManagedBy: metricsStackManagedBy, labelPartOf: metricsStackPartOf})
	return ns
}

// metricsStorage is where an install's Prometheus would keep its history: the cluster's default
// StorageClass from the cache, else emptyDir (none, or the classes cannot be read).
func (s *Session) metricsStorage(ctx context.Context, contextID string) helm.MetricsStorage {
	objs, err := s.ObjectsIn(ctx, contextID, "storageclasses", nil)
	if err != nil {
		return helm.MetricsStorage{}
	}
	for _, o := range objs {
		a := o.GetAnnotations()
		if a[defaultClassAnnotation] == "true" || a[defaultClassAnnotationBeta] == "true" {
			policy := model.NestedString(o.Object, "reclaimPolicy")
			if policy == "" {
				policy = "Delete"
			}
			return helm.MetricsStorage{StorageClass: o.GetName(), ReclaimPolicy: policy}
		}
	}
	return helm.MetricsStorage{}
}

// MetricsStackPlan is what an Apply does: the components to install and to remove, and the
// namespace to create or delete.
type MetricsStackPlan struct {
	Install []helm.MetricsComponent `json:"install,omitempty"`
	Remove  []helm.MetricsComponent `json:"remove,omitempty"`
	// CreateNamespace: the namespace does not exist and an install needs it; Cogpit creates it
	// with its labels. DeleteNamespace: no component is left and the namespace is Cogpit's.
	CreateNamespace bool                `json:"createNamespace,omitempty"`
	DeleteNamespace bool                `json:"deleteNamespace,omitempty"`
	Storage         helm.MetricsStorage `json:"storage"`
	// State is what the plan was made from.
	State MetricsStackState `json:"-"`
}

// Empty reports whether the plan changes nothing.
func (p MetricsStackPlan) Empty() bool {
	return len(p.Install) == 0 && len(p.Remove) == 0 && !p.CreateNamespace && !p.DeleteNamespace
}

// Destructive reports whether the plan deletes something: a release (its objects, Prometheus's
// history) or the namespace.
func (p MetricsStackPlan) Destructive() bool { return len(p.Remove) > 0 || p.DeleteNamespace }

// PlanMetricsStack plans an Apply: want says which components shall be installed; a component it
// leaves out keeps its state. A component switched on that has a release (in whatever state) is left
// alone, one switched off that has one is removed; the namespace Cogpit created goes when nothing is
// left in its keeping, also when an earlier Apply was interrupted after the last uninstall. Nothing
// is planned while the state is not known.
func (s *Session) PlanMetricsStack(ctx context.Context, contextID string, want map[helm.MetricsComponent]bool) (MetricsStackPlan, error) {
	st, err := s.MetricsStack(ctx, contextID)
	if err != nil {
		return MetricsStackPlan{}, err
	}
	if !st.Known {
		return MetricsStackPlan{State: st}, fmt.Errorf("cannot read what is installed in %s: %s", st.Namespace, st.Reason)
	}
	p := MetricsStackPlan{State: st, Storage: st.Storage}
	left := 0
	for _, c := range helm.MetricsComponents {
		x := st.Component(c)
		on, set := want[c]
		if !set {
			on = x.Installed
		}
		switch {
		case on && !x.Installed:
			p.Install = append(p.Install, c)
		case !on && x.Installed:
			p.Remove = append(p.Remove, c)
		}
		if on {
			left++
		}
	}
	p.CreateNamespace = len(p.Install) > 0 && !st.NamespaceExists
	p.DeleteNamespace = left == 0 && st.NamespaceExists && st.NamespaceCogpits
	return p, nil
}

// Running reports whether a Helm action of this Cogpit runs on one of the stack's releases.
func (st MetricsStackState) Running() bool {
	for _, c := range st.Components {
		if c.Running {
			return true
		}
	}
	return false
}

// WaitMetricsStack waits (bounded by ctx and 15 s) until the cache shows the stack as a plan left
// it: the installed releases there, the removed ones gone. It returns the state then, or the last
// one read.
func (s *Session) WaitMetricsStack(ctx context.Context, p MetricsStackPlan) (MetricsStackState, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	for {
		st, err := s.MetricsStack(ctx, p.State.Context)
		if err != nil {
			return st, err
		}
		done := true
		for _, c := range p.Install {
			done = done && st.Component(c).Installed
		}
		for _, c := range p.Remove {
			done = done && !st.Component(c).Installed
		}
		if done || ctx.Err() != nil {
			return st, nil
		}
		select {
		case <-ctx.Done():
			return st, nil
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// MetricsStackNames checks that the bundle's names are the ones metrics detection ranks first: two
// packages of one layer name them, and a test holds them together.
func MetricsStackNames() error {
	if helm.MetricsNamespace != metrics.CogpitPrometheusNamespace ||
		!strings.HasPrefix(metrics.CogpitPrometheusService, helm.MetricsPrometheus.Release()) {
		return fmt.Errorf("the bundle installs %s/%s, detection looks for %s/%s", helm.MetricsNamespace,
			helm.MetricsPrometheus.Release(), metrics.CogpitPrometheusNamespace, metrics.CogpitPrometheusService)
	}
	return nil
}
