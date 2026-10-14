package metrics

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// The cluster's totals from Prometheus (the cluster Overview's history, hubcluster.go): the sums of
// CPU and memory over the nodes' root cgroups, cAdvisor's series the node charts read (prom.go,
// nodePlan), so that the history and the Metrics Server's live values (a node's usage is its root
// cgroup's) mean the same. Every node is one query per metric without a node matcher; the masters
// are the control-plane nodes named by their node label or instance; the workers are every node
// less the masters, so no query names them and a cluster of thousands of workers sends a short one.

// clusterQueries are the queries of the cluster's chart at a rate window: CPU and memory over every
// node, and over the masters when the cluster has any ("" otherwise).
func clusterQueries(masters []string, win time.Duration) (cpuAll, memAll, cpuMaster, memMaster string) {
	cpuAll = fmt.Sprintf(`sum(rate(%s{id="/"}[%s]))`, metricCPU, promDuration(win))
	memAll = fmt.Sprintf(`sum(%s{id="/"})`, metricMemory)
	if len(masters) == 0 {
		return cpuAll, memAll, "", ""
	}
	quoted := make([]string, len(masters))
	for i, m := range masters {
		quoted[i] = regexp.QuoteMeta(m)
	}
	names := "(" + strings.Join(quoted, "|") + ")"
	group := func(metric string, rate bool) string {
		sels := [2]string{
			fmt.Sprintf(`%s{id="/",node=~%s}`, metric, quoteLabel(names)),
			fmt.Sprintf(`%s{id="/",instance=~%s}`, metric, quoteLabel(names+"(:[0-9]+)?")),
		}
		f := func(s string) string {
			if rate {
				return fmt.Sprintf("sum(rate(%s[%s]))", s, promDuration(win))
			}
			return "sum(" + s + ")"
		}
		return f(sels[0]) + " or " + f(sels[1])
	}
	return cpuAll, memAll, group(metricCPU, true), group(metricMemory, false)
}

// clusterRange answers a range query of the cluster scope: the targets "" (every node), master and
// worker. CPU and memory (WantUsage, every query must answer) and the pod counts (WantPods,
// kube-state-metrics': a failure leaves them out) exist for the cluster; a query that wants neither
// costs nothing.
func (p *Prometheus) clusterRange(ctx context.Context, q RangeQuery) (*ObjectSeries, error) {
	out := &ObjectSeries{Targets: map[string]*TargetSeries{}, From: q.From, To: q.To, Step: q.Step}
	if q.Want.Has(WantPods) {
		p.clusterPods(ctx, q, out)
	}
	if q.Want != 0 && !q.Want.Has(WantUsage) {
		return out, nil
	}
	masters := ClusterMasters(q.Name)
	cpuAll, memAll, cpuMaster, memMaster := clusterQueries(masters, rangeRate(q.Step))
	run := func(expr string) ([]promSeries, error) { return p.queryRange(ctx, expr, q.From, q.To, q.Step) }
	var (
		res  [4][]promSeries
		errs [4]error
		jobs []func()
	)
	for i, expr := range []string{cpuAll, memAll, cpuMaster, memMaster} {
		if expr != "" {
			jobs = append(jobs, func() { res[i], errs[i] = run(expr) })
		}
	}
	runParallel(jobs...)
	if err := firstStateError(errs[:]...); err != nil {
		return nil, err
	}
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
	if len(masters) == 0 {
		set(TargetWorker, allCPU, allMem)
		return out, nil
	}
	mCPU, mMem := first(res[2]), first(res[3])
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
