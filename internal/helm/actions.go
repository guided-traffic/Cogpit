package helm

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"helm.sh/helm/v4/pkg/action"
	"helm.sh/helm/v4/pkg/kube"

	"github.com/guided-traffic/cogpit/internal/cluster"
)

// Defaults of the release actions, those of the helm CLI (inferred: Lens's are not captured).
const (
	// DefaultTimeout bounds the wait for a release's hooks (helm --timeout 5m0s).
	DefaultTimeout = 5 * time.Minute
	// rollbackHistory is how many revisions a rollback keeps (helm rollback --history-max 10):
	// the SDK deletes the oldest release Secrets beyond it.
	rollbackHistory = 10
)

// Actions runs Helm release actions with the Helm SDK in Cogpit's process (D-3), on one
// connection's own client configuration (NewActionConfig: its credentials, rate limiter and
// request counter, no kubeconfig file). The SDK reads and writes the release's Secrets itself
// (list, get, create, update, delete secrets in the release's namespace) and changes the release's
// objects with whatever verbs their kinds need: those requests are part of the action, not reads
// of the cache.
type Actions struct {
	// config returns the SDK's action configuration for a namespace.
	config func(namespace string) (*action.Configuration, error)
	// Timeout bounds hooks; zero means DefaultTimeout.
	Timeout time.Duration
}

// NewActions returns the release actions on a connection.
func NewActions(conn *cluster.Conn, log *slog.Logger) *Actions {
	return &Actions{config: func(namespace string) (*action.Configuration, error) {
		if conn == nil {
			return nil, errors.New("no cluster connection")
		}
		return NewActionConfig(conn.RESTConfig(), namespace, log)
	}}
}

func (a *Actions) timeout() time.Duration {
	if a.Timeout > 0 {
		return a.Timeout
	}
	return DefaultTimeout
}

// Rollback rolls a release back to an earlier revision, as helm rollback <name> <revision> does
// with its defaults: a new revision with that revision's chart, values and manifest, the objects
// updated to it, hooks run and waited for (not the objects' readiness), history kept to ten
// revisions. Until it finishes the new revision is pending-rollback; a process that dies before
// leaves it so (D-11). The Helm SDK takes no context: the action runs to its end or its timeout.
func (a *Actions) Rollback(namespace, name string, revision int) (err error) {
	defer recoverAction(&err, "rollback", namespace, name)
	cfg, err := a.config(namespace)
	if err != nil {
		return fmt.Errorf("helm rollback %s/%s: %w", namespace, name, err)
	}
	rb := action.NewRollback(cfg)
	rb.Version = revision
	rb.WaitStrategy = kube.HookOnlyStrategy
	rb.Timeout = a.timeout()
	rb.MaxHistory = rollbackHistory
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
	cfg, err := a.config(namespace)
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

// recoverAction turns a panic in the SDK into the action's error: the goroutine boundary of a
// cluster write (D-11).
func recoverAction(err *error, what, namespace, name string) {
	if r := recover(); r != nil {
		*err = fmt.Errorf("helm %s %s/%s: internal error: %v", what, namespace, name, r)
	}
}
