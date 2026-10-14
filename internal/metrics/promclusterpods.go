package metrics

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/guided-traffic/cogpit/internal/model"
)

// The cluster Overview's Pods history (G12): the pods on the cluster's nodes over the chart's range,
// from kube-state-metrics through Prometheus, asked for once when the page opens or its range
// changes; the cache counts the pods after that (engine/clusteroverview_charts.go). The count is
// the cache's rule: the pods scheduled to a node of the group that have not finished (neither
// Succeeded nor Failed), each counted once however many kube-state-metrics replicas or jobs report
// it. Without kube-state-metrics the queries answer nothing and the chart keeps the cache's counts
// alone; the source says "none".

// Where the pod counts come from (ObjectSeries.PodsSource).
const (
	PodsFromKubeStateMetrics = "kube-state-metrics"
	PodsFromNowhere          = "none"
)

// podCountQuery counts the pods that are not finished on the nodes the matcher selects
// (`node!=""`: every scheduled pod; `node=~"(m1|m2)"`: the masters').
func podCountQuery(nodeMatcher string) string {
	return fmt.Sprintf(`count(max by (namespace, pod) (kube_pod_info{%s}) unless on (namespace, pod) `+
		`(max by (namespace, pod) (kube_pod_status_phase{phase=~"Succeeded|Failed"}) == 1))`, nodeMatcher)
}

// podCountQueries are the cluster's pod-count queries: every node's, and the masters' when there
// are masters ("" otherwise). Node names are quoted for the regular expression and the literal.
func podCountQueries(masters []string) (all, master string) {
	all = podCountQuery(`node!=""`)
	if len(masters) == 0 {
		return all, ""
	}
	quoted := make([]string, len(masters))
	for i, m := range masters {
		quoted[i] = regexp.QuoteMeta(m)
	}
	return all, podCountQuery("node=~" + quoteLabel("("+strings.Join(quoted, "|")+")"))
}

// clusterPods adds the pod counts to a cluster range answer: "" every node, master, worker (every
// node less the masters). A time every node has a count for and the masters none means no pod on a
// master then (count() of nothing answers nothing). A failed query leaves the counts out (the
// CPU and memory history does not wait for kube-state-metrics); no series at all is the source
// "none".
func (p *Prometheus) clusterPods(ctx context.Context, q RangeQuery, out *ObjectSeries) {
	masters := ClusterMasters(q.Name)
	allQ, masterQ := podCountQueries(masters)
	var (
		all, master []promSeries
		errAll      error
		errMaster   error
	)
	jobs := []func(){func() { all, errAll = p.queryRange(ctx, allQ, q.From, q.To, q.Step) }}
	if masterQ != "" {
		jobs = append(jobs, func() { master, errMaster = p.queryRange(ctx, masterQ, q.From, q.To, q.Step) })
	}
	runParallel(jobs...)
	if errAll != nil || errMaster != nil {
		p.opts.Logger.Debug("metrics: the pod counts failed", "class", errorClass(firstStateError(errAll, errMaster)))
		return
	}
	first := func(s []promSeries) []point {
		if len(s) == 0 {
			return nil
		}
		return s[0].points
	}
	allPts := first(all)
	if len(allPts) == 0 {
		out.PodsSource = PodsFromNowhere
		return
	}
	out.PodsSource = PodsFromKubeStateMetrics
	targetOf(out, "").Pods = counts(allPts)
	if len(masters) == 0 {
		targetOf(out, TargetWorker).Pods = counts(allPts)
		return
	}
	var mPts, wPts []point
	joinHas(allPts, first(master), func(t time.Time, a, m float64, hasA, hasM bool) {
		if !hasA {
			return
		}
		if !hasM {
			m = 0 // the masters ran no unfinished pod then
		}
		mPts = append(mPts, point{t: t.UnixMilli(), v: m})
		wPts = append(wPts, point{t: t.UnixMilli(), v: max(a-m, 0)})
	})
	targetOf(out, TargetMaster).Pods = counts(mPts)
	targetOf(out, TargetWorker).Pods = counts(wPts)
}

// counts turns points into count samples.
func counts(pts []point) []model.CountSample {
	out := make([]model.CountSample, 0, len(pts))
	for _, p := range pts {
		out = append(out, model.CountSample{At: time.UnixMilli(p.t), Value: p.v})
	}
	return out
}

// targetOf returns a target of an answer, made when missing.
func targetOf(out *ObjectSeries, name string) *TargetSeries {
	t := out.Targets[name]
	if t == nil {
		t = &TargetSeries{}
		out.Targets[name] = t
	}
	return t
}
