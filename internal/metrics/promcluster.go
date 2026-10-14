package metrics

import (
	"context"
	"fmt"
	"time"
)

// The cluster's totals from Prometheus (the cluster Overview's history, hubcluster.go): the sums of
// CPU and memory over the nodes, from the series the node charts read (promrange.go, nodePlan):
// node-exporter's where the Prometheus has them (A43; kube-prometheus-stack drops cAdvisor's root
// cgroup), else the nodes' root cgroups, so that the history and the Metrics Server's live values
// (a node's usage is its root cgroup's working set) mean the same. Every node is one query per
// metric without a node matcher (every exporter, every root cgroup the Prometheus scrapes); the
// masters are the control-plane nodes named by their node label or instance (their names or
// addresses); the workers are every node less the masters, so no query names them and a cluster of
// thousands of workers sends a short one.

// clusterQueries are the queries of the cluster's chart at a rate window: CPU and memory over every
// node, and over the masters when the cluster has any ("" otherwise), from cAdvisor's root
// cgroups. The masters are found by their names and addresses (matchGroup).
func clusterQueries(masters, addrs []string, win time.Duration) (cpuAll, memAll, cpuMaster, memMaster string) {
	cpuAll = fmt.Sprintf(`sum(rate(%s{id="/"}[%s]))`, metricCPU, promDuration(win))
	memAll = fmt.Sprintf(`sum(%s{id="/"})`, metricMemory)
	if len(masters) == 0 {
		return cpuAll, memAll, "", ""
	}
	m := matchGroup(masters, addrs)
	return cpuAll, memAll, nodeSum(metricCPU, m, win), nodeSum(metricMemory, m, 0)
}

// clusterExporterQueries are clusterQueries from node-exporter's series (A43): every exporter the
// Prometheus scrapes, and the masters' by the node label or the instance.
func clusterExporterQueries(masters, addrs []string, win time.Duration) (cpuAll, memAll, cpuMaster, memMaster string) {
	cpuAll, memAll = exporterCPU("", win), exporterMemory("")
	if len(masters) == 0 {
		return cpuAll, memAll, "", ""
	}
	m := matchGroup(masters, addrs)
	return cpuAll, memAll, exporterOf(func(x string) string { return exporterCPU(x, win) }, m), exporterOf(exporterMemory, m)
}

// clusterRange answers a range query of the cluster scope: the targets "" (every node), master and
// worker. CPU and memory (WantUsage, every query must answer) and the pod counts (WantPods,
// kube-state-metrics': a failure leaves them out) exist for the cluster; a query that wants neither
// costs nothing. CPU and memory come from node-exporter where this Prometheus has its series, else
// from cAdvisor's root cgroups, as a node's chart's do (nodeSource, A43), and UsageSource says
// which.
func (p *Prometheus) clusterRange(ctx context.Context, q RangeQuery) (*ObjectSeries, error) {
	out := &ObjectSeries{Targets: map[string]*TargetSeries{}, From: q.From, To: q.To, Step: q.Step}
	if q.Want.Has(WantPods) {
		p.clusterPods(ctx, q, out)
	}
	if q.Want != 0 && !q.Want.Has(WantUsage) {
		return out, nil
	}
	masters := ClusterMasters(q.Name)
	win := rangeRate(q.Step)
	run := func(expr string) ([]promSeries, error) { return p.queryRange(ctx, expr, q.From, q.To, q.Step) }
	var res [4][]promSeries
	ask := func(exprs ...string) (bool, error) {
		var errs [4]error
		res = [4][]promSeries{}
		var jobs []func()
		for i, expr := range exprs {
			if expr != "" {
				jobs = append(jobs, func() { res[i], errs[i] = run(expr) })
			}
		}
		runParallel(jobs...)
		if err := firstStateError(errs[:]...); err != nil {
			return false, err
		}
		return hasPoints(res[0]) || hasPoints(res[1]), nil // every node's: the masters' may lack an exporter
	}
	exporter := func() (bool, error) {
		a, b, c, d := clusterExporterQueries(masters, q.Addresses, win)
		return ask(a, b, c, d)
	}
	cadvisor := func() (bool, error) {
		a, b, c, d := clusterQueries(masters, q.Addresses, win)
		return ask(a, b, c, d)
	}
	source, err := p.nodeSource(exporter, cadvisor)
	if err != nil {
		return nil, err
	}
	out.UsageSource = source
	first := func(s []promSeries) []point {
		if len(s) == 0 {
			return nil
		}
		return s[0].points // a sum without "by" is one series
	}
	set := func(target string, cpu, mem []point) {
		if u := mergeUsage(cpu, mem); len(u) > 0 {
			targetOf(out, target).Usage = u
		}
	}
	allCPU, allMem := first(res[0]), first(res[1])
	set("", allCPU, allMem)
	mCPU, mMem := first(res[2]), first(res[3])
	if len(masters) == 0 || source == UsageNodeExporter && len(mCPU) == 0 && len(mMem) == 0 {
		// No masters; or no exporter on the masters (a DaemonSet that does not tolerate their
		// taint): every exporter's sum is the workers', and the masters have no history.
		set(TargetWorker, allCPU, allMem)
		return out, nil
	}
	set(TargetMaster, mCPU, mMem)
	set(TargetWorker, subtract(allCPU, mCPU), subtract(allMem, mMem))
	return out, nil
}

// subtract is a - b at the times both have a point: the workers' sum is every node's less the
// masters'. A time only a has is left out (the masters' part is not known then).
func subtract(a, b []point) []point {
	var out []point
	joinHas(a, b, func(t time.Time, x, y float64, hasA, hasB bool) {
		if hasA && hasB {
			out = append(out, point{t: t.UnixMilli(), v: max(x-y, 0)})
		}
	})
	return out
}
