package commands

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

// The Flux CD actions (resource-parity › Paid Lens Desktop views › Flux CD, G23; night decision
// D-33): Reconcile, Suspend and Resume on one Flux object and on a selection, each one JSON merge
// patch as the flux CLI sends it (cluster.FluxPatch), bound to the object's UID. Lens's Flux
// actions are not documented or captured: their names, menu places and rules are Cogpit's.
//
//   - Reconcile sets reconcile.fluxcd.io/requestedAt to now; for a Helm release optionally also
//     reconcile.fluxcd.io/forceAt (force), as `flux reconcile helmrelease --force` does. Not on a
//     suspended object (the controller would ignore it, as the flux CLI refuses it), nor on an
//     Alert or Provider that reports no status (notification v1beta3's static objects).
//   - Suspend sets spec.suspend: true; Resume sets it false and requests a reconcile, as `flux
//     resume` does. Each is offered where the other is not.
//
// All are mutating, not destructive (D-33): agents need the user's acceptance unless agent writes
// are allowed for the context (D-2), and the audit log has every agent's.

// Command ids of the Flux actions.
const (
	fluxReconcileID     = "flux.reconcile"
	fluxSuspendID       = "flux.suspend"
	fluxResumeID        = "flux.resume"
	fluxReconcileManyID = "flux.reconcileMany"
	fluxSuspendManyID   = "flux.suspendMany"
	fluxResumeManyID    = "flux.resumeMany"
)

// Row-menu places of the Flux actions: before Edit and Delete, Reconcile first; Suspend and Resume
// share a place, as only one of them applies.
const (
	orderFluxReconcile = 100
	orderFluxSuspend   = 200
)

// The parameters of the Flux actions: Reconcile's take force, Suspend's and Resume's do not.
type (
	fluxReconcileParams struct {
		Ref   model.ObjectRef `json:"ref" jsonschema:"the Flux object"`
		Force bool            `json:"force,omitempty" jsonschema:"Helm releases only: also force a one-off install or upgrade even without changes (reconcile.fluxcd.io/forceAt), as flux reconcile helmrelease --force does"`
	}
	fluxReconcileManyParams struct {
		Refs  []model.ObjectRef `json:"refs" jsonschema:"the Flux objects, as ui_state's selected lists them, at most 500"`
		Force bool              `json:"force,omitempty" jsonschema:"Helm releases only: also force a one-off install or upgrade (reconcile.fluxcd.io/forceAt); refused when a selected object is not a Helm release"`
	}
	fluxRefParams struct {
		Ref model.ObjectRef `json:"ref" jsonschema:"the Flux object"`
	}
	fluxRefsParams struct {
		Refs []model.ObjectRef `json:"refs" jsonschema:"the Flux objects, as ui_state's selected lists them, at most 500"`
	}
)

// fluxKindKeys returns the Flux kind keys that take Reconcile (suspend false) or Suspend and Resume.
func fluxKindKeys(suspend bool) []string {
	var out []string
	for _, k := range kinds.FluxKinds() {
		if (suspend && k.Suspend) || (!suspend && k.Reconcile) {
			out = append(out, k.Key)
		}
	}
	return out
}

// fluxAction is one of the three actions: what it writes and when it applies to an object.
type fluxAction struct {
	id, manyID, label, done string
	suspend                 bool // its kinds are those with spec.suspend
	order                   int
	// applies returns nil when the action applies to the object in its state now, else a fluxSkip
	// that says why not.
	applies func(k kinds.FluxKind, obj *unstructured.Unstructured) error
	request func(force bool) cluster.FluxRequest
}

// fluxSkip is an object an action applies to by its kind but not in its state now: a suspended
// object for Reconcile (greyed, with the reason), one suspended already for Suspend (hidden: Resume
// takes its place). An action on a selection skips such objects.
type fluxSkip struct {
	reason string
	hide   bool // the single action is not offered at all (engine.ErrNotApplicable)
}

func (e fluxSkip) Error() string { return e.reason }

// Is makes a hidden skip engine.ErrNotApplicable.
func (e fluxSkip) Is(target error) bool { return e.hide && target == engine.ErrNotApplicable }

var (
	fluxReconcile = fluxAction{
		id: fluxReconcileID, manyID: fluxReconcileManyID, label: "Reconcile", done: "reconcile requested", order: orderFluxReconcile,
		applies: func(k kinds.FluxKind, obj *unstructured.Unstructured) error {
			if kinds.IsFluxSuspended(obj) {
				return fluxSkip{reason: obj.GetName() + " is suspended: resume it to reconcile it"}
			}
			if (k.APIKind == "Alert" || k.APIKind == "Provider") && len(model.NestedSlice(obj, "status", "conditions")) == 0 {
				return fluxSkip{reason: obj.GetName() + " reports no status: the notification controller does not reconcile it", hide: true}
			}
			return nil
		},
		request: func(force bool) cluster.FluxRequest { return cluster.FluxRequest{Reconcile: true, Force: force} },
	}
	fluxSuspend = fluxAction{
		id: fluxSuspendID, manyID: fluxSuspendManyID, label: "Suspend", done: "suspended", suspend: true, order: orderFluxSuspend,
		applies: func(_ kinds.FluxKind, obj *unstructured.Unstructured) error {
			if kinds.IsFluxSuspended(obj) {
				return fluxSkip{reason: obj.GetName() + " is suspended already", hide: true}
			}
			return nil
		},
		request: func(bool) cluster.FluxRequest { yes := true; return cluster.FluxRequest{Suspend: &yes} },
	}
	fluxResume = fluxAction{
		id: fluxResumeID, manyID: fluxResumeManyID, label: "Resume", done: "resumed", suspend: true, order: orderFluxSuspend,
		applies: func(_ kinds.FluxKind, obj *unstructured.Unstructured) error {
			if !kinds.IsFluxSuspended(obj) {
				return fluxSkip{reason: obj.GetName() + " is not suspended", hide: true}
			}
			return nil
		},
		request: func(bool) cluster.FluxRequest { no := false; return cluster.FluxRequest{Reconcile: true, Suspend: &no} },
	}
)

// Descriptions of the actions for agents.
var fluxDescriptions = map[string]string{
	fluxReconcileID: "Asks Flux to reconcile one object now, as flux reconcile does without --with-source: sets the annotation " +
		"reconcile.fluxcd.io/requestedAt to the current time (with force, a Helm release also gets reconcile.fluxcd.io/forceAt). " +
		"Does not wait for the controller; the object's Status column follows its conditions. Not offered for a suspended " +
		"object, nor for an Alert or Provider that reports no status. Kinds: every Flux kind but External Artifacts.",
	fluxSuspendID: "Suspends one Flux object, as flux suspend does: sets spec.suspend to true, so that its controller leaves it " +
		"alone until it is resumed. Offered only where flux_resume is not.",
	fluxResumeID: "Resumes a suspended Flux object, as flux resume does: sets spec.suspend to false and requests a reconcile " +
		"(reconcile.fluxcd.io/requestedAt). Offered only for a suspended object, in flux_suspend's place.",
}

func registerFlux(g *registrar) {
	add(g, fluxOneSpec(fluxReconcile, func(p fluxReconcileParams) (model.ObjectRef, bool) { return p.Ref, p.Force }))
	add(g, fluxManySpec(fluxReconcile, func(p fluxReconcileManyParams) ([]model.ObjectRef, bool) { return p.Refs, p.Force }))
	for _, a := range []fluxAction{fluxSuspend, fluxResume} {
		add(g, fluxOneSpec(a, func(p fluxRefParams) (model.ObjectRef, bool) { return p.Ref, false }))
		add(g, fluxManySpec(a, func(p fluxRefsParams) ([]model.ObjectRef, bool) { return p.Refs, false }))
	}
	registerFluxDashboard(g)
}

type fluxDashboardParams struct {
	Namespaces []string `json:"namespaces,omitempty" jsonschema:"namespaces to count; omitted: the namespace filter the user has set"`
	Table      bool     `json:"table,omitempty" jsonschema:"true: include the page's table, the Flux objects of those namespaces that are not ready (most recent change first)"`
}

// FluxDashboardResult is flux.dashboard's result.
type FluxDashboardResult struct {
	engine.FluxDashboardModel
	// NotReady is the page's table: Name, Kind, Namespace, Status, Message, Age. Only with table:
	// true.
	NotReady *engine.TableSnapshot `json:"notReady,omitempty"`
}

// registerFluxDashboard registers the Flux dashboard's read (D-34): agents read what the page
// shows; ui.openView "flux/dashboard" shows it to the user.
func registerFluxDashboard(g *registrar) {
	add(g, engine.Spec[fluxDashboardParams]{
		ID: "flux.dashboard", Label: engine.FluxDashboardTitle, MenuPath: "View/" + engine.FluxDashboardTitle, Danger: engine.DangerRead,
		Verbs: cacheReads,
		Description: "Reads the Flux dashboard: for each Flux kind the cluster serves (Helm releases, Kustomizations, image " +
			"repositories, policies and update automations, alerts, providers, receivers, Git, OCI and Helm repositories, " +
			"buckets, external artifacts) the number of objects in the selected namespaces by state: ready, notReady (Ready " +
			"False or Stalled), reconciling, suspended, unknown (no Ready condition yet); with table, also every object that " +
			"is not ready, with its message. Read from the cache. ui_openView with view flux/dashboard shows the page to the " +
			"user; flux_reconcile, flux_suspend and flux_resume act on the objects.",
		Enabled: func(s *engine.Session, _ fluxDashboardParams) error { _, err := connected(s); return err },
		Run: func(ctx context.Context, s *engine.Session, p fluxDashboardParams) (any, error) {
			m, table, err := s.FluxDashboard(ctx, p.Namespaces, p.Table)
			if err != nil {
				return nil, err
			}
			return FluxDashboardResult{FluxDashboardModel: *m, NotReady: table}, nil
		},
	})
}

// fluxOneSpec is an action on one object: the row menu's, the drawer's and the palette's. get
// reads the object and force from its parameters.
func fluxOneSpec[P any](a fluxAction, get func(P) (model.ObjectRef, bool)) engine.Spec[P] {
	return engine.Spec[P]{
		ID: a.id, Label: a.label, MenuPath: "Flux/" + a.label, Kinds: fluxKindKeys(a.suspend),
		Order: a.order, Danger: engine.DangerMutating,
		Verbs:       []engine.Verb{{Verb: nwVerbPatch, Resource: "*"}},
		Description: fluxDescriptions[a.id],
		Enabled: func(s *engine.Session, p P) error {
			ref, force := get(p)
			if err := fluxApplies(s, a, ref, force); err != nil {
				return err
			}
			_, _, err := fluxActionsOf(s)
			return err
		},
		Targets: func(s *engine.Session, p P) ([]model.ObjectRef, error) { ref, _ := get(p); return targetOf(s, ref) },
		Run: func(ctx context.Context, s *engine.Session, p P) (any, error) {
			ref, force := get(p)
			return act(ctx, s, ref, a.done, func(_ engine.Actions, ref model.ObjectRef) error {
				return fluxWrite(ctx, s, a, ref, force)
			})
		},
	}
}

// fluxManySpec is an action on a selection: the selection's menu's. It acts on the selected
// objects it applies to (Reconcile skips the suspended ones, Suspend the suspended, Resume the
// others), one after the other, as Delete of a selection does.
func fluxManySpec[P any](a fluxAction, get func(P) ([]model.ObjectRef, bool)) engine.Spec[P] {
	desc := "Does what " + strings.ReplaceAll(a.id, ".", "_") + " does for several Flux objects at once, as the table's " +
		"selection menu does: it acts on the selected objects it applies to, one after the other, each bound to its UID, " +
		"and skips the others. When an object fails the others still go on, and the error names which were done and which " +
		"were not, each with its reason; nothing is retried. At most 500 objects."
	return engine.Spec[P]{
		ID: a.manyID, Label: a.label, MenuPath: "Flux/" + a.label + " Selected", Kinds: fluxKindKeys(a.suspend),
		Order: a.order, Danger: engine.DangerMutating,
		Verbs:       []engine.Verb{{Verb: nwVerbPatch, Resource: "*"}},
		Description: desc,
		Enabled: func(s *engine.Session, p P) error {
			refs, force := get(p)
			if _, err := fluxManyTargets(s, a, refs, force); err != nil {
				return err
			}
			_, _, err := fluxActionsOf(s)
			return err
		},
		Targets: func(s *engine.Session, p P) ([]model.ObjectRef, error) {
			refs, force := get(p)
			return fluxManyTargets(s, a, refs, force)
		},
		Run: func(ctx context.Context, s *engine.Session, p P) (any, error) {
			refs, force := get(p)
			return fluxRunMany(ctx, s, a, refs, force)
		},
	}
}

// fluxApplies checks that the action applies to the object as the cache holds it.
func fluxApplies(s *engine.Session, a fluxAction, ref model.ObjectRef, force bool) error {
	k, ok := kinds.FluxKindOf(ref.Kind)
	if !ok || (a.suspend && !k.Suspend) || (!a.suspend && !k.Reconcile) {
		return fmt.Errorf("%w: %s does not apply to %s", engine.ErrNotApplicable, a.label, ref.Kind)
	}
	if force && k.APIKind != "HelmRelease" {
		return fmt.Errorf("force applies to Helm releases only, not to %s", k.Title)
	}
	obj, err := s.Lookup(ref)
	if err != nil {
		return err
	}
	return a.applies(k, obj)
}

// fluxManyTargets returns the selected objects the action applies to, each once, in order; an
// error when there are none, or a selected object is not a Flux object of the action's kinds or
// not in the cache.
func fluxManyTargets(s *engine.Session, a fluxAction, refs []model.ObjectRef, force bool) ([]model.ObjectRef, error) {
	if len(refs) == 0 {
		return nil, errors.New("no objects are selected")
	}
	if len(refs) > engine.MaxSelection {
		return nil, fmt.Errorf("at most %d objects at once, %d are selected", engine.MaxSelection, len(refs))
	}
	var out []model.ObjectRef
	var reason error
	for _, r := range refs {
		var skip fluxSkip
		err := fluxApplies(s, a, r, force)
		switch {
		case errors.As(err, &skip):
			reason = err
		case err != nil:
			return nil, fmt.Errorf("%s: %w", r.String(), err)
		default:
			ref, _, rerr := s.ResolveRef(r)
			if rerr != nil {
				return nil, fmt.Errorf("%s: %w", r.String(), rerr)
			}
			if !slices.ContainsFunc(out, ref.Same) {
				out = append(out, ref)
			}
		}
	}
	if len(out) == 0 {
		if reason == nil {
			reason = errors.New("nothing to do")
		}
		return nil, fmt.Errorf("%w: no selected object to %s: %v", engine.ErrNotApplicable, strings.ToLower(a.label), reason)
	}
	return out, nil
}

// FluxManyResult is the result of an action on a selection when every object was done.
type FluxManyResult struct {
	Action string            `json:"action"` // "reconcile requested", "suspended", "resumed"
	Done   []model.ObjectRef `json:"done"`
}

// FluxManyError is the error of an action on a selection that did not do every object.
type FluxManyError struct {
	Action string
	Done   []model.ObjectRef
	Failed []DeleteFailure
}

func (e *FluxManyError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s: %d of %d objects", e.Action, len(e.Done), len(e.Done)+len(e.Failed))
	if len(e.Done) > 0 {
		b.WriteString(" (" + refList(e.Done) + ")")
	}
	b.WriteString("; not done: ")
	for i, f := range e.Failed {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(f.Ref.String() + ": " + f.Error)
	}
	b.WriteString(". Nothing was retried.")
	return b.String()
}

// fluxRunMany runs an action on the selection's objects it applies to, one after the other, each
// bound to its UID; every failure is kept for the report.
func fluxRunMany(ctx context.Context, s *engine.Session, a fluxAction, refs []model.ObjectRef, force bool) (any, error) {
	b, err := writable(s)
	if err != nil {
		return nil, err
	}
	if err := engine.CheckConfirmedContext(ctx, b.Context); err != nil {
		return nil, err
	}
	targets, err := fluxManyTargets(s, a, refs, force)
	if err != nil {
		return nil, err
	}
	if err := engine.CheckConfirmedTargets(ctx, targets); err != nil {
		return nil, err
	}
	var done []model.ObjectRef
	var failed []DeleteFailure
	var stop error
	for _, ref := range targets {
		if stop == nil {
			stop = ctx.Err()
		}
		if stop != nil {
			failed = append(failed, DeleteFailure{Ref: ref, Error: fmt.Sprintf("%v: %v", errNotAttempted, stop)})
			continue
		}
		if ref.Context != b.Context || s.Backend() != b {
			stop = fmt.Errorf("%w: the connected cluster changed", engine.ErrWrongCluster)
			failed = append(failed, DeleteFailure{Ref: ref, Error: stop.Error()})
			continue
		}
		if err := fluxWrite(ctx, s, a, ref, force); err != nil {
			failed = append(failed, DeleteFailure{Ref: ref, Error: err.Error()})
			continue
		}
		done = append(done, ref)
	}
	if len(failed) > 0 {
		return nil, &FluxManyError{Action: a.done, Done: done, Failed: failed}
	}
	userToast(s, engine.LevelInfo, fmt.Sprintf("%d Flux objects %s", len(done), a.done))
	return FluxManyResult{Action: a.done, Done: done}, nil
}

// fluxWrite patches one object: its kind's served version, the UID the user confirmed (or the
// cache's), the action's request.
func fluxWrite(ctx context.Context, s *engine.Session, a fluxAction, ref model.ObjectRef, force bool) error {
	b, w, err := fluxActionsOf(s)
	if err != nil {
		return err
	}
	p, ok := b.Profile(ref.Kind)
	if !ok {
		return fmt.Errorf("%w: the cluster does not serve %s", engine.ErrNotApplicable, ref.Kind)
	}
	uid, err := sameAsConfirmed(ctx, s, ref)
	if err != nil {
		return err
	}
	return replaced(ref, w.PatchFlux(ctx, p.GVR(), ref.Namespace, ref.Name, uid, a.request(force)))
}

// fluxActionsOf is the writable check of the Flux commands.
func fluxActionsOf(s *engine.Session) (*engine.Backend, engine.FluxActions, error) {
	b, err := writable(s)
	if err != nil {
		return nil, nil, err
	}
	w, ok := b.Writer().(engine.FluxActions)
	if !ok {
		return nil, nil, errors.New("this connection cannot change Flux objects")
	}
	return b, w, nil
}
