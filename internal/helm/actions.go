package helm

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"helm.sh/helm/v4/pkg/action"
	"helm.sh/helm/v4/pkg/kube"
	rspb "helm.sh/helm/v4/pkg/release/v1"

	"github.com/guided-traffic/cogpit/internal/cluster"
)

// Defaults of the release actions, those of the helm CLI (inferred: Lens's are not captured),
// except the history an upgrade keeps, which is Lens's (captured).
const (
	// DefaultTimeout bounds the wait for a release's hooks (helm --timeout 5m0s).
	DefaultTimeout = 5 * time.Minute
	// MaxHistory is how many revisions a rollback or an upgrade keeps (helm --history-max 10;
	// Lens keeps ten, captured 2026-10-12: 12 revisions before a rollback, 10 after): the SDK
	// deletes the oldest release Secrets beyond it.
	MaxHistory = 10
)

// ErrNoRepository refuses a chart version other than the release's own: without chart
// repositories the release's own chart, taken from its newest revision, is the only one Cogpit has.
var ErrNoRepository = errors.New("without chart repositories only the release's own chart version can be installed")

// Actions runs Helm release actions with the Helm SDK in Cogpit's process (D-3), on one
// connection's own client configuration (NewActionConfig: its credentials, rate limiter and
// request counter, no kubeconfig file). The SDK reads and writes the release's Secrets itself
// (list, get, create, update, delete secrets in the release's namespace) and changes the release's
// objects with whatever verbs their kinds need: those requests are part of the action, not reads
// of the cache.
type Actions struct {
	// config returns the SDK's action configuration for a namespace, logging to log (nil: the
	// connection's logger).
	config func(namespace string, log *slog.Logger) (*action.Configuration, error)
	// log is the logger the SDK logs to.
	log *slog.Logger
	// Timeout bounds hooks; zero means DefaultTimeout.
	Timeout time.Duration
}

// NewActions returns the release actions on a connection.
func NewActions(conn *cluster.Conn, log *slog.Logger) *Actions {
	if log == nil {
		log = slog.Default()
	}
	return &Actions{log: log, config: func(namespace string, l *slog.Logger) (*action.Configuration, error) {
		if conn == nil {
			return nil, errors.New("no cluster connection")
		}
		return NewActionConfig(conn.RESTConfig(), namespace, l)
	}}
}

func (a *Actions) timeout() time.Duration {
	if a.Timeout > 0 {
		return a.Timeout
	}
	return DefaultTimeout
}

func (a *Actions) logger() *slog.Logger {
	if a.log == nil {
		return slog.Default()
	}
	return a.log
}

// Rollback rolls a release back to an earlier revision, as helm rollback <name> <revision> does
// with its defaults: a new revision with that revision's chart, values and manifest, the objects
// updated to it, hooks run and waited for (not the objects' readiness), history kept to ten
// revisions. Until it finishes the new revision is pending-rollback; a process that dies before
// leaves it so (D-11). The Helm SDK takes no context: the action runs to its end or its timeout.
// A release left pending-upgrade by an interrupted upgrade can be rolled back (the SDK's rollback
// does not refuse a pending release).
//
// from is the revision the caller saw as the newest (its cache's): when the server holds another
// one, the rollback is refused with ErrReleaseMoved and ErrUnchanged before anything changes; 0
// checks nothing.
func (a *Actions) Rollback(namespace, name string, revision, from int) (err error) {
	defer recoverAction(&err, "rollback", namespace, name)
	cfg, err := a.config(namespace, a.logger())
	if err != nil {
		return fmt.Errorf("helm rollback %s/%s: %w", namespace, name, err)
	}
	if from > 0 {
		rls, err := lastRelease(cfg, name)
		if err == nil {
			err = movedFrom(rls, from)
		}
		if err != nil {
			return fmt.Errorf("helm rollback %s/%s: %w; %w", namespace, name, err, ErrUnchanged)
		}
	}
	rb := action.NewRollback(cfg)
	rb.Version = revision
	rb.WaitStrategy = kube.HookOnlyStrategy
	rb.Timeout = a.timeout()
	rb.MaxHistory = MaxHistory
	if err := rb.Run(name); err != nil {
		return fmt.Errorf("helm rollback %s/%s to revision %d: %w", namespace, name, revision, err)
	}
	return nil
}

// Uninstall uninstalls a release, as helm uninstall <name> does with its defaults: hooks run, the
// release's objects deleted (background propagation), its history deleted. Until it finishes the
// release is uninstalling; a process that dies before leaves it so (D-11). The Helm SDK takes no
// context: the action runs to its end or its timeout.
func (a *Actions) Uninstall(namespace, name string) (err error) {
	defer recoverAction(&err, "uninstall", namespace, name)
	cfg, err := a.config(namespace, a.logger())
	if err != nil {
		return fmt.Errorf("helm uninstall %s/%s: %w", namespace, name, err)
	}
	un := action.NewUninstall(cfg)
	un.WaitStrategy = kube.HookOnlyStrategy
	un.Timeout = a.timeout()
	un.DeletionPropagation = "background"
	if _, err := un.Run(name); err != nil {
		return fmt.Errorf("helm uninstall %s/%s: %w", namespace, name, err)
	}
	return nil
}

// Upgrade upgrades a release to its own chart (the chart its newest revision holds: Cogpit has no
// chart repositories yet) with new values, as helm upgrade <name> <chart> --version <v> -f values
// does with its defaults: a new revision pending-upgrade, the objects changed to its manifest,
// hooks run and waited for (not the objects' readiness), history kept to ten revisions (Lens's).
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
	defer func() { err = Scrubbed(err, values) }() // last: also scrubs a panic's text
	defer recoverAction(&err, "upgrade", namespace, name)
	unchanged := func(err error) error {
		return fmt.Errorf("helm upgrade %s/%s: %w; %w", namespace, name, err, ErrUnchanged)
	}
	cfg, err := a.config(namespace, slog.New(scrubHandler{next: a.logger().Handler(), values: values}))
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
	if v := rls.Chart.Metadata.Version; chartVersion != "" && chartVersion != v {
		return 0, unchanged(fmt.Errorf("chart %s %s: %w (%s)", rls.Chart.Metadata.Name, chartVersion, ErrNoRepository, v))
	}
	up := action.NewUpgrade(cfg)
	up.Namespace = namespace
	up.WaitStrategy = kube.HookOnlyStrategy
	up.Timeout = a.timeout()
	up.MaxHistory = MaxHistory
	up.ResetValues = true
	res, err := up.RunWithContext(context.WithoutCancel(ctx), name, rls.Chart, values)
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

// ErrUnchanged marks an upgrade that failed before it changed the release: no revision was made,
// and nothing is left to finish or roll back.
var ErrUnchanged = errors.New("the release is unchanged")

// ErrReleaseMoved marks an action refused because the server holds another newest revision than
// the one the caller based it on: its cache had not seen a newer one yet.
var ErrReleaseMoved = errors.New("the release moved")

// movedFrom fails with ErrReleaseMoved when rls is not at revision from (0: no check).
func movedFrom(rls *rspb.Release, from int) error {
	if from > 0 && rls.Version != from {
		return fmt.Errorf("%w: it is at revision %d on the server, the action is based on revision %d", ErrReleaseMoved, rls.Version, from)
	}
	return nil
}

// lastRelease returns a release's newest revision with its chart, as the SDK reads it (a list of
// the release's Secrets: part of the action).
func lastRelease(cfg *action.Configuration, name string) (*rspb.Release, error) {
	last, err := cfg.Releases.Last(name)
	if err != nil {
		return nil, err
	}
	rls, ok := last.(*rspb.Release)
	if !ok || rls == nil || rls.Chart == nil || rls.Chart.Metadata == nil {
		return nil, errors.New("the release's newest revision holds no chart")
	}
	return rls, nil
}

// recoverAction turns a panic in the SDK into the action's error: the goroutine boundary of a
// cluster write (D-11).
func recoverAction(err *error, what, namespace, name string) {
	if r := recover(); r != nil {
		*err = fmt.Errorf("helm %s %s/%s: internal error: %v", what, namespace, name, r)
	}
}
