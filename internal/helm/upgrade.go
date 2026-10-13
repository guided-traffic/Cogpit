package helm

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"helm.sh/helm/v4/pkg/action"
	chart "helm.sh/helm/v4/pkg/chart/v2"
	"helm.sh/helm/v4/pkg/kube"
	rspb "helm.sh/helm/v4/pkg/release/v1"
	"k8s.io/apimachinery/pkg/util/version"
)

// Upgrading a release (G11, architecture §8): helm upgrade with the helm CLI's defaults, in
// Cogpit's process, either to the release's own chart (the one its newest revision holds: new
// values, the same chart version) or to a chart version downloaded from a repository the user
// added (Fetcher.FetchChart: bounded, checked against the index's digest when it lists one).

// ErrNoRepository refuses a chart version other than the release's own when no chart was
// downloaded for it: without a repository that lists the version, the release's own chart, taken
// from its newest revision, is the only one Cogpit has.
var ErrNoRepository = errors.New("no chart repository added to Cogpit lists this chart version; without one only the release's own chart version can be installed")

// ErrOtherChart refuses a downloaded chart whose name is not the release's chart's: Cogpit offers a
// repository's versions of the release's own chart, matched by name, and upgrades to nothing else.
var ErrOtherChart = errors.New("the chart is not the release's chart")

// Upgrade upgrades a release to its own chart (the chart its newest revision holds) with new
// values, as helm upgrade <name> <chart> --version <v> -f values does with its defaults: a new
// revision pending-upgrade, the objects changed to its manifest, hooks run and waited for (not the
// objects' readiness), history kept to ten revisions (Lens's). chartVersion, when not empty, must
// be the release's own (ErrNoRepository otherwise); UpgradeChart takes another version.
//
// The values replace the user-supplied ones whole (--reset-values): what the editor holds is what
// the release gets, and an empty editor is no user-supplied values, where the helm CLI would keep
// the old ones. A release left pending by another action is refused by the SDK.
//
// ctx is the dispatch's context, but its cancellation does not reach the SDK: the SDK's upgrade
// marks a release failed when its context ends while its own goroutine keeps changing the
// cluster, so the upgrade runs to its end or its timeout, as rollback and uninstall do. Every
// error and every line the SDK logs is scrubbed of the values (ScrubValues). An error that left
// the release as it was (refused before the SDK made a revision: a template or schema that
// rejects the values, a pending release, another chart version) matches ErrUnchanged. It
// returns the new revision.
//
// from is the revision the values are based on (the caller's cache's newest): when the server
// holds another one, the upgrade is refused with ErrReleaseMoved and ErrUnchanged before the SDK
// runs, so a cache that lags never lets the values replace a revision nobody reviewed; 0 checks
// nothing. The SDK reads the newest revision again itself, a window of milliseconds, not of a
// watch's lag.
func (a *Actions) Upgrade(ctx context.Context, namespace, name, chartVersion string, from int, values map[string]any) (revision int, err error) {
	return a.upgrade(ctx, namespace, name, nil, chartVersion, from, values)
}

// UpgradeChart upgrades a release to a chart downloaded from a repository (Fetcher.FetchChart), as
// helm upgrade <name> <repository>/<chart> --version <v> -f values does: the same defaults,
// values, binding to from, scrubbing and ErrUnchanged as Upgrade. The chart must be the release's
// own by name (ErrOtherChart) and installable (not a library chart, its dependencies packaged with
// it); its version may be newer or older than the release's. Its digest was checked when it was
// downloaded.
func (a *Actions) UpgradeChart(ctx context.Context, namespace, name string, ch *Chart, from int, values map[string]any) (revision int, err error) {
	if ch == nil || ch.chart == nil {
		return 0, fmt.Errorf("helm upgrade %s/%s: no chart; %w", namespace, name, ErrUnchanged)
	}
	return a.upgrade(ctx, namespace, name, ch, ch.Version, from, values)
}

// upgrade runs an upgrade to ch, or with ch nil to the release's own chart at chartVersion.
func (a *Actions) upgrade(ctx context.Context, namespace, name string, ch *Chart, chartVersion string, from int, values map[string]any) (revision int, err error) {
	defer func() { err = Scrubbed(err, values) }() // last: also scrubs a panic's text
	defer recoverAction(&err, "upgrade", namespace, name)
	unchanged := func(err error) error {
		return fmt.Errorf("helm upgrade %s/%s: %w; %w", namespace, name, err, ErrUnchanged)
	}
	cfg, err := a.config(namespace, a.driver(), slog.New(scrubHandler{next: a.logger().Handler(), values: values}))
	if err != nil {
		return 0, unchanged(err)
	}
	rls, err := lastRelease(cfg, name)
	if err != nil {
		return 0, unchanged(err)
	}
	if err := movedFrom(rls, from); err != nil {
		return 0, unchanged(err)
	}
	target, err := upgradeChart(rls, ch, chartVersion)
	if err != nil {
		return 0, unchanged(err)
	}
	up := action.NewUpgrade(cfg)
	up.Namespace = namespace
	up.WaitStrategy = kube.HookOnlyStrategy
	up.Timeout = a.timeout()
	up.MaxHistory = MaxHistory
	up.ResetValues = true
	res, err := up.RunWithContext(context.WithoutCancel(ctx), name, target, values)
	if err != nil {
		if now, lerr := lastRelease(cfg, name); lerr == nil && now.Version == rls.Version {
			return 0, unchanged(err) // the SDK refused before it made a revision
		}
		return 0, fmt.Errorf("helm upgrade %s/%s: %w", namespace, name, err)
	}
	if out, ok := res.(*rspb.Release); ok && out != nil {
		return out.Version, nil
	}
	return rls.Version + 1, nil
}

// upgradeChart is the chart an upgrade installs: the release's own at its version (chartVersion
// empty or that version), or a downloaded chart of the release's chart's name that helm can
// install.
func upgradeChart(rls *rspb.Release, ch *Chart, chartVersion string) (*chart.Chart, error) {
	own := rls.Chart.Metadata
	if ch == nil {
		if chartVersion != "" && chartVersion != own.Version {
			return nil, fmt.Errorf("chart %s %s: %w (the release's own is %s)", own.Name, chartVersion, ErrNoRepository, own.Version)
		}
		return rls.Chart, nil
	}
	if ch.Name != own.Name {
		return nil, fmt.Errorf("chart %s/%s %s: %w (%s)", ch.Repository, ch.Name, ch.Version, ErrOtherChart, own.Name)
	}
	if err := ch.checkInstallable(); err != nil {
		return nil, err
	}
	return ch.chart, nil
}

// CompareVersions compares two chart versions as semantic versions (helm's chart versions: a
// leading "v" allowed, a prerelease older than its release): -1, 0 or 1, and ok false when either
// does not parse.
func CompareVersions(a, b string) (cmp int, ok bool) {
	va, err := version.Parse(a)
	if err != nil {
		return 0, false
	}
	vb, err := version.Parse(b)
	if err != nil {
		return 0, false
	}
	switch {
	case va.LessThan(vb):
		return -1, true
	case vb.LessThan(va):
		return 1, true
	}
	return 0, true
}
