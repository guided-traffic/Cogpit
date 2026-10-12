package helm

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"

	"helm.sh/helm/v4/pkg/action"
	chartutil "helm.sh/helm/v4/pkg/chart/v2/util"
	"helm.sh/helm/v4/pkg/kube"
	rspb "helm.sh/helm/v4/pkg/release/v1"
	"helm.sh/helm/v4/pkg/storage/driver"
	"k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/yaml"
)

// Installing a chart (G11, architecture §8): helm install with the helm CLI's defaults (hooks run
// and waited for, not the objects' readiness; 5 minutes; server-side apply; no --create-namespace
// unless asked; no --atomic), in Cogpit's process on the connection's own client configuration. A
// dry run first renders the chart against the cluster without changing it (helm install
// --dry-run=server): the objects' kinds and names are the preview. An install that a crash
// interrupts leaves the release pending-install (D-11): the engine's journal offers to uninstall
// it.

// InstallRequest is one install: a chart, the release's name and namespace, and the values, which
// become the release's user-supplied values whole (a Helm Install tab holds the chart's
// values.yaml, so they are its defaults with the user's changes).
type InstallRequest struct {
	Namespace       string
	Name            string
	Chart           *Chart
	Values          map[string]any
	CreateNamespace bool
}

// RenderedObject is one object a chart renders, as a preview names it.
type RenderedObject struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	// Namespace is the object's namespace: its manifest's, else the release's; empty for a
	// built-in cluster-scoped kind.
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name"`
}

// InstallPreview is what a dry run of an install renders: the objects, the hooks and the custom
// resource definitions of the chart's crds/ folder (which helm installs before the rest, and which
// an uninstall leaves).
type InstallPreview struct {
	Objects []RenderedObject `json:"objects"`
	Hooks   []RenderedObject `json:"hooks,omitempty"`
	CRDs    []RenderedObject `json:"crds,omitempty"`
}

// CheckReleaseName checks a release name as helm does (a DNS subdomain of at most 53 characters).
func CheckReleaseName(name string) error {
	if err := chartutil.ValidateReleaseName(name); err != nil {
		return fmt.Errorf("release name %q: %w", name, err)
	}
	return nil
}

// CheckNamespace checks a namespace's name (a DNS label).
func CheckNamespace(name string) error {
	if errs := validation.IsDNS1123Label(name); len(errs) > 0 {
		return fmt.Errorf("namespace %q: %s", name, strings.Join(errs, "; "))
	}
	return nil
}

func checkInstall(req InstallRequest) error {
	if err := CheckReleaseName(req.Name); err != nil {
		return err
	}
	if err := CheckNamespace(req.Namespace); err != nil {
		return err
	}
	return req.Chart.checkInstallable()
}

func (a *Actions) newInstall(cfg *action.Configuration, req InstallRequest) *action.Install {
	in := action.NewInstall(cfg)
	in.ReleaseName, in.Namespace = req.Name, req.Namespace
	in.CreateNamespace = req.CreateNamespace
	in.WaitStrategy = kube.HookOnlyStrategy
	in.Timeout = a.timeout()
	return in
}

// PreviewInstall renders an install against the cluster without changing it (the SDK's dry run on
// the server: discovery, the cluster's capabilities, a get of each rendered object to find one that
// exists already and is not the release's, and whatever the chart's templates look up). It returns
// the objects' kinds and names, never their content: a rendered manifest holds the values. Errors
// are scrubbed of the values the user changed.
func (a *Actions) PreviewInstall(ctx context.Context, req InstallRequest) (p InstallPreview, err error) {
	scrub := userValues(req)
	defer func() { err = Scrubbed(err, scrub) }()
	defer recoverAction(&err, "install preview", req.Namespace, req.Name)
	if err := checkInstall(req); err != nil {
		return p, err
	}
	cfg, err := a.config(req.Namespace, slog.New(scrubHandler{next: a.logger().Handler(), values: scrub}))
	if err != nil {
		return p, fmt.Errorf("helm install %s/%s: %w", req.Namespace, req.Name, err)
	}
	in := a.newInstall(cfg, req)
	in.DryRunStrategy = action.DryRunServer
	res, err := in.RunWithContext(ctx, req.Chart.chart, req.Values)
	if err != nil {
		return p, fmt.Errorf("helm install %s/%s (dry run): %w", req.Namespace, req.Name, err)
	}
	rel, ok := res.(*rspb.Release)
	if !ok || rel == nil {
		return p, errors.New("the dry run returned no release")
	}
	p.Objects = renderedObjects(rel.Manifest, req.Namespace)
	for _, h := range rel.Hooks {
		if h != nil {
			p.Hooks = append(p.Hooks, renderedObjects(h.Manifest, req.Namespace)...)
		}
	}
	for _, c := range req.Chart.chart.CRDObjects() {
		if c.File != nil {
			p.CRDs = append(p.CRDs, renderedObjects(string(c.File.Data), "")...)
		}
	}
	if p.Objects == nil {
		p.Objects = []RenderedObject{}
	}
	return p, nil
}

// Install installs a chart as a new release and returns its revision (1). A release of that name
// in the namespace, whatever its state, is refused before anything changes (helm install refuses
// one that is not uninstalled; Cogpit does not reuse a name with kept history either). An error
// that left no release record (refused by the SDK before it stored revision 1: a template or
// schema that refuses the values, an object that exists already, a namespace that does not exist)
// matches ErrUnchanged: nothing is left to uninstall, although custom resource definitions from
// the chart's crds/ folder and a namespace it was asked to create stay, as helm leaves them.
//
// ctx is the dispatch's context, but its cancellation does not reach the SDK, as for the upgrade:
// the install runs to its end or its timeout. Errors and the SDK's log lines are scrubbed of the
// values the user changed from the chart's defaults.
func (a *Actions) Install(ctx context.Context, req InstallRequest) (revision int, err error) {
	scrub := userValues(req)
	defer func() { err = Scrubbed(err, scrub) }() // last: also scrubs a panic's text
	defer recoverAction(&err, "install", req.Namespace, req.Name)
	unchanged := func(err error) error {
		return fmt.Errorf("helm install %s/%s: %w; %w", req.Namespace, req.Name, err, ErrUnchanged)
	}
	if err := checkInstall(req); err != nil {
		return 0, unchanged(err)
	}
	cfg, err := a.config(req.Namespace, slog.New(scrubHandler{next: a.logger().Handler(), values: scrub}))
	if err != nil {
		return 0, unchanged(err)
	}
	switch h, herr := cfg.Releases.History(req.Name); {
	case herr == nil && len(h) > 0:
		return 0, unchanged(fmt.Errorf("a release named %s exists in namespace %s (revision %d)", req.Name, req.Namespace, len(h)))
	case herr != nil && !errors.Is(herr, driver.ErrReleaseNotFound):
		return 0, unchanged(herr)
	}
	res, err := a.newInstall(cfg, req).RunWithContext(context.WithoutCancel(ctx), req.Chart.chart, req.Values)
	if err != nil {
		if h, herr := cfg.Releases.History(req.Name); errors.Is(herr, driver.ErrReleaseNotFound) || (herr == nil && len(h) == 0) {
			return 0, unchanged(err) // the SDK refused before it stored the release
		}
		return 0, fmt.Errorf("helm install %s/%s: %w", req.Namespace, req.Name, err)
	}
	if rel, ok := res.(*rspb.Release); ok && rel != nil {
		return rel.Version, nil
	}
	return 1, nil
}

// renderedObjects names the objects of a rendered manifest.
func renderedObjects(manifest, namespace string) []RenderedObject {
	rs := manifestResources(manifest, namespace)
	out := make([]RenderedObject, 0, len(rs))
	for _, r := range rs {
		out = append(out, RenderedObject{APIVersion: r.apiVersion, Kind: r.kind, Namespace: r.namespace, Name: r.name})
	}
	return out
}

// userValues are the values of an install that are not the chart's own defaults: the ones the user
// typed or changed, which ScrubValues keeps out of errors and logs. The chart's defaults are public
// (they come from the repository), and scrubbing them would garble every message that names an
// image or a service type.
func userValues(req InstallRequest) map[string]any { return req.Chart.UserValues(req.Values) }

// UserValues returns the leaves of values that differ from the chart's defaults: what an install's
// errors and logs are scrubbed of (ScrubValues).
func (c *Chart) UserValues(values map[string]any) map[string]any {
	return valuesDiff(values, c.Defaults())
}

// valuesDiff returns the leaves of values that differ from the same path in defaults.
func valuesDiff(values, defaults map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range values {
		d, ok := defaults[k]
		vm, isMap := v.(map[string]any)
		dm, defIsMap := d.(map[string]any)
		switch {
		case isMap && ok && defIsMap:
			if sub := valuesDiff(vm, dm); len(sub) > 0 {
				out[k] = sub
			}
		case ok && sameValue(v, d):
		default:
			out[k] = v
		}
	}
	return out
}

// sameValue compares two values as YAML reads them (numbers may be int64 or float64).
func sameValue(a, b any) bool {
	if reflect.DeepEqual(a, b) {
		return true
	}
	ya, err1 := yaml.Marshal(a)
	yb, err2 := yaml.Marshal(b)
	return err1 == nil && err2 == nil && string(ya) == string(yb)
}
