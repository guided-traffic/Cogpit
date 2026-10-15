package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/helm"
	"github.com/guided-traffic/cogpit/internal/model"
)

// cluster.metricsStack and cluster.setMetricsStack: Cogpit Metrics (Cluster Settings › Cogpit
// Metrics, D-61), Cogpit's counterpart of Lens's Lens Metrics page: three switches (Prometheus,
// kube-state-metrics, node-exporter) and Apply, which installs the charts Cogpit bundles as one
// Helm release each in the namespace cogpit-metrics, or removes them. An install is mutating and
// a removal destructive (privilege-footprint.md); an agent's request always waits for the user,
// as a Helm install does (A2). Each release is recorded in the Helm journal before the SDK runs
// (D-11), so an install or an uninstall a crash interrupts is listed by helm.pendingOperations and
// shown by the Helm banner; the page shows each release's state and Apply finishes what is left.

type metricsStackParams struct {
	Prometheus       *bool `json:"prometheus,omitempty" jsonschema:"install (true) or remove (false) Prometheus, release cogpit-prometheus; omitted: as it is"`
	KubeStateMetrics *bool `json:"kubeStateMetrics,omitempty" jsonschema:"install (true) or remove (false) kube-state-metrics, release cogpit-kube-state-metrics; omitted: as it is"`
	NodeExporter     *bool `json:"nodeExporter,omitempty" jsonschema:"install (true) or remove (false) node-exporter, release cogpit-node-exporter (a pod on every node on the host's network, port 9100); omitted: as it is"`
}

// want is the switches the parameters set.
func (p metricsStackParams) want() map[helm.MetricsComponent]bool {
	w := map[helm.MetricsComponent]bool{}
	for c, v := range map[helm.MetricsComponent]*bool{helm.MetricsPrometheus: p.Prometheus, helm.MetricsKubeStateMetrics: p.KubeStateMetrics,
		helm.MetricsNodeExporter: p.NodeExporter} {
		if v != nil {
			w[c] = *v
		}
	}
	return w
}

// MetricsStackView is cluster.metricsStack's answer: the state, and with switches given the plan an
// Apply of them would carry out, with the sentence its confirmation shows (the window shows it in
// its own dialog before an install, which the engine does not confirm for the window).
type MetricsStackView struct {
	engine.MetricsStackState
	Plan *MetricsStackPlanView `json:"plan,omitempty"`
}

// MetricsStackPlanView is a plan as cluster.metricsStack reports it.
type MetricsStackPlanView struct {
	Install         []string `json:"install,omitempty"`
	Remove          []string `json:"remove,omitempty"`
	CreateNamespace bool     `json:"createNamespace,omitempty"`
	DeleteNamespace bool     `json:"deleteNamespace,omitempty"`
	// Destructive: the Apply deletes something, and the engine asks every origin to confirm it.
	Destructive bool `json:"destructive,omitempty"`
	// Empty: the switches are as installed; an Apply is refused.
	Empty  bool   `json:"empty,omitempty"`
	Detail string `json:"detail"`
}

// MetricsStackResult is what cluster.setMetricsStack did, and the state after it.
type MetricsStackResult struct {
	Installed        []string                 `json:"installed,omitempty"`
	Removed          []string                 `json:"removed,omitempty"`
	NamespaceCreated bool                     `json:"namespaceCreated,omitempty"`
	NamespaceDeleted bool                     `json:"namespaceDeleted,omitempty"`
	State            engine.MetricsStackState `json:"state"`
}

// The resources the stack's releases are made of (the bundled charts rendered: prometheus 29.36.1,
// TestBundledMetricsChartsRender), which Helm creates with server-side apply and deletes.
var metricsStackObjects = []engine.Verb{
	{Resource: "deployments", Group: stackGroupApps}, {Resource: "daemonsets", Group: stackGroupApps}, {Resource: "services"},
	{Resource: kindConfigMaps}, {Resource: "serviceaccounts"}, {Resource: "persistentvolumeclaims"},
	{Resource: stackClusterRoles, Group: rbacGroup}, {Resource: stackClusterRoleBindings, Group: rbacGroup},
}

// The verbs, groups and resources of Cogpit Metrics' declaration (each area declares its own).
const (
	rbacGroup                = "rbac.authorization.k8s.io"
	stackGroupApps           = "apps"
	stackClusterRoles        = "clusterroles"
	stackClusterRoleBindings = "clusterrolebindings"
	stackVerbCreate          = "create"
	stackVerbPatch           = "patch"
	stackVerbDelete          = "delete"
)

// metricsStackVerbs are cluster.setMetricsStack's: the release records, the namespace, and create,
// patch and delete of the releases' objects.
func metricsStackVerbs() []engine.Verb {
	out := releaseSecretVerbs(nwVerbList, nwVerbGet, stackVerbCreate, "update", stackVerbDelete)
	out = append(out, engine.Verb{Verb: stackVerbCreate, Resource: namespacesResource}, engine.Verb{Verb: stackVerbDelete, Resource: namespacesResource})
	for _, o := range metricsStackObjects {
		for _, v := range []string{stackVerbCreate, stackVerbPatch, stackVerbDelete} {
			x := o
			x.Verb = v
			out = append(out, x)
		}
	}
	return out
}

// namespacesResource is the core namespaces resource.
const namespacesResource = "namespaces"

func registerMetricsStack(g *registrar) {
	add(g, engine.Spec[metricsStackParams]{
		ID: "cluster.metricsStack", Label: "Cogpit Metrics", Danger: engine.DangerRead, Verbs: releaseCacheReads,
		Description: "Returns Cogpit Metrics of the connected cluster (Cluster Settings › Cogpit Metrics, Lens's Lens Metrics): " +
			"the metrics stack Cogpit bundles (prometheus-community's prometheus chart at a pinned version, its SHA-256 " +
			"checked) and, for Prometheus, kube-state-metrics and node-exporter, whether its release is installed in the " +
			"namespace cogpit-metrics, its Helm status, revision and chart version, and whether a Helm action of Cogpit runs " +
			"on it; whether the namespace exists and is Cogpit's; where an install's Prometheus would keep its history (the " +
			"cluster's default StorageClass, else an emptyDir). With switches given (the parameters of " +
			"cluster_setMetricsStack) also the plan an Apply of them would carry out and the sentence its confirmation shows; " +
			"nothing is changed. Read from the cache; cluster_setMetricsStack changes it.",
		Enabled: func(s *engine.Session, _ metricsStackParams) error { return metricsStackPossible(s) },
		Run: func(ctx context.Context, s *engine.Session, p metricsStackParams) (any, error) {
			if len(p.want()) == 0 {
				st, err := s.MetricsStack(ctx)
				return MetricsStackView{MetricsStackState: st}, err
			}
			plan, err := s.PlanMetricsStack(ctx, p.want())
			if err != nil {
				return nil, err
			}
			return MetricsStackView{MetricsStackState: plan.State, Plan: planView(s, plan)}, nil
		},
	})
	add(g, engine.Spec[metricsStackParams]{
		ID: "cluster.setMetricsStack", Label: "Apply Cogpit Metrics", Danger: engine.DangerMutating, Verbs: metricsStackVerbs(),
		Description: "Installs or removes Cogpit Metrics in the connected cluster, as Lens's Lens Metrics page's Apply does: " +
			"each of prometheus, kubeStateMetrics and nodeExporter set true is installed as a Helm release (cogpit-prometheus, " +
			"cogpit-kube-state-metrics, cogpit-node-exporter) in the namespace cogpit-metrics from the charts Cogpit bundles " +
			"(nothing is downloaded; the cluster pulls the images from quay.io and registry.k8s.io); each set false is " +
			"uninstalled, its objects deleted (Prometheus's collected history with it); an omitted one stays as it is. Cogpit " +
			"creates the namespace, labelled as Cogpit's, and deletes it once nothing is left in it. Cluster-wide it creates " +
			"ClusterRoles and ClusterRoleBindings for Prometheus (get, list and watch of nodes, pods, services, endpoints, " +
			"ingresses and configmaps) and kube-state-metrics (list and watch of the cluster's objects, Secrets left out); " +
			"node-exporter runs a pod on every node with the host's network and process namespace on port 9100, and needs a " +
			"namespace whose Pod Security admits that. It needs the rights to create all of it; Cogpit's charts read from the " +
			"Prometheus it installs, which detection prefers to any other. An agent's request always waits for the user's " +
			"acceptance; a removal is destructive. Each release is recorded first, so one a crash interrupts is listed by " +
			"helm_pendingOperations; cluster_metricsStack shows the result.",
		AgentsAsk: func(*engine.Session, metricsStackParams) bool { return true }, // a Helm install by an agent always asks (A2)
		Escalate: func(s *engine.Session, p metricsStackParams) bool {
			plan, err := planOf(s, p)
			return err != nil || plan.Destructive() // a plan that cannot be read asks as the worst case
		},
		Enabled: func(s *engine.Session, p metricsStackParams) error {
			if err := metricsStackPossible(s); err != nil {
				return err
			}
			if s.HelmOps() == nil {
				return engine.ErrNoHelmJournal
			}
			if len(p.want()) == 0 {
				return errors.New("say which component to install or remove: prometheus, kubeStateMetrics, nodeExporter")
			}
			return metricsStackPermitted(s, p)
		},
		Targets: func(s *engine.Session, p metricsStackParams) ([]model.ObjectRef, error) {
			plan, err := planOf(s, p)
			if err != nil {
				return nil, err
			}
			if plan.Empty() {
				return nil, errNothingToApply
			}
			return planTargets(plan), nil
		},
		Detail: func(s *engine.Session, p metricsStackParams) string {
			plan, err := planOf(s, p)
			if err != nil {
				return ""
			}
			return planDetail(s, plan)
		},
		Run: runMetricsStack,
	})
}

// metricsStackPossible refuses where Cogpit cannot install the stack: no cluster, a read-only
// connection, the demo cluster.
func metricsStackPossible(s *engine.Session) error {
	b, err := writable(s)
	if err != nil {
		return err
	}
	if _, err := b.HelmInstaller(); err != nil {
		return engine.ErrNoMetricsStack
	}
	return nil
}

// metricsStackPermitted checks the verbs the access reviews know of, without waiting: creating the
// cluster roles and their bindings for an install of Prometheus or kube-state-metrics, deleting
// them for a removal. What the reviews
// do not know yet is the API server's to decide.
func metricsStackPermitted(s *engine.Session, p metricsStackParams) error {
	b := s.Backend()
	if b == nil || b.Access == nil {
		return nil
	}
	var verbs []string
	for c, on := range p.want() {
		switch {
		case c == helm.MetricsNodeExporter: // its chart creates no cluster role
			continue
		case on:
			verbs = append(verbs, stackVerbCreate)
		default:
			verbs = append(verbs, stackVerbDelete)
		}
	}
	for _, verb := range verbs {
		for _, res := range []string{stackClusterRoles, stackClusterRoleBindings} {
			v := engine.Verb{Verb: verb, Group: rbacGroup, Resource: res}
			if allowed, known := b.Access.Allowed("", v); known && !allowed {
				return fmt.Errorf("%w: you may not %s %s.%s, which Cogpit Metrics needs", engine.ErrNotPermitted, verb, res, rbacGroup)
			}
		}
	}
	return nil
}

// planTimeout bounds the cache reads of a plan made outside a dispatch's context (the
// confirmation's targets and sentence).
const planTimeout = 10 * time.Second

// planOf plans an Apply of the parameters from the cache.
func planOf(s *engine.Session, p metricsStackParams) (engine.MetricsStackPlan, error) {
	ctx, cancel := context.WithTimeout(context.Background(), planTimeout)
	defer cancel()
	return s.PlanMetricsStack(ctx, p.want())
}

// planView reports a plan.
func planView(s *engine.Session, plan engine.MetricsStackPlan) *MetricsStackPlanView {
	v := &MetricsStackPlanView{CreateNamespace: plan.CreateNamespace, DeleteNamespace: plan.DeleteNamespace,
		Destructive: plan.Destructive(), Empty: plan.Empty(), Detail: planDetail(s, plan)}
	for _, c := range plan.Install {
		v.Install = append(v.Install, c.Release())
	}
	for _, c := range plan.Remove {
		v.Remove = append(v.Remove, c.Release())
	}
	return v
}

// errNothingToApply refuses an Apply that changes nothing, before anyone is asked.
var errNothingToApply = errors.New("nothing to apply: Cogpit Metrics is as the switches say")

// planTargets are the objects a plan acts on: the releases it removes and installs, and the
// namespace when it creates or deletes it.
func planTargets(plan engine.MetricsStackPlan) []model.ObjectRef {
	ctxName := plan.State.Context
	var out []model.ObjectRef
	for _, c := range append(append([]helm.MetricsComponent(nil), plan.Remove...), plan.Install...) {
		out = append(out, engine.HelmReleaseRef(ctxName, helm.MetricsNamespace, c.Release()))
	}
	if plan.CreateNamespace || plan.DeleteNamespace {
		out = append(out, model.ObjectRef{Context: ctxName, Kind: namespacesResource, Name: helm.MetricsNamespace})
	}
	return out
}

// sameTargets reports whether two lists name the same objects in the same order.
func sameTargets(a, b []model.ObjectRef) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].Same(b[i]) {
			return false
		}
	}
	return true
}

// planDetail is the confirmation's text: what is installed, from which chart, what it creates
// cluster-wide and where its history goes; what is removed and lost.
func planDetail(s *engine.Session, plan engine.MetricsStackPlan) string {
	var b strings.Builder
	if plan.Empty() {
		return "Nothing changes: the switches are as installed."
	}
	if len(plan.Install) > 0 {
		names := make([]string, len(plan.Install))
		for i, c := range plan.Install {
			names[i] = c.Label() + " (release " + c.Release() + ")"
		}
		fmt.Fprintf(&b, "Installs %s in namespace %s from chart %s %s that Cogpit bundles (prometheus-community, its SHA-256 checked; "+
			"the cluster pulls the images from quay.io and registry.k8s.io), with your identity", strings.Join(names, ", "),
			helm.MetricsNamespace, helm.MetricsChartName, helm.MetricsChartVersion)
		if plan.CreateNamespace {
			b.WriteString(", creating the namespace")
		}
		b.WriteString(".")
		if scoped := clusterScoped(s, plan); scoped != "" {
			b.WriteString(" Cluster-wide it creates " + scoped + ".")
		}
		for _, c := range plan.Install {
			switch c {
			case helm.MetricsPrometheus:
				b.WriteString(" Prometheus keeps its history on " + plan.Storage.String() + "; Cogpit's charts read from it.")
			case helm.MetricsKubeStateMetrics:
				b.WriteString(" kube-state-metrics may list and watch the cluster's objects, Secrets left out.")
			case helm.MetricsNodeExporter:
				b.WriteString(" node-exporter runs a pod on every node with the host's network and process namespace and its root " +
					"filesystem read-only, on port 9100 of each node: where another node-exporter holds the port, its pods do not " +
					"start, and a namespace whose Pod Security does not admit host namespaces refuses them.")
			}
		}
	}
	for _, c := range plan.Remove {
		fmt.Fprintf(&b, " Uninstalls release %s: its objects and its cluster roles are deleted", c.Release())
		if c == helm.MetricsPrometheus {
			b.WriteString(", the history Prometheus collected with them")
		}
		b.WriteString(".")
	}
	if plan.DeleteNamespace {
		fmt.Fprintf(&b, " Deletes namespace %s, which Cogpit created, and whatever is left in it.", helm.MetricsNamespace)
	}
	return strings.TrimSpace(b.String())
}

// clusterScoped names the cluster-scoped objects the plan's installs create, rendered offline
// from the bundled charts (helm template): "ClusterRole cogpit-prometheus-server, …".
func clusterScoped(s *engine.Session, plan engine.MetricsStackPlan) string {
	b := s.Backend()
	if b == nil {
		return ""
	}
	installer, err := b.HelmInstaller()
	if err != nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), planTimeout)
	defer cancel()
	var out []string
	for _, c := range plan.Install {
		ch, err := helm.BundledMetricsChart(c)
		if err != nil {
			continue
		}
		pre, err := installer.PreviewInstall(ctx, helm.InstallRequest{Namespace: helm.MetricsNamespace, Name: c.Release(), Chart: ch,
			Values: helm.MetricsValues(c, plan.Storage), Offline: true, Driver: s.HelmDriver()})
		if err != nil {
			continue
		}
		for _, o := range pre.Objects {
			if o.Namespace == "" {
				out = append(out, o.Kind+" "+o.Name)
			}
		}
	}
	return strings.Join(out, ", ")
}

// runMetricsStack carries a plan out: the removals first, then the namespace, the installs, and the
// namespace's deletion when nothing is left. Each release goes through the Helm journal; the first
// failure stops what follows.
func runMetricsStack(ctx context.Context, s *engine.Session, p metricsStackParams) (any, error) {
	b, err := writable(s)
	if err != nil {
		return nil, err
	}
	if err := engine.CheckConfirmedContext(ctx, b.Context); err != nil {
		return nil, err
	}
	plan, err := s.PlanMetricsStack(ctx, p.want())
	if err != nil {
		return nil, err
	}
	if plan.State.Running() {
		return nil, errors.New("a Helm action on Cogpit Metrics is running; nothing was done")
	}
	if plan.Empty() {
		return nil, errNothingToApply
	}
	// What the user accepted is what runs: a plan that changed since (another Cogpit, the user in a
	// terminal) is refused, as a release that moved is.
	if req, ok := engine.Confirmed(ctx); ok && !sameTargets(req.Objects, planTargets(plan)) {
		return nil, fmt.Errorf("%w: Cogpit Metrics changed since you confirmed; nothing was done: apply again", engine.ErrTargetsChanged)
	}
	res := MetricsStackResult{}
	for _, c := range plan.Remove {
		if err := uninstallComponent(ctx, s, b, c); err != nil {
			return nil, err
		}
		res.Removed = append(res.Removed, c.Release())
	}
	if plan.CreateNamespace {
		if err := createMetricsNamespace(ctx, b); err != nil {
			return nil, err
		}
		res.NamespaceCreated = true
	}
	for _, c := range plan.Install {
		if err := installComponent(ctx, s, b, c, plan.Storage); err != nil {
			return nil, forCaller(ctx, "install", c.Release(), err)
		}
		res.Installed = append(res.Installed, c.Release())
	}
	if plan.DeleteNamespace {
		deleted, err := deleteMetricsNamespace(ctx, s, b)
		if err != nil {
			return nil, err
		}
		res.NamespaceDeleted = deleted
	}
	res.State, _ = s.WaitMetricsStack(ctx, plan)
	return res, nil
}

// installComponent installs one component's release, recorded in the Helm journal first.
func installComponent(ctx context.Context, s *engine.Session, b *engine.Backend, c helm.MetricsComponent, storage helm.MetricsStorage) error {
	installer, err := b.HelmInstaller()
	if err != nil {
		return err
	}
	ch, err := helm.BundledMetricsChart(c)
	if err != nil {
		return err
	}
	op := engine.PendingHelmOp{
		Context: b.Context, Namespace: helm.MetricsNamespace, Release: c.Release(), Action: engine.HelmInstall,
		Chart: ch.Repository + "/" + ch.Name, ChartVersion: ch.Version, Started: s.Now().UTC(), StartedBy: startedBy(ctx),
	}
	return runHelmOp(ctx, s, op, fmt.Sprintf("Installing %s (release %s) in namespace %s…", c.Label(), c.Release(), helm.MetricsNamespace), "", func(ctx context.Context) error {
		_, err := installer.InstallChart(ctx, helm.InstallRequest{Namespace: helm.MetricsNamespace, Name: c.Release(), Chart: ch,
			Values: helm.MetricsValues(c, storage), Driver: s.HelmDriver()})
		return err
	})
}

// uninstallComponent uninstalls one component's release from the cache's record of it, recorded in
// the Helm journal first.
func uninstallComponent(ctx context.Context, s *engine.Session, b *engine.Backend, c helm.MetricsComponent) error {
	w, err := b.HelmWriter()
	if err != nil {
		return err
	}
	ref := engine.HelmReleaseRef(b.Context, helm.MetricsNamespace, c.Release())
	obj, err := s.Object(ctx, ref)
	if err != nil {
		return err
	}
	op := engine.PendingHelmOp{
		Context: b.Context, Namespace: helm.MetricsNamespace, Release: c.Release(), Action: engine.HelmUninstall,
		FromRevision: helm.ReleaseRevision(obj), Started: s.Now().UTC(), StartedBy: startedBy(ctx),
	}
	return runHelmOp(ctx, s, op, fmt.Sprintf("Removing %s (release %s)…", c.Label(), c.Release()), "", func(ctx context.Context) error {
		return w.UninstallRelease(ctx, helm.MetricsNamespace, c.Release(), helm.ReleaseStorage(obj))
	})
}

// namespacesGVR is the core namespaces resource.
var namespacesGVR = schema.GroupVersionResource{Version: "v1", Resource: namespacesResource}

// createMetricsNamespace creates the stack's namespace with Cogpit's labels; one that exists
// meanwhile is taken as it is.
func createMetricsNamespace(ctx context.Context, b *engine.Backend) error {
	w, err := b.ObjectWriter()
	if err != nil {
		return err
	}
	if _, err := w.CreateObject(ctx, namespacesGVR, "", engine.MetricsNamespaceObject(), false); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("create namespace %s: %w", helm.MetricsNamespace, err)
	}
	return nil
}

// deleteMetricsNamespace deletes the stack's namespace when it is still Cogpit's (its labels, read
// again from the cache), bound to its UID, and reports whether it did.
func deleteMetricsNamespace(ctx context.Context, s *engine.Session, b *engine.Backend) (bool, error) {
	ns, err := s.Object(ctx, model.ObjectRef{Context: b.Context, Kind: namespacesResource, Name: helm.MetricsNamespace})
	switch {
	case errors.Is(err, engine.ErrNotFound):
		return false, nil
	case err != nil:
		return false, err
	}
	if !engine.CogpitsMetricsNamespace(ns) {
		return false, nil // not Cogpit's (any more): left alone
	}
	w, err := b.ObjectWriter()
	if err != nil {
		return false, err
	}
	if err := w.DeleteObject(ctx, namespacesGVR, "", helm.MetricsNamespace, string(ns.GetUID())); err != nil && !apierrors.IsNotFound(err) {
		return false, fmt.Errorf("delete namespace %s: %w", helm.MetricsNamespace, err)
	}
	userToast(s, engine.LevelInfo, "Namespace "+helm.MetricsNamespace+" deleted")
	return true, nil
}
