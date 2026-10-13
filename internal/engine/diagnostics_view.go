package engine

import (
	"cmp"
	"slices"
	"strings"
	"time"

	"github.com/guided-traffic/cogpit/internal/cache"
)

// The Diagnostics view (D-13 cache transparency; beyond-lens › Cache transparency): what the
// status bar sums up, in full, for the connected cluster: its API requests by verb and by verb and
// resource since connecting, each watch (informer) with its state and objects, and the objects per
// kind. It reads the counters the connection and the cache keep (Session.Diagnostics); it sends no
// request. Memory per kind is not measured.

// DockDiagnostics: the Diagnostics dock tab (diagnostics.open).
const DockDiagnostics DockType = "diagnostics"

// Watch states of the Diagnostics view.
const (
	WatchOpen    = "open"    // a handle holds it: a view, a drawer or a lookup uses it
	WatchWarm    = "warm"    // no handle: kept running until its warm period ends
	WatchCovered = "covered" // a label-selected scope served by another informer's watch
)

// DiagCount is one counter: a verb's requests.
type DiagCount struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

// DiagRequest is the requests of one verb on one resource.
type DiagRequest struct {
	Verb     string `json:"verb"`
	Resource string `json:"resource,omitempty"` // empty for requests without one (discovery)
	Count    int64  `json:"count"`
}

// DiagWatch is one informer, or a label-selected scope another informer serves.
type DiagWatch struct {
	Scope   string `json:"scope"` // "pods", "secrets with owner=helm", "pods in team-a"
	Kind    string `json:"kind"`  // the resource with its group, kubectl's way: "deployments.apps"
	State   string `json:"state"` // WatchOpen, WatchWarm or WatchCovered
	Handles int    `json:"handles"`
	Objects int    `json:"objects"`
	Events  int64  `json:"events"` // delivered since the informer started, its initial list included
	Synced  bool   `json:"synced"`
	// WarmUntil is when a warm informer stops; CoveredBy the scope whose watch serves a covered one.
	WarmUntil time.Time `json:"warmUntil,omitzero"`
	CoveredBy string    `json:"coveredBy,omitempty"`
	Error     string    `json:"error,omitempty"`
}

// DiagKind is the objects Cogpit holds of one kind: those of its watches (a covered scope's
// objects are its informer's, counted once).
type DiagKind struct {
	Kind    string `json:"kind"`
	Objects int    `json:"objects"`
	Watches int    `json:"watches"`
}

// DiagnosticsReport is the Diagnostics view's content.
type DiagnosticsReport struct {
	Context   string `json:"context"`
	Connected bool   `json:"connected"`
	Demo      bool   `json:"demo,omitempty"`
	// Requests counts the connection's API requests since connecting, watches included; the demo
	// cluster has no connection and counts none.
	Requests         int64         `json:"requests"`
	RequestsByVerb   []DiagCount   `json:"requestsByVerb"`
	RequestsByTarget []DiagRequest `json:"requestsByVerbAndResource"`
	// Watches are the informers, open first, then warm, then covered, each by scope.
	Watches []DiagWatch `json:"watches"`
	Open    int         `json:"open"`
	Warm    int         `json:"warm"`
	Covered int         `json:"covered"`
	// Kinds are the objects per kind, most first; Objects their sum.
	Kinds   []DiagKind `json:"kinds"`
	Objects int        `json:"objects"`
	// Snapshots counts the table snapshots built since start.
	Snapshots uint64 `json:"snapshots"`
}

// DiagnosticsReport returns the connected cluster's counters as the Diagnostics view shows them.
func (s *Session) DiagnosticsReport() DiagnosticsReport {
	return ReportOf(s.Diagnostics(), s.Backend() != nil)
}

// ReportOf arranges the counters for the Diagnostics view: requests by verb and by verb and
// resource, most first; watches open, warm, then covered; objects per kind.
func ReportOf(d Diagnostics, connected bool) DiagnosticsReport {
	r := DiagnosticsReport{Context: d.Context, Demo: d.Demo, Connected: connected, Snapshots: d.Snapshots,
		Requests: d.Requests.Total, RequestsByVerb: []DiagCount{}, RequestsByTarget: []DiagRequest{},
		Watches: []DiagWatch{}, Kinds: []DiagKind{}}
	for verb, n := range d.Requests.ByVerb {
		r.RequestsByVerb = append(r.RequestsByVerb, DiagCount{Name: verb, Count: n})
	}
	slices.SortFunc(r.RequestsByVerb, func(a, b DiagCount) int { return cmp.Or(cmp.Compare(b.Count, a.Count), cmp.Compare(a.Name, b.Name)) })
	for key, n := range d.Requests.ByVerbResource {
		verb, resource, _ := strings.Cut(key, " ")
		r.RequestsByTarget = append(r.RequestsByTarget, DiagRequest{Verb: verb, Resource: resource, Count: n})
	}
	slices.SortFunc(r.RequestsByTarget, func(a, b DiagRequest) int {
		return cmp.Or(cmp.Compare(b.Count, a.Count), cmp.Compare(a.Resource, b.Resource), cmp.Compare(a.Verb, b.Verb))
	})
	kinds := map[string]*DiagKind{}
	for _, inf := range d.Informers {
		w := diagWatch(inf)
		r.Watches = append(r.Watches, w)
		switch w.State {
		case WatchOpen:
			r.Open++
		case WatchWarm:
			r.Warm++
		default:
			r.Covered++
			continue // its objects are its informer's
		}
		k := kinds[w.Kind]
		if k == nil {
			k = &DiagKind{Kind: w.Kind}
			kinds[w.Kind] = k
		}
		k.Objects += w.Objects
		k.Watches++
		r.Objects += w.Objects
	}
	rank := map[string]int{WatchOpen: 0, WatchWarm: 1, WatchCovered: 2}
	slices.SortFunc(r.Watches, func(a, b DiagWatch) int {
		return cmp.Or(cmp.Compare(rank[a.State], rank[b.State]), cmp.Compare(a.Scope, b.Scope))
	})
	for _, k := range kinds {
		r.Kinds = append(r.Kinds, *k)
	}
	slices.SortFunc(r.Kinds, func(a, b DiagKind) int { return cmp.Or(cmp.Compare(b.Objects, a.Objects), cmp.Compare(a.Kind, b.Kind)) })
	return r
}

func diagWatch(inf cache.InformerStats) DiagWatch {
	kind := inf.Scope.GVR.Resource
	if g := inf.Scope.GVR.Group; g != "" {
		kind += "." + g
	}
	w := DiagWatch{Scope: inf.Scope.String(), Kind: kind, Handles: inf.Handles, Objects: inf.Objects, Events: inf.Events,
		Synced: inf.Synced, CoveredBy: inf.CoveredBy, Error: inf.Error}
	switch {
	case inf.CoveredBy != "":
		w.State = WatchCovered
	case inf.Handles > 0:
		w.State = WatchOpen
	default:
		w.State, w.WarmUntil = WatchWarm, inf.WarmUntil
	}
	return w
}
