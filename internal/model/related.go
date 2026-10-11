package model

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

// RelatedIndex declares an index over the cached objects of another kind, for cells and drawer
// sections that show objects of that kind: a PVC's pods (pods by the claims they mount), a node's
// pods (pods by node name). Indexes are declared once, in package kinds, and named by profiles in
// KindProfile.Needs; the engine builds them from the cache (architecture §4.4).
type RelatedIndex struct {
	// Kind is the kind key of the indexed objects: "pods".
	Kind string
	// Name names the index, unique per kind: "byNode", "byClaim".
	Name string
	// Keys returns the keys an object is found under: a pod's node name; a pod's
	// "<namespace>/<claim>" for each claim it mounts. It must tolerate any content without
	// panicking.
	Keys func(obj *unstructured.Unstructured) []string
}

// ID identifies the index: "<kind>/<name>".
func (ix RelatedIndex) ID() string { return ix.Kind + "/" + ix.Name }

// Related looks up the objects of other kinds through the indexes a profile declares in Needs. It
// never blocks and never calls the API server. Objects are shared with the cache and must not be
// modified.
type Related interface {
	// Objects returns the objects the index maps key to, in no particular order; nil while the
	// index's kind is still listing, when the cluster does not serve it, or when the profile did
	// not declare the index.
	Objects(index RelatedIndex, key string) []*unstructured.Unstructured
}

// noRelated answers every lookup with nothing.
type noRelated struct{}

func (noRelated) Objects(RelatedIndex, string) []*unstructured.Unstructured { return nil }

// NoRelated is a Related without objects, for rows and details built outside a view.
var NoRelated Related = noRelated{}

func relatedOrNone(rel Related) Related {
	if rel == nil {
		return NoRelated
	}
	return rel
}

// DetailSections returns a profile's own drawer sections for obj: DetailWith with rel, else
// Detail. It may panic if the profile does; callers recover.
func (p *KindProfile) DetailSections(obj *unstructured.Unstructured, rel Related) []DetailSection {
	switch {
	case p.DetailWith != nil:
		return p.DetailWith(obj, relatedOrNone(rel))
	case p.Detail != nil:
		return p.Detail(obj)
	}
	return nil
}
