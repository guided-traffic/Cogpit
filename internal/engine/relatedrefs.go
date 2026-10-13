package engine

import (
	"cmp"
	"context"
	"slices"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
)

// The related objects of object.get (G17, related: true): the objects the drawer shows for an
// object, as references, from the cache. They are what the drawer's model links to (its owner in
// Controlled By, a pod's node and volumes' claims, a node's pods and devices, a claim's pods, a
// Service's endpoint slices: whatever the related-objects indexes and the profiles' links give),
// and, for a workload, the objects it controls: a Deployment's replica sets and their pods, a Cron
// Job's jobs and their pods, the pods of a Stateful Set, Daemon Set, Replica Set or Job (by the
// controller owner reference, as the workload's logs find them). Nothing is asked of the API
// server: a kind the cache does not hold yet starts its informer, as resources.list does, and is
// waited for at most relatedWait.

// RelatedRef is one object related to an object.get's object.
type RelatedRef struct {
	Ref model.ObjectRef `json:"ref"`
	// Relation says how it relates: the drawer item or section that names it ("Controlled By",
	// "Node", "Pods"), or for the objects a workload controls "Replica Sets", "Jobs" or "Pods".
	Relation string `json:"relation"`
}

// RelatedRefs are the related objects of one object: at most MaxRelated of Total.
type RelatedRefs struct {
	Refs      []RelatedRef `json:"related"`
	Total     int          `json:"relatedTotal"`
	Truncated bool         `json:"relatedTruncated,omitempty"`
	// Skipped names the kinds whose objects could not be read, with the reason (one the user may not
	// list, one still listing after relatedWait).
	Skipped map[string]string `json:"relatedSkipped,omitempty"`
}

// MaxRelated bounds the related objects object.get returns; Total counts them all.
const MaxRelated = 200

// relatedWait bounds the wait for a kind the cache does not hold yet.
const relatedWait = 5 * time.Second

// Relations of the objects a workload controls.
const (
	relationReplicaSets = "Replica Sets"
	relationJobs        = "Jobs"
	relationPods        = "Pods"
)

// controlledKinds are what each workload kind controls, in the order they are listed: a kind and,
// below it, the kind its objects control in turn (a Deployment's replica sets' pods).
var controlledKinds = map[string][]string{
	logKindDeployments:  {logKindReplicaSets, logKindPods},
	"cronjobs":          {logKindJobs, logKindPods},
	logKindStatefulSets: {logKindPods},
	logKindDaemonSets:   {logKindPods},
	logKindReplicaSets:  {logKindPods},
	logKindJobs:         {logKindPods},
}

var controlledRelation = map[string]string{logKindReplicaSets: relationReplicaSets, logKindJobs: relationJobs, logKindPods: relationPods}

// Related returns the objects related to the object of the drawer model m: the links of its
// sections in the drawer's order, then the objects it controls. The object itself and links to a
// kind's list (no name) are left out; each object comes once, with its first relation.
func (s *Session) Related(ctx context.Context, m *DetailModel) RelatedRefs {
	out := RelatedRefs{Refs: []RelatedRef{}}
	seen := map[model.ObjectRef]bool{refKey(m.Ref): true}
	add := func(ref model.ObjectRef, relation string) {
		if ref.Kind == "" || ref.Name == "" {
			return
		}
		if ref.Context == "" {
			ref.Context = m.Ref.Context
		}
		k := refKey(ref)
		if seen[k] {
			return
		}
		seen[k] = true
		out.Total++
		if len(out.Refs) < MaxRelated {
			out.Refs = append(out.Refs, RelatedRef{Ref: ref, Relation: relation})
		}
	}
	for _, sec := range m.Sections {
		for _, it := range sec.Items {
			name := cmp.Or(it.Name, sec.Title)
			if it.Link != nil {
				add(*it.Link, name)
			}
			for _, c := range it.Chips {
				if c.Link != nil {
					add(*c.Link, name)
				}
			}
		}
		if sec.Table != nil {
			for _, row := range sec.Table.Cells {
				for _, c := range row {
					if c.Link != nil {
						add(*c.Link, sec.Title)
					}
				}
			}
		}
	}
	if !m.Missing && !m.Loading {
		for _, r := range s.controlled(ctx, m.Ref, &out) {
			add(r.Ref, r.Relation)
		}
	}
	out.Truncated = out.Total > len(out.Refs)
	return out
}

// refKey is a reference without its context, the key Related dedupes by.
func refKey(r model.ObjectRef) model.ObjectRef {
	return model.ObjectRef{Kind: r.Kind, Namespace: r.Namespace, Name: r.Name}
}

// controlled returns the objects a workload controls, from the cache, by their controller owner
// references: each level's objects sorted by name. A kind that cannot be read is noted in
// out.Skipped.
func (s *Session) controlled(ctx context.Context, ref model.ObjectRef, out *RelatedRefs) []RelatedRef {
	kinds, ok := controlledKinds[ref.Kind]
	if !ok {
		return nil
	}
	obj, err := s.Lookup(ref)
	if err != nil {
		return nil
	}
	wctx, cancel := context.WithTimeout(ctx, relatedWait)
	defer cancel()
	owners := map[string]bool{string(obj.GetUID()): true}
	var refs []RelatedRef
	for _, kind := range kinds {
		if _, _, err := s.Kind(kind); err != nil {
			continue // not served: nothing of it to list
		}
		objs, err := s.Objects(wctx, kind, []string{ref.Namespace})
		if err != nil {
			if out.Skipped == nil {
				out.Skipped = map[string]string{}
			}
			out.Skipped[kind] = err.Error()
			return refs
		}
		var level []*unstructured.Unstructured
		for _, o := range objs {
			if owners[controllerUID(o)] {
				level = append(level, o)
			}
		}
		slices.SortFunc(level, func(a, b *unstructured.Unstructured) int { return cmp.Compare(a.GetName(), b.GetName()) })
		next := make(map[string]bool, len(level))
		for _, o := range level {
			next[string(o.GetUID())] = true
			refs = append(refs, RelatedRef{Ref: model.ObjectRef{Context: ref.Context, Kind: kind, Namespace: o.GetNamespace(), Name: o.GetName()},
				Relation: controlledRelation[kind]})
		}
		owners = next
	}
	return refs
}
