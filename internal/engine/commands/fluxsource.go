package commands

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

// Reconcile with Source (D-33, owner 2026-10-14), as `flux reconcile kustomization|helmrelease
// --with-source` does it: a reconcile request on the object's source first, then on the object
// (kinds.FluxSourceOf names the source). A Helm release's HelmChart is reconciled with the chart's
// own source before it, as the flux CLI does; an OCI Helm repository is skipped, as the flux CLI
// skips it (no controller reconciles it). Each patch is reconciliation's annotation alone, bound
// to its object's UID like Reconcile's.
//
// Unlike the flux CLI, Cogpit does not wait for the source to be ready before it annotates the
// object, as its Reconcile does not wait for the controller: kustomize-controller and
// helm-controller reconcile an object again by themselves when its source's artifact changes
// (Flux's documentation; inferred, not verified on a cluster that runs the controllers). Every
// object is checked before the first patch: a suspended source, or one missing from the cluster,
// refuses the whole action with nothing done, where the flux CLI would stop after the sources it
// had already annotated.

// fluxSourceChainLen bounds the walk from an object to its sources: a Helm release's HelmChart,
// then the chart's source.
const fluxSourceChainLen = 2

// fluxSourceChain returns what Reconcile with Source writes for ref, in order: its sources, the
// deepest first, then ref itself (resolved, with the connected context). It reads the sources from
// the cache, waiting for their kinds' initial lists (bounded by ctx and 15 seconds), and refuses an
// object or source that is suspended or missing.
func fluxSourceChain(ctx context.Context, s *engine.Session, ref model.ObjectRef) ([]model.ObjectRef, error) {
	ref, _, err := s.ResolveRef(ref)
	if err != nil {
		return nil, err
	}
	obj, err := s.Object(ctx, ref)
	if err != nil {
		return nil, err
	}
	if kinds.IsFluxSuspended(obj) {
		return nil, fmt.Errorf("%s is suspended: resume it to reconcile it", ref.String())
	}
	b := s.Backend()
	var sources []model.ObjectRef
	key, cur := ref.Kind, obj
	for range fluxSourceChainLen {
		src, err := kinds.FluxSourceOf(key, cur)
		if err != nil {
			return nil, err
		}
		src.Context = ref.Context
		if _, ok := b.Profile(src.Kind); !ok {
			return nil, fmt.Errorf("the cluster does not serve %s, the source of %s", src.Kind, ref.Name)
		}
		o, err := s.Object(ctx, src)
		if err != nil {
			return nil, fmt.Errorf("its source %s: %w", src.String(), err)
		}
		if kinds.IsFluxSuspended(o) {
			return nil, fmt.Errorf("its source %s is suspended: resume it to reconcile with it", src.String())
		}
		if !kinds.FluxSourceStatic(src.Kind, o) {
			sources = append(sources, src)
		}
		if !kinds.FluxSourceHasSource(src.Kind) {
			break
		}
		key, cur = src.Kind, o
	}
	slices.Reverse(sources)
	return append(sources, ref), nil
}

// fluxSourceTargets is Reconcile with Source's Targets: the sources and the object, for the
// confirmation that names them (an agent's, D-2).
func fluxSourceTargets(s *engine.Session, ref model.ObjectRef) ([]model.ObjectRef, error) {
	return fluxSourceChain(context.Background(), s, ref)
}

// fluxSourcePermitted greys Reconcile with Source when the access reviews say the user may not
// patch the object's direct source (the object's own patch is the command's declared verb). A
// review not answered yet lets the API server decide, as for every command.
func fluxSourcePermitted(s *engine.Session, a fluxAction, ref model.ObjectRef) error {
	if !a.withSource {
		return nil
	}
	b := s.Backend()
	obj, err := s.Lookup(ref)
	if b == nil || err != nil {
		return err
	}
	src, err := kinds.FluxSourceOf(ref.Kind, obj)
	if err != nil {
		return err
	}
	p, ok := b.Profile(src.Kind)
	if !ok {
		return fmt.Errorf("the cluster does not serve %s, the source of %s", src.Kind, ref.Name)
	}
	if b.Access == nil {
		return nil
	}
	v := engine.Verb{Verb: nwVerbPatch, Group: p.Group, Resource: p.Resource}
	if allowed, known := b.Access.Allowed(src.Namespace, v); known && !allowed {
		return fmt.Errorf("%w: you may not patch %s.%s in %s, the source of %s", engine.ErrNotPermitted, p.Resource, p.Group, src.Namespace, ref.Name)
	}
	return nil
}

// fluxRunWithSource writes the chain of one object, one patch after the other; a failure stops it
// and says what was done before.
func fluxRunWithSource(ctx context.Context, s *engine.Session, a fluxAction, ref model.ObjectRef) (any, error) {
	b, err := writable(s)
	if err != nil {
		return nil, err
	}
	if ref, _, err = s.ResolveRef(ref); err != nil {
		return nil, err
	}
	if ref.Context != b.Context {
		return nil, fmt.Errorf("%w: the connected cluster changed", engine.ErrWrongCluster)
	}
	if err := engine.CheckConfirmedContext(ctx, b.Context); err != nil {
		return nil, err
	}
	chain, err := fluxSourceChain(ctx, s, ref)
	if err != nil {
		return nil, fmt.Errorf("%s: %w; nothing was done", ref.String(), err)
	}
	if err := engine.CheckConfirmedTargets(ctx, chain); err != nil {
		return nil, err
	}
	var done []model.ObjectRef
	for _, r := range chain {
		if err := fluxWrite(ctx, s, a, r, false); err != nil {
			if len(done) == 0 {
				return nil, fmt.Errorf("%s: %w; nothing was done", r.String(), err)
			}
			return nil, fmt.Errorf("%s: %w; a reconcile was requested for %s only", r.String(), err, refList(done))
		}
		done = append(done, r)
	}
	userToast(s, engine.LevelInfo, ref.Kind+" "+ref.Key()+" "+a.done)
	return FluxManyResult{Action: a.done, Done: done}, nil
}

// fluxSourcePlan is one selected object of Reconcile with Source on a selection: its chain, or why
// nothing can be written for it.
type fluxSourcePlan struct {
	ref   model.ObjectRef
	chain []model.ObjectRef
	err   error
}

// fluxSourcePlans returns the chain of every selected object Reconcile with Source applies to, and
// every object they write, each once, in order: a source two objects share is written once.
func fluxSourcePlans(ctx context.Context, s *engine.Session, a fluxAction, refs []model.ObjectRef) ([]fluxSourcePlan, []model.ObjectRef, error) {
	objs, err := fluxManyTargets(s, a, refs, false)
	if err != nil {
		return nil, nil, err
	}
	plans := make([]fluxSourcePlan, len(objs))
	var all []model.ObjectRef
	for i, r := range objs {
		chain, err := fluxSourceChain(ctx, s, r)
		plans[i] = fluxSourcePlan{ref: r, chain: chain, err: err}
		for _, c := range chain {
			if !slices.ContainsFunc(all, c.Same) {
				all = append(all, c)
			}
		}
	}
	return plans, all, nil
}

// fluxManySourceTargets is Reconcile with Source's Targets on a selection: every object it writes,
// the sources included; an error when it can write none.
func fluxManySourceTargets(s *engine.Session, a fluxAction, refs []model.ObjectRef) ([]model.ObjectRef, error) {
	plans, all, err := fluxSourcePlans(context.Background(), s, a, refs)
	if err != nil {
		return nil, err
	}
	if len(all) == 0 {
		return nil, fmt.Errorf("%s: %w", plans[0].ref.String(), plans[0].err)
	}
	return all, nil
}

// fluxRunManyWithSource writes the chain of each selected object, one object after the other. A
// source written for one object is not written again for the next; a source that failed fails the
// objects that need it. The others go on, and the error names what was done and what was not.
func fluxRunManyWithSource(ctx context.Context, s *engine.Session, a fluxAction, refs []model.ObjectRef) (any, error) {
	b, err := writable(s)
	if err != nil {
		return nil, err
	}
	if err := engine.CheckConfirmedContext(ctx, b.Context); err != nil {
		return nil, err
	}
	plans, all, err := fluxSourcePlans(ctx, s, a, refs)
	if err != nil {
		return nil, err
	}
	if err := engine.CheckConfirmedTargets(ctx, all); err != nil {
		return nil, err
	}
	written := map[model.ObjectRef]error{}
	var done []model.ObjectRef
	var failed []DeleteFailure
	var stop error
	for _, pl := range plans {
		if stop == nil {
			stop = ctx.Err()
		}
		switch {
		case pl.err != nil:
			failed = append(failed, DeleteFailure{Ref: pl.ref, Error: pl.err.Error()})
			continue
		case stop != nil:
			failed = append(failed, DeleteFailure{Ref: pl.ref, Error: fmt.Sprintf("%v: %v", errNotAttempted, stop)})
			continue
		case pl.ref.Context != b.Context || s.Backend() != b:
			stop = fmt.Errorf("%w: the connected cluster changed", engine.ErrWrongCluster)
			failed = append(failed, DeleteFailure{Ref: pl.ref, Error: stop.Error()})
			continue
		}
		if err := fluxWriteChain(ctx, s, a, pl, written, &done); err != nil {
			failed = append(failed, DeleteFailure{Ref: pl.ref, Error: err.Error()})
		}
	}
	if len(failed) > 0 {
		return nil, &FluxManyError{Action: a.done, Done: done, Failed: failed}
	}
	userToast(s, engine.LevelInfo, fmt.Sprintf("%d Flux objects %s", len(plans), a.done))
	return FluxManyResult{Action: a.done, Done: done}, nil
}

// errSourceFailed is an object whose source's patch failed for an object before it.
var errSourceFailed = errors.New("its source was not reconciled")

// fluxWriteChain writes one plan's chain, skipping what an earlier plan wrote: each write's
// outcome is kept in written, each success added to done.
func fluxWriteChain(ctx context.Context, s *engine.Session, a fluxAction, pl fluxSourcePlan, written map[model.ObjectRef]error, done *[]model.ObjectRef) error {
	for _, r := range pl.chain {
		prev, seen := written[r]
		switch {
		case seen && prev != nil:
			return fmt.Errorf("%w: %s: %v", errSourceFailed, r.String(), prev)
		case seen:
			continue
		}
		err := fluxWrite(ctx, s, a, r, false)
		written[r] = err
		if err != nil {
			if r != pl.ref {
				return fmt.Errorf("its source %s: %w", r.String(), err)
			}
			return err
		}
		*done = append(*done, r)
	}
	return nil
}
