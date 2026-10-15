package commands

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
)

// Kind keys of the node and workload actions. Their profiles may come from other files or, for
// kinds Cogpit has no profile for yet, not exist: the commands are then disabled.
const (
	nodeKindKey        = "nodes"
	statefulSetKindKey = "statefulsets"
	cronJobKindKey     = "cronjobs"
)

// API verbs and groups of the node and workload actions' Verbs (prefixed: other areas declare
// their own).
const (
	nwVerbCreate = "create"
	nwVerbGet    = "get"
	nwVerbList   = "list"
	nwVerbWatch  = "watch"
	nwVerbPatch  = "patch"
	nwGroupApps  = "apps"
	nwGroupBatch = "batch"
)

// DrainResult is node.drain's result.
type DrainResult struct {
	Ref     model.ObjectRef `json:"ref"`
	Action  string          `json:"action"`         // "drained"
	Evicted int             `json:"evicted"`        // pods evicted
	Left    string          `json:"left,omitempty"` // pods left running (DaemonSet and mirror pods)
}

type drainParams struct {
	Ref model.ObjectRef `json:"ref" jsonschema:"the node: kind nodes and its name"`
	engine.DrainSettings
}

type pendingDrainsParams struct {
	All bool `json:"all,omitempty" jsonschema:"list the pending drains of every context, not only the connected one's"`
}

// drainDetail names, for the confirmation, the drain options that lose data or end pods for good,
// so that "Finish drain" or an agent's drain with them never reads like a plain drain.
func drainDetail(_ *engine.Session, p drainParams) string {
	var notes []string
	if p.DeleteEmptyDirData {
		notes = append(notes, "the emptyDir data of evicted pods is lost")
	}
	if p.Force {
		notes = append(notes, "pods that no controller manages are evicted and not started again")
	}
	if p.IgnoreDaemonSets != nil && !*p.IgnoreDaemonSets {
		notes = append(notes, "the drain refuses a node that runs DaemonSet pods")
	}
	if p.GracePeriodSeconds > 0 {
		notes = append(notes, fmt.Sprintf("each pod gets %d seconds to shut down", p.GracePeriodSeconds))
	}
	if len(notes) == 0 {
		return ""
	}
	return "With these options: " + strings.Join(notes, "; ") + "."
}

// drainVerbs are the requests kubectl's drain library makes, as cluster.DrainNode documents them;
// the discovery GET on /api/v1 is not a resource and is allowed to every authenticated user.
var drainVerbs = []engine.Verb{
	{Verb: nwVerbPatch, Resource: nodeKindKey},
	{Verb: nwVerbList, Resource: kindPods},
	{Verb: nwVerbGet, Resource: kindPods},
	{Verb: nwVerbCreate, Resource: kindPods, Subresource: "eviction"},
	{Verb: nwVerbGet, Group: nwGroupApps, Resource: "daemonsets"},
}

func registerNodes(g *registrar) {
	add(g, engine.Spec[refParams]{
		ID: "node.cordon", Label: "Cordon", MenuPath: "Node/Cordon", Kinds: []string{nodeKindKey},
		Order: 200, Danger: engine.DangerMutating,
		Verbs: []engine.Verb{{Verb: nwVerbPatch, Resource: nodeKindKey}},
		Description: "Cordons a node as kubectl cordon does: sets spec.unschedulable, so that no new pods are scheduled " +
			"on it; running pods stay. Not offered for a cordoned node, where node_uncordon takes its place.",
		Enabled: func(s *engine.Session, p refParams) error { return nodeToggleEnabled(s, p.Ref, false) },
		Targets: func(s *engine.Session, p refParams) ([]model.ObjectRef, error) { return targetOf(s, p.Ref) },
		Run: func(ctx context.Context, s *engine.Session, p refParams) (any, error) {
			return act(ctx, s, p.Ref, "cordoned", func(w engine.Actions, ref model.ObjectRef) error {
				nw, err := asNodeActions(w)
				if err != nil {
					return err
				}
				return nw.SetNodeUnschedulable(ctx, ref.Name, true)
			})
		},
	})

	add(g, engine.Spec[refParams]{
		ID: "node.uncordon", Label: "Uncordon", MenuPath: "Node/Uncordon", Kinds: []string{nodeKindKey},
		Order: 200, Danger: engine.DangerMutating,
		Verbs: []engine.Verb{{Verb: nwVerbPatch, Resource: nodeKindKey}},
		Description: "Uncordons a cordoned node as kubectl uncordon does: removes spec.unschedulable, so that pods are " +
			"scheduled on it again. Offered only for a cordoned node, in node_cordon's place. Also clears the node's " +
			"unfinished drain from node_pendingDrains.",
		Enabled: func(s *engine.Session, p refParams) error { return nodeToggleEnabled(s, p.Ref, true) },
		Targets: func(s *engine.Session, p refParams) ([]model.ObjectRef, error) { return targetOf(s, p.Ref) },
		Run: func(ctx context.Context, s *engine.Session, p refParams) (any, error) {
			res, err := act(ctx, s, p.Ref, "uncordoned", func(w engine.Actions, ref model.ObjectRef) error {
				nw, err := asNodeActions(w)
				if err != nil {
					return err
				}
				return nw.SetNodeUnschedulable(ctx, ref.Name, false)
			})
			if err != nil {
				return nil, err
			}
			forgetDrainAfterUncordon(s, res.(ActionResult).Ref)
			return res, nil
		},
	})

	add(g, engine.Spec[drainParams]{
		ID: "node.drain", Label: "Drain", MenuPath: "Node/Drain", Kinds: []string{nodeKindKey},
		Order: 300, Danger: engine.DangerDestructive,
		Verbs: drainVerbs,
		Description: "Drains a node as kubectl drain does: cordons it, then evicts every pod on it through the eviction " +
			"API, which honours PodDisruptionBudgets, and waits until they are gone. By default it leaves DaemonSet pods " +
			"running and refuses (before evicting anything) when the node runs pods with emptyDir data or pods no " +
			"controller manages; deleteEmptyDirData and force allow those. A drain that fails or times out leaves the " +
			"node cordoned and partly drained; it stays in node_pendingDrains, and so does one a crash of Cogpit " +
			"interrupts. Run it again to finish it. Always asks the user to confirm.",
		Detail: drainDetail,
		Enabled: func(s *engine.Session, p drainParams) error {
			if _, err := cachedOfKind(s, p.Ref, nodeKindKey); err != nil {
				return err
			}
			b, _, err := nodeActionsOf(s)
			if err != nil {
				return err
			}
			d := s.Drains()
			if d == nil {
				return engine.ErrNoDrainJournal
			}
			if d.Running(b.Context, p.Ref.Name) {
				return fmt.Errorf("a drain of node %s is running", p.Ref.Name)
			}
			return nil
		},
		Targets: func(s *engine.Session, p drainParams) ([]model.ObjectRef, error) { return targetOf(s, p.Ref) },
		Prefill: func(_ *engine.Session, p drainParams) (drainParams, error) {
			p.DrainSettings = p.WithDefaults()
			return p, nil
		},
		Run: runDrain,
	})

	add(g, engine.Spec[pendingDrainsParams]{
		ID: "node.pendingDrains", Label: "Unfinished Drains", Danger: engine.DangerRead,
		Verbs: []engine.Verb{{Verb: nwVerbList, Resource: nodeKindKey}, {Verb: nwVerbWatch, Resource: nodeKindKey}},
		Description: "Lists the drains Cogpit started on the connected context (all: on every context) that did not " +
			"finish: context, node, when and by whom started, the settings, status (running in this Cogpit or, " +
			"with elsewhere and its owner's process, in another one, interrupted by an earlier run of Cogpit ending, or failed with its error) and the node's state from the " +
			"cache (cordoned, schedulable, missing, unknown). Offer node_drain to finish one, node_uncordon to give up " +
			"and make the node schedulable, or node_dismissDrain to forget it.",
		Run: func(_ context.Context, s *engine.Session, p pendingDrainsParams) (any, error) {
			list, err := s.PendingDrains(p.All)
			if err != nil {
				return nil, err
			}
			if list == nil {
				list = []engine.PendingDrainInfo{}
			}
			return list, nil
		},
	})

	add(g, engine.Spec[refParams]{
		ID: "node.dismissDrain", Label: "Dismiss Unfinished Drain", Danger: engine.DangerUI,
		Description: "Forgets an unfinished drain from node_pendingDrains without changing the cluster: the node stays " +
			"as it is (cordoned and partly drained, unless someone changed it). The reference's context defaults to " +
			"the connected one.",
		Enabled: func(s *engine.Session, p refParams) error {
			if s.Drains() == nil {
				return engine.ErrNoDrainJournal
			}
			if p.Ref.Kind != nodeKindKey || p.Ref.Name == "" {
				return errors.New("name a node: kind nodes and its name")
			}
			return nil
		},
		Run: func(_ context.Context, s *engine.Session, p refParams) (any, error) {
			ref := p.Ref
			if ref.Context == "" {
				b := s.Backend()
				if b == nil {
					return nil, engine.ErrNotConnected
				}
				ref.Context = b.Context
			}
			ref.Namespace = ""
			d := s.Drains()
			if _, ok, err := d.Pending(ref.Context, ref.Name); err != nil {
				return nil, err
			} else if !ok {
				return nil, fmt.Errorf("no unfinished drain of node %s in context %s", ref.Name, ref.Context)
			}
			if err := d.Forget(ref.Context, ref.Name); err != nil {
				return nil, err
			}
			return ActionResult{Ref: ref, Action: "dismissed"}, nil
		},
	})
}

// cachedOfKind is the first Enabled check of actions on one object: a reference of another kind
// does not apply at all, and the object must be in the cache.
func cachedOfKind(s *engine.Session, ref model.ObjectRef, kind string) (*unstructured.Unstructured, error) {
	if ref.Kind != kind {
		return nil, fmt.Errorf("%w: this action applies to %s", engine.ErrNotApplicable, kind)
	}
	return s.Lookup(ref)
}

// nodeActionsOf is the writable check of the node commands.
func nodeActionsOf(s *engine.Session) (*engine.Backend, engine.NodeActions, error) {
	b, err := writable(s)
	if err != nil {
		return nil, nil, err
	}
	w, err := asNodeActions(b.Writer())
	if err != nil {
		return nil, nil, err
	}
	return b, w, nil
}

func asNodeActions(w engine.Actions) (engine.NodeActions, error) {
	nw, ok := w.(engine.NodeActions)
	if !ok {
		return nil, errors.New("this connection cannot change nodes")
	}
	return nw, nil
}

// nodeToggleEnabled is the Enabled check of Cordon (wantCordoned false) and Uncordon (true). Which
// of the two applies comes from the cache first, so that a row menu hides the other one even on a
// cluster Cogpit cannot change; then whether the change can be made.
func nodeToggleEnabled(s *engine.Session, ref model.ObjectRef, wantCordoned bool) error {
	obj, err := cachedOfKind(s, ref, nodeKindKey)
	if err != nil {
		return err
	}
	cordoned, _ := model.NestedBool(obj.Object, "spec", "unschedulable")
	switch {
	case cordoned && !wantCordoned:
		return fmt.Errorf("%w: node %s is cordoned already", engine.ErrNotApplicable, ref.Name)
	case !cordoned && wantCordoned:
		return fmt.Errorf("%w: node %s is not cordoned", engine.ErrNotApplicable, ref.Name)
	}
	b, _, err := nodeActionsOf(s)
	if err != nil {
		return err
	}
	if s.Drains().Running(b.Context, ref.Name) {
		return fmt.Errorf("a drain of node %s is running", ref.Name)
	}
	return nil
}

// runDrain is node.drain's Run: record the drain, drain, record the outcome. The journal entry is
// written before the node is cordoned, so that a crash at any later point leaves it behind.
func runDrain(ctx context.Context, s *engine.Session, p drainParams) (any, error) {
	b, w, err := nodeActionsOf(s)
	if err != nil {
		return nil, err
	}
	ref, _, err := s.ResolveRef(p.Ref)
	if err != nil {
		return nil, err
	}
	if ref.Context != b.Context {
		return nil, fmt.Errorf("%w: the connected cluster changed", engine.ErrWrongCluster)
	}
	if err := engine.CheckConfirmedContext(ctx, b.Context); err != nil {
		return nil, err
	}
	d := s.Drains()
	if d == nil {
		return nil, engine.ErrNoDrainJournal
	}
	entry := engine.PendingDrain{
		Context: ref.Context, Node: ref.Name, Started: s.Now().UTC(), Settings: p.WithDefaults(),
	}
	if req, ok := engine.Confirmed(ctx); ok {
		entry.StartedBy = req.Origin
	}
	if err := d.Begin(entry); err != nil {
		return nil, err
	}
	userToast(s, "info", fmt.Sprintf("Draining node %s (context %s)…", ref.Name, ref.Context))

	var last cluster.DrainProgress // calls come one at a time; read after the drain returned
	left := ""
	// An interruption of the connection stops the drain, which stays in the journal (D-58, S8).
	octx, ocancel := s.OperationContext(ctx)
	defer ocancel()
	drainErr := drainSafely(octx, s, w, ref.Name, p.ClusterOptions(), func(pr cluster.DrainProgress) {
		last = pr
		d.Progress(ref.Context, pr)
		switch pr.Step {
		case cluster.DrainListed:
			left = pr.Message
			userToast(s, "info", fmt.Sprintf("Node %s cordoned; evicting %d pods", ref.Name, pr.Total))
		case cluster.DrainBlocked:
			userToast(s, "warning", fmt.Sprintf("Node %s: the eviction of pod %s waits: %s", ref.Name, pr.Pod, pr.Message))
		}
	})
	if errors.Is(context.Cause(octx), engine.ErrConnectionLost) && drainErr != nil {
		drainErr = fmt.Errorf("%w: %w", engine.ErrConnectionLost, drainErr)
	}
	if err := d.End(entry, drainErr); err != nil {
		s.Logger().Error("node.drain: recording the drain's end", "node", ref.Name, "err", err)
		userToast(s, "error", "Cogpit could not record the end of the drain of node "+ref.Name+": "+err.Error())
	}
	if drainErr != nil {
		msg := fmt.Sprintf("The drain of node %s did not finish; the node stays cordoned and partly drained. "+
			"Finish the drain or uncordon the node.", ref.Name)
		userToast(s, "warning", msg)
		return nil, fmt.Errorf("%w; node %s stays cordoned and partly drained: run node_drain again to finish, "+
			"or node_uncordon to make it schedulable", drainErr, ref.Name)
	}
	userToast(s, "info", fmt.Sprintf("Node %s drained: %d pods evicted", ref.Name, last.Done))
	return DrainResult{Ref: ref, Action: "drained", Evicted: last.Done, Left: left}, nil
}

// drainSafely runs the drain and turns a panic into an error: the drain's goroutine boundary
// (D-11). The drain's record is then kept as failed.
func drainSafely(ctx context.Context, s *engine.Session, w engine.NodeActions, node string, opts cluster.DrainOptions,
	progress func(cluster.DrainProgress)) (err error) {
	defer func() {
		if r := recover(); r != nil {
			s.Logger().Error("node.drain panicked", "node", node, "panic", fmt.Sprint(r), "stack", string(debug.Stack()))
			err = fmt.Errorf("drain node %s: internal error: %v", node, r)
		}
	}()
	return w.DrainNode(ctx, node, opts, progress)
}

// forgetDrainAfterUncordon clears the node's pending drain once it is uncordoned: the user gave
// the drain up.
func forgetDrainAfterUncordon(s *engine.Session, ref model.ObjectRef) {
	d := s.Drains()
	if d == nil {
		return
	}
	_, ok, err := d.Pending(ref.Context, ref.Name)
	if err == nil && ok {
		err = d.Forget(ref.Context, ref.Name)
	}
	if err != nil {
		s.Logger().Warn("node.uncordon: the unfinished drain stays recorded", "node", ref.Name, "err", err)
		userToast(s, "warning", "Node "+ref.Name+" is uncordoned, but its unfinished drain stays recorded: "+err.Error())
	}
}

// userToast emits a toast event.
func userToast(s *engine.Session, level, msg string) {
	s.Emit(engine.Event{Type: engine.EventToast, Level: level, Message: msg})
}
