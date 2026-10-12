package commands

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/helm"
	"github.com/guided-traffic/cogpit/internal/model"
)

// Helm release actions (G11): rollback and uninstall through the Helm SDK in Cogpit's process,
// both destructive (privilege-footprint.md), confirmed for every origin, bound to the revision the
// user saw, and recorded in the Helm journal before the SDK runs (D-11).

// HelmResult is the result of a release action.
type HelmResult struct {
	Ref      model.ObjectRef `json:"ref"`
	Action   string          `json:"action"`             // "rolledBack", "uninstalled", "upgraded", "dismissed"
	Revision int             `json:"revision,omitempty"` // a rollback's target; the revision an upgrade made
}

type rollbackParams struct {
	Ref      model.ObjectRef `json:"ref" jsonschema:"the release: kind helmreleases, its namespace and name"`
	Revision int             `json:"revision" jsonschema:"a revision of the release's history to roll back to: an earlier one, or the newest (a new revision equal to it); the dialog starts with the one before the newest"`
}

type pendingHelmParams struct {
	All bool `json:"all,omitempty" jsonschema:"list the pending Helm actions of every context, not only the connected one's"`
}

// releaseSecretVerbs are the requests the Helm SDK makes on a release's own Secrets in its
// namespace; the release's objects need whatever verbs their kinds need, which no static list can
// name.
func releaseSecretVerbs(verbs ...string) []engine.Verb {
	out := make([]engine.Verb, 0, len(verbs))
	for _, v := range verbs {
		out = append(out, engine.Verb{Verb: v, Resource: kindSecrets})
	}
	return out
}

func registerHelm(g *registrar) {
	add(g, engine.Spec[rollbackParams]{
		ID: "helm.release.rollback", Label: "Rollback", MenuPath: "Helm/Rollback", Kinds: []string{helm.ReleaseKind},
		Order: 200, Danger: engine.DangerDestructive,
		Verbs: releaseSecretVerbs("list", "get", "create", "update", "delete"),
		Description: "Rolls a Helm release back to an earlier revision of its history, as helm rollback does with its " +
			"defaults (hooks waited for, ten revisions kept): a new revision with that revision's chart, values and " +
			"manifest, and the release's objects changed to it. The target may be the newest revision itself (Lens's dialog " +
			"allows it): helm then writes a new revision equal to it and applies its manifest again. Offered after an upgrade (a release at revision 2 or later). " +
			"Besides the release's Secrets it needs whatever verbs the release's objects need. Refused when the release " +
			"moved to another revision since the user confirmed. A rollback a crash of Cogpit interrupts stays in " +
			"helm_pendingOperations; run it again to finish it. Always asks the user to confirm.",
		Enabled: func(s *engine.Session, p rollbackParams) error {
			obj, err := cachedOfKind(s, p.Ref, helm.ReleaseKind)
			if err != nil {
				return err
			}
			if helm.ReleaseRevision(obj) <= 1 {
				return fmt.Errorf("%w: release %s has no earlier revision (a rollback comes after an upgrade)", engine.ErrNotApplicable, p.Ref.Name)
			}
			return helmWritable(s, p.Ref)
		},
		Targets: func(s *engine.Session, p rollbackParams) ([]model.ObjectRef, error) { return targetOf(s, p.Ref) },
		Prefill: func(s *engine.Session, p rollbackParams) (rollbackParams, error) {
			if p.Revision == 0 {
				if obj, err := s.Lookup(p.Ref); err == nil {
					p.Revision = previousRevision(obj)
				}
			}
			return p, nil
		},
		Detail: func(_ *engine.Session, p rollbackParams) string { return RollbackDetail(p.Revision) },
		Run:    runRollback,
	})

	add(g, engine.Spec[refParams]{
		ID: "helm.release.uninstall", Label: labelDelete, MenuPath: "Helm/Uninstall", Kinds: []string{helm.ReleaseKind},
		Order: engine.OrderDelete, Danger: engine.DangerDestructive,
		Verbs: releaseSecretVerbs("list", "get", "update", "delete"),
		Description: "Uninstalls a Helm release as helm uninstall does with its defaults: runs its hooks, deletes every object " +
			"of the release and its history. Besides the release's Secrets it needs whatever verbs deleting the release's " +
			"objects needs. Refused when the release moved to another revision since the user confirmed. An uninstall a " +
			"crash of Cogpit interrupts stays in helm_pendingOperations; run it again to finish it. Always asks the user to confirm.",
		Enabled: func(s *engine.Session, p refParams) error {
			if _, err := cachedOfKind(s, p.Ref, helm.ReleaseKind); err != nil {
				return err
			}
			return helmWritable(s, p.Ref)
		},
		Targets: func(s *engine.Session, p refParams) ([]model.ObjectRef, error) { return targetOf(s, p.Ref) },
		Detail: func(*engine.Session, refParams) string {
			return "Uninstalling deletes every object of the release and its history."
		},
		Run: runUninstall,
	})

	add(g, engine.Spec[pendingHelmParams]{
		ID: "helm.pendingOperations", Label: "Unfinished Helm Actions", Danger: engine.DangerRead,
		Verbs: releaseCacheReads,
		Description: "Lists the Helm rollbacks, uninstalls, upgrades and installs Cogpit started on the connected context (all: on every " +
			"context) that did not finish: context, namespace, release, action, the target revision of a rollback, the " +
			"revision an upgrade started from (fromRevision) and its chart (an install's as <repository>/<chart>), when and by whom started, status (running in " +
			"this Cogpit or, with elsewhere and its owner's process, in another one, interrupted by an earlier run of Cogpit ending, or failed with its error) and the release's status " +
			"from the cache (pending-install, pending-rollback, pending-upgrade, uninstalling, deployed, missing, unknown). Offer " +
			"helm_release_rollback or helm_release_uninstall again to finish a rollback or an uninstall; an upgrade is " +
			"finished by helm_release_rollback to its fromRevision (the Helm SDK refuses to upgrade a release left " +
			"pending-upgrade); an install by helm_release_uninstall. helm_dismissOperation forgets one.",
		Run: func(ctx context.Context, s *engine.Session, p pendingHelmParams) (any, error) {
			list, err := s.PendingHelmOps(p.All)
			if err != nil {
				return nil, err
			}
			if list == nil {
				list = []engine.PendingHelmOpInfo{}
			}
			if engine.OriginOf(ctx) != engine.OriginGUI {
				for i := range list { // the SDK's text can quote what a chart's lookup read (forCaller)
					if list[i].Error != "" {
						list[i].Error = withheldHelmError
					}
				}
			}
			return list, nil
		},
	})

	add(g, engine.Spec[refParams]{
		ID: "helm.dismissOperation", Label: "Dismiss Unfinished Helm Action", Danger: engine.DangerUI,
		Description: "Forgets an unfinished Helm action from helm_pendingOperations without changing the cluster: the release " +
			"stays as it is (pending-rollback or uninstalling, unless someone changed it). The reference's context " +
			"defaults to the connected one.",
		Enabled: func(s *engine.Session, p refParams) error {
			if s.HelmOps() == nil {
				return engine.ErrNoHelmJournal
			}
			if p.Ref.Kind != helm.ReleaseKind || p.Ref.Namespace == "" || p.Ref.Name == "" {
				return errors.New("name a release: kind helmreleases, its namespace and name")
			}
			return nil
		},
		Run: func(_ context.Context, s *engine.Session, p refParams) (any, error) {
			ref := p.Ref
			if ref.Context == "" {
				b, err := connected(s)
				if err != nil {
					return nil, err
				}
				ref.Context = b.Context
			}
			ops := s.HelmOps()
			if _, ok, err := ops.Pending(ref.Context, ref.Namespace, ref.Name); err != nil {
				return nil, err
			} else if !ok {
				return nil, fmt.Errorf("no unfinished Helm action on release %s/%s in context %s", ref.Namespace, ref.Name, ref.Context)
			}
			if err := ops.Forget(ref.Context, ref.Namespace, ref.Name); err != nil {
				return nil, err
			}
			return HelmResult{Ref: ref, Action: "dismissed"}, nil
		},
	})
}

// helmWritable is the writable check of the release actions: a cluster that can run them, a
// journal to record them in, and no action on the release running.
func helmWritable(s *engine.Session, ref model.ObjectRef) error {
	b, err := writable(s)
	if err != nil {
		return err
	}
	if _, err := b.HelmWriter(); err != nil {
		return err
	}
	ops := s.HelmOps()
	if ops == nil {
		return engine.ErrNoHelmJournal
	}
	if ops.Running(b.Context, ref.Namespace, ref.Name) {
		return fmt.Errorf("a Helm action on release %s is running", ref.Name)
	}
	return nil
}

// RollbackDetail is a rollback's confirmation detail, which names the revision it rolls back to:
// the GUI's Rollback dialog binds its answer to it.
func RollbackDetail(revision int) string {
	return fmt.Sprintf("The release's objects are changed to revision %d's manifest.", revision)
}

// previousRevision is the revision a rollback dialog starts with: the newest one before the
// release's newest, as helm rollback without a revision picks.
func previousRevision(obj *unstructured.Unstructured) int {
	cur := helm.ReleaseRevision(obj)
	best := 0
	for _, r := range helm.ReleaseRevisions(obj) {
		if r < cur && r > best {
			best = r
		}
	}
	if best == 0 && cur > 1 {
		best = cur - 1
	}
	return best
}

// releaseTarget is what a release action acts on: the connected backend's Helm writer, the
// resolved release, and the release as the cache holds it, bound to the revision the user
// confirmed: a release that moved to a new revision since (its newest release Secret's UID
// changed) is refused.
func releaseTarget(ctx context.Context, s *engine.Session, ref model.ObjectRef) (engine.HelmWriter, model.ObjectRef, *unstructured.Unstructured, error) {
	b, err := writable(s)
	if err != nil {
		return nil, ref, nil, err
	}
	w, err := b.HelmWriter()
	if err != nil {
		return nil, ref, nil, err
	}
	ref, _, err = s.ResolveRef(ref)
	if err != nil {
		return nil, ref, nil, err
	}
	if ref.Context != b.Context {
		return nil, ref, nil, fmt.Errorf("%w: the connected cluster changed", engine.ErrWrongCluster)
	}
	if err := engine.CheckConfirmedContext(ctx, b.Context); err != nil {
		return nil, ref, nil, err
	}
	obj, err := s.Object(ctx, ref)
	if err != nil {
		return nil, ref, nil, err
	}
	uid, err := engine.ActUID(ctx, ref, func() (string, bool) { return string(obj.GetUID()), true })
	if err != nil {
		return nil, ref, nil, err
	}
	if uid != string(obj.GetUID()) {
		return nil, ref, nil, fmt.Errorf("%w: release %s moved to revision %d since it was confirmed; nothing was done",
			engine.ErrTargetsChanged, ref.Name, helm.ReleaseRevision(obj))
	}
	return w, ref, obj, nil
}

func runRollback(ctx context.Context, s *engine.Session, p rollbackParams) (any, error) {
	w, ref, obj, err := releaseTarget(ctx, s, p.Ref)
	if err != nil {
		return nil, err
	}
	cur, revs := helm.ReleaseRevision(obj), helm.ReleaseRevisions(obj)
	if p.Revision < 1 || p.Revision > cur || !slices.Contains(revs, p.Revision) {
		return nil, fmt.Errorf("release %s has no revision %d to roll back to (its revisions: %s)", ref.Name, p.Revision, revisionList(revs))
	}
	op := engine.PendingHelmOp{
		Context: ref.Context, Namespace: ref.Namespace, Release: ref.Name, Action: engine.HelmRollback,
		Revision: p.Revision, FromRevision: cur, Started: s.Now().UTC(), StartedBy: startedBy(ctx),
	}
	err = runHelmOp(s, op, fmt.Sprintf("Rolling back release %s to revision %d…", ref.Name, p.Revision), func() error {
		return w.RollbackRelease(ctx, ref.Namespace, ref.Name, p.Revision, cur)
	})
	if err != nil {
		return nil, movedErr(err)
	}
	userToast(s, engine.LevelInfo, fmt.Sprintf("Release %s rolled back to revision %d", ref.Name, p.Revision))
	return HelmResult{Ref: ref, Action: "rolledBack", Revision: p.Revision}, nil
}

func runUninstall(ctx context.Context, s *engine.Session, p refParams) (any, error) {
	w, ref, obj, err := releaseTarget(ctx, s, p.Ref)
	if err != nil {
		return nil, err
	}
	op := engine.PendingHelmOp{
		Context: ref.Context, Namespace: ref.Namespace, Release: ref.Name, Action: engine.HelmUninstall,
		FromRevision: helm.ReleaseRevision(obj), Started: s.Now().UTC(), StartedBy: startedBy(ctx),
	}
	err = runHelmOp(s, op, fmt.Sprintf("Uninstalling release %s…", ref.Name), func() error {
		return w.UninstallRelease(ctx, ref.Namespace, ref.Name)
	})
	if err != nil {
		return nil, err
	}
	userToast(s, engine.LevelInfo, "Release "+ref.Name+" uninstalled")
	return HelmResult{Ref: ref, Action: "uninstalled"}, nil
}

// runHelmOp records an action, runs it with a panic boundary (D-11), and records its outcome.
// movedErr marks an action the Helm SDK refused because the server holds a newer revision than
// the cache showed (helm.ErrReleaseMoved) as ErrTargetsChanged, as a release that moved in the
// cache is.
func movedErr(err error) error {
	if errors.Is(err, helm.ErrReleaseMoved) && !errors.Is(err, engine.ErrTargetsChanged) {
		return fmt.Errorf("%w: %w", engine.ErrTargetsChanged, err)
	}
	return err
}

func runHelmOp(s *engine.Session, op engine.PendingHelmOp, starting string, run func() error) error {
	ops := s.HelmOps()
	if ops == nil {
		return engine.ErrNoHelmJournal
	}
	if err := ops.Begin(op); err != nil {
		return err
	}
	userToast(s, engine.LevelInfo, starting)
	opErr := helmSafely(s, op, run)
	recorded := opErr
	if errors.Is(opErr, helm.ErrUnchanged) {
		recorded = nil // refused before it changed the release: nothing to finish or roll back
	}
	if err := ops.End(op, recorded); err != nil {
		s.Logger().Error("helm: recording the end of a release action", "release", op.Release, "err", err)
		userToast(s, engine.LevelError, "Cogpit could not record the end of the "+string(op.Action)+" of release "+op.Release+": "+err.Error())
	}
	if recorded == nil && opErr != nil {
		userToast(s, engine.LevelWarning, fmt.Sprintf("Release %s was not changed: %v", op.Release, opErr))
		return opErr
	}
	if opErr != nil {
		userToast(s, engine.LevelWarning, fmt.Sprintf("The %s of release %s did not finish: %v", op.Action, op.Release, opErr))
		return fmt.Errorf("%w; the release may be left pending: run it again to finish, or dismiss it in helm_pendingOperations", opErr)
	}
	return nil
}

// withheldErr is a Helm action's error as a caller outside Cogpit's window gets it: the SDK's text
// stays in the window (runHelmOp's toast), errors.Is still sees what it wraps.
type withheldErr struct {
	msg string
	err error
}

func (e *withheldErr) Error() string { return e.msg }
func (e *withheldErr) Unwrap() error { return e.err }

// withheldHelmError stands for a failed Helm action's error where a caller outside Cogpit's
// window reads it (forCaller).
const withheldHelmError = "failed; the error is shown in Cogpit's window only"

// forCaller keeps the text of an install's or upgrade's error from agents and the CLI. A chart's
// templates run the values through tpl, so an agent's values can reach lookup, and the SDK's error
// can quote what lookup read, a Secret's data included (review 2026-10-13); the values themselves
// are scrubbed (helm.Scrubbed), what the cluster returned is not. The user reads the full error in
// the window's toast; the caller learns that the action failed and where to look.
func forCaller(ctx context.Context, action, release string, err error) error {
	if err == nil || engine.OriginOf(ctx) == engine.OriginGUI {
		return err
	}
	return &withheldErr{err: err, msg: fmt.Sprintf("the %s of release %s failed; Cogpit's window shows the error, which is not "+
		"passed on because a chart's templates can quote what they read from the cluster; a release left pending is listed "+
		"by helm_pendingOperations", action, release)}
}

// helmSafely runs a release action and turns a panic into an error: the action's goroutine
// boundary (D-11). Its journal entry is then kept as failed.
func helmSafely(s *engine.Session, op engine.PendingHelmOp, run func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			s.Logger().Error("helm action panicked", "release", op.Release, "action", op.Action, "panic", fmt.Sprint(r), "stack", string(debug.Stack()))
			err = fmt.Errorf("helm %s %s/%s: internal error: %v", op.Action, op.Namespace, op.Release, r)
		}
	}()
	return run()
}

// startedBy is the origin a confirmed command was dispatched by, for the journal.
func startedBy(ctx context.Context) engine.Origin {
	if req, ok := engine.Confirmed(ctx); ok {
		return req.Origin
	}
	return engine.OriginOf(ctx)
}

func revisionList(revs []int) string {
	out := make([]string, len(revs))
	for i, r := range revs {
		out[i] = strconv.Itoa(r)
	}
	return strings.Join(out, ", ")
}
