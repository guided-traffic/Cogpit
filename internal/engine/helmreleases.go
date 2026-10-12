package engine

import (
	"context"
	"errors"

	"github.com/guided-traffic/cogpit/internal/helm"
)

// Helm releases (G11, architecture §8). The Releases view and drawer read the cache through the
// virtual kind helm.NewReleaseProfile makes from the Secrets labelled owner=helm; the connector
// offers it on every cluster that serves Secrets. Rollback, uninstall and upgrade run the Helm SDK
// in Cogpit's process, recorded in a journal first so that one a crash interrupts is offered again
// (helmops.go, D-11); the upgrade's values come from a Helm Upgrade dock tab (helmupgrade.go).

// HelmWriter runs the Helm release actions. The Helm SDK on the backend's connection implements
// it; tests put a fake in Backend.Actions, which the engine reaches with a type assertion on
// Backend.Writer, as it reaches ObjectWriter.
type HelmWriter interface {
	// RollbackRelease rolls a release back to an earlier revision (helm rollback); from is the
	// newest revision the cache showed, and a release the server holds at another one is refused
	// (helm.ErrReleaseMoved with helm.ErrUnchanged).
	RollbackRelease(ctx context.Context, namespace, name string, revision, from int) error
	// UninstallRelease uninstalls a release, its objects and its history (helm uninstall).
	UninstallRelease(ctx context.Context, namespace, name string) error
	// UpgradeRelease upgrades a release to a chart version with new values (helm upgrade) and
	// returns the revision it made. Errors never quote a value.
	UpgradeRelease(ctx context.Context, namespace, name string, up HelmUpgradeRequest) (revision int, err error)
}

// HelmUpgradeRequest is what an upgrade installs: the chart version (empty: the release's own; the
// only one there is without chart repositories) and the values, which replace the release's
// user-supplied values whole.
type HelmUpgradeRequest struct {
	ChartVersion string
	Values       map[string]any
	// FromRevision is the revision the values are based on, as the cache showed it: a release the
	// server holds at another one is refused (helm.ErrReleaseMoved with helm.ErrUnchanged).
	FromRevision int
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

func (h sdkHelm) RollbackRelease(_ context.Context, namespace, name string, revision, from int) error {
	return h.a.Rollback(namespace, name, revision, from)
}

func (h sdkHelm) UninstallRelease(_ context.Context, namespace, name string) error {
	return h.a.Uninstall(namespace, name)
}

func (h sdkHelm) UpgradeRelease(ctx context.Context, namespace, name string, up HelmUpgradeRequest) (int, error) {
	return h.a.Upgrade(ctx, namespace, name, up.ChartVersion, up.FromRevision, up.Values)
}
