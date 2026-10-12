package metrics

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/guided-traffic/cogpit/internal/model"
)

// The cluster as a whole (G12, the cluster Overview's charts): a scope whose object is the sum over
// the cluster's nodes. It is polled like a drawer's object, through the same roles and fallback:
// history from Prometheus once when the page opens or its range changes (the sums over the nodes'
// root cgroups, promcluster.go), then the Metrics Server's live values (one list of the nodes'
// measurements per poll, summed here), Prometheus's newest points while the Metrics Server fails.
//
// The chart has three targets: "" every node, TargetMaster the control-plane nodes, TargetWorker
// the others. Which nodes are control-plane nodes is the cache's to know, not the metrics': the
// scope's Name lists them (ClusterScope), so a change of the control plane is another scope with a
// history of its own.

// KindCluster is the scope of the cluster as a whole.
const KindCluster Kind = "cluster"

// The targets of the cluster's chart besides "" (every node).
const (
	TargetMaster = "master"
	TargetWorker = "worker"
)

// ClusterScope is the scope of the cluster's totals, with the control-plane nodes named so that the
// chart can tell the masters' sum from the workers'. The names are sorted and joined by commas
// (node names are DNS subdomains: no comma).
func ClusterScope(masters []string) Scope {
	names := slices.Clone(masters)
	slices.Sort(names)
	names = slices.Compact(names)
	return Scope{Kind: KindCluster, Name: strings.Join(names, ",")}
}

// ClusterMasters returns the control-plane nodes a cluster scope (or a range query of one) names.
func ClusterMasters(name string) []string {
	if name == "" {
		return nil
	}
	return strings.Split(name, ",")
}

// clusterInstant asks a provider for every node's current values: the cluster's live value.
func clusterInstant(ctx context.Context, prov Provider, ds *dataSet) error {
	ds.nodes = map[string]model.NodeUsage{}
	l, err := prov.ListNodes(ctx)
	if err != nil {
		return err
	}
	for _, n := range l {
		ds.nodes[n.Name] = n
	}
	if len(l) == 0 {
		return ErrNoData
	}
	return nil
}

// clusterSums adds up the nodes' usage per target: every node, the masters, the workers. A part is
// missing from a sum only when every node in it lacks it; at is the newest measurement's time.
func clusterSums(nodes map[string]model.NodeUsage, masters []string) (all, master, worker model.Usage, hasMaster, hasWorker bool, at time.Time) {
	first := map[string]bool{}
	add := func(target string, sum *model.Usage, u model.Usage) {
		if !first[target] {
			*sum, first[target] = u, true
			return
		}
		*sum = sum.Add(u)
	}
	for name, n := range nodes {
		add("", &all, n.Usage)
		if slices.Contains(masters, name) {
			add(TargetMaster, &master, n.Usage)
		} else {
			add(TargetWorker, &worker, n.Usage)
		}
		if n.Timestamp.After(at) {
			at = n.Timestamp
		}
	}
	return all, master, worker, first[TargetMaster], first[TargetWorker], at
}

// addClusterLocked adds a live result of every node to the cluster's chart: the three sums at the
// newest measurement's time. A target without a node (a cluster without control-plane nodes has no
// masters) gets no point.
func (h *Hub) addClusterLocked(ch *chart, sc Scope, ds *dataSet, gapAfter time.Duration) bool {
	if len(ds.nodes) == 0 {
		return false
	}
	all, master, worker, hasMaster, hasWorker, at := clusterSums(ds.nodes, ClusterMasters(sc.Name))
	if at.IsZero() {
		at = h.opts.Now()
	}
	added := ch.addUsage("", model.UsageSample{At: at, Usage: all}, gapAfter)
	if hasMaster {
		ch.addUsage(TargetMaster, model.UsageSample{At: at, Usage: master}, 0)
	}
	if hasWorker {
		ch.addUsage(TargetWorker, model.UsageSample{At: at, Usage: worker}, 0)
	}
	return added
}

// seedClusterLocked starts the cluster's chart with a fresh list of the nodes' measurements (the
// Nodes view's), so the page shows a value before its first poll answered.
func (h *Hub) seedClusterLocked(st *scopeState) {
	if ls := h.scopes[Scope{Kind: KindNodes}]; ls != nil && h.validLocked(ls) {
		h.addClusterLocked(st.chart, st.scope, ls.data, 0)
	}
}
