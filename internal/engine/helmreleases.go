package engine

import (
	"context"
	"errors"

	"github.com/guided-traffic/cogpit/internal/helm"
)

// Helm releases (G11, architecture §8). The Releases view and drawer read the cache through the
// virtual kind helm.NewReleaseProfile makes from the records labelled owner=helm, Secrets and
// ConfigMaps (helmdriver.go); the connector offers it on every cluster that serves Secrets.
// Rollback, uninstall and upgrade run the Helm SDK in Cogpit's process with the storage driver the
// release's records are in, recorded in a journal first so that one a crash interrupts is offered
// again (helmops.go, D-11); the upgrade's values come from a Helm Upgrade dock tab (helmupgrade.go).

// HelmWriter runs the Helm release actions. The Helm SDK on the backend's connection implements
// it; tests put a fake in Backend.Actions, which the engine reaches with a type assertion on
// Backend.Writer, as it reaches ObjectWriter. driver is the storage driver the release's records
// are in (helm.ReleaseStorage): the SDK reads and writes that driver's records only.
type HelmWriter interface {
	// RollbackRelease rolls a release back to an earlier revision (helm rollback); from is the
	// newest revision the cache showed, and a release the server holds at another one is refused
	// (helm.ErrReleaseMoved with helm.ErrUnchanged).
	RollbackRelease(ctx context.Context, namespace, name string, driver helm.Driver, revision, from int) error
	// UninstallRelease uninstalls a release, its objects and its history (helm uninstall).
	UninstallRelease(ctx context.Context, namespace, name string, driver helm.Driver) error
	// UpgradeRelease upgrades a release to a chart version with new values (helm upgrade) and
	// returns the revision it made. Errors never quote a value.
	UpgradeRelease(ctx context.Context, namespace, name string, up HelmUpgradeRequest) (revision int, err error)
}

// HelmUpgradeRequest is what an upgrade installs: the chart version (empty: the release's own),
// the chart downloaded from a repository for it (nil: the release's own chart, which must then be
// at ChartVersion) and the values, which replace the release's user-supplied values whole.
type HelmUpgradeRequest struct {
	ChartVersion string
	// Chart is the chart a repository serves at ChartVersion, downloaded and checked against its
	// index's digest (helmupgrade.go); nil upgrades with the release's own chart.
	Chart  *helm.Chart
	Values map[string]any
	// FromRevision is the revision the values are based on, as the cache showed it: a release the
	// server holds at another one is refused (helm.ErrReleaseMoved with helm.ErrUnchanged).
	FromRevision int
	// Driver is the storage driver the release's records are in (helm.ReleaseStorage; zero:
	// helm.DriverSecret).
	Driver helm.Driver
}

// ErrNoHelmWriter is returned when the connected backend cannot run Helm actions: the demo cluster.
var ErrNoHelmWriter = errors.New("this cluster connection cannot run Helm actions")

// HelmWriter returns what runs the backend's Helm actions: the writer in Backend.Actions if it
// implements HelmWriter, else the Helm SDK on the backend's connection.
func (b *Backend) HelmWriter() (HelmWriter, error) {
	w := b.Writer()
	if hw, ok := w.(HelmWriter); ok {
		return hw, nil
	}
	if w == nil || b.Conn == nil {
		return nil, ErrNoHelmWriter
	}
	return sdkHelm{helm.NewActions(b.Conn, b.logger())}, nil
}

// sdkHelm adapts the Helm SDK's actions to HelmWriter. Rollback and uninstall take no context in
// the SDK; the upgrade does not pass its cancellation on (helm.Actions.Upgrade): an action runs to
// its end or its timeout (helm.DefaultTimeout).
type sdkHelm struct{ a *helm.Actions }

// on returns the actions with the release's storage driver.
func (h sdkHelm) on(driver helm.Driver) *helm.Actions {
	a := *h.a
	a.Driver = driver
	return &a
}

func (h sdkHelm) RollbackRelease(_ context.Context, namespace, name string, driver helm.Driver, revision, from int) error {
	return h.on(driver).Rollback(namespace, name, revision, from)
}

func (h sdkHelm) UninstallRelease(_ context.Context, namespace, name string, driver helm.Driver) error {
	return h.on(driver).Uninstall(namespace, name)
}

func (h sdkHelm) UpgradeRelease(ctx context.Context, namespace, name string, up HelmUpgradeRequest) (int, error) {
	a := h.on(up.Driver)
	if up.Chart != nil {
		return a.UpgradeChart(ctx, namespace, name, up.Chart, up.FromRevision, up.Values)
	}
	return a.Upgrade(ctx, namespace, name, up.ChartVersion, up.FromRevision, up.Values)
}
