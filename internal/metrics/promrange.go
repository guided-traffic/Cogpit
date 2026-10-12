package metrics

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/guided-traffic/cogpit/internal/model"
)

// rangePlan is the set of queries behind one object's charts.
type rangePlan struct {
	cpu, mem, rx, tx           string
	fsUsage, fsReads, fsWrites string
	// sumWhole: the whole object's usage is the sum of its containers' (a pod); false when the
	// queries already give the whole (a node).
	sumWhole bool
}

// podPlan is a pod's queries, the counters' rates over win (rangeRate).
func podPlan(namespace, pod string, win time.Duration) rangePlan {
	return rangePlan{
		cpu: podRate(metricCPU, namespace, pod, win), mem: podGauge(metricMemory, namespace, pod),
		rx: podNetwork(metricNetRx, namespace, pod, win), tx: podNetwork(metricNetTx, namespace, pod, win),
		fsUsage: podGauge(metricFSUsage, namespace, pod),
		fsReads: podRate(metricFSReads, namespace, pod, win), fsWrites: podRate(metricFSWrites, namespace, pod, win),
		sumWhole: true,
	}
}

// nodePlan reads a node's root cgroup. node-exporter's node_* metrics are not used: they are
// labelled by the exporter's instance (an address), which has no reliable link to the node's
// name, while cAdvisor's series carry the node label or the node's name as the instance.
func nodePlan(node string, win time.Duration) rangePlan {
	return rangePlan{
		cpu: nodeSum(metricCPU, node, win), mem: nodeSum(metricMemory, node, 0),
		rx: nodeSum(metricNetRx, node, win), tx: nodeSum(metricNetTx, node, win),
		fsUsage: nodeSumWhere(metricFSUsage, nodeDisks, node, 0),
		fsReads: nodeSum(metricFSReads, node, win), fsWrites: nodeSum(metricFSWrites, node, win),
	}
}

type fsData struct {
	source               string // "usage", "io" or ""
	usage, reads, writes []promSeries
}

// rangeFilesystem asks for the filesystem metric this Prometheus has for the kind's objects. The
// first poll of a kind discovers it by trying: the usage metric first, the read and write rates
// when there is no usage series; the answer is remembered per kind (pods, nodes) for the
// provider's life. While neither exists it tries both every time.
func (p *Prometheus) rangeFilesystem(kind Kind, plan rangePlan, run func(string) ([]promSeries, error)) (fsData, error) {
	known := &p.fsMode[fsKind(kind)]
	mode := known.Load()
	if mode == fsUnknown || mode == fsUsage {
		usage, err := run(plan.fsUsage)
		if err != nil {
			return fsData{}, err
		}
		if hasPoints(usage) {
			known.Store(fsUsage)
			return fsData{source: FilesystemUsage, usage: usage}, nil
		}
		if mode == fsUsage { // had usage before: this object has none
			return fsData{}, nil
		}
	}
	var reads, writes []promSeries
	var rerr, werr error
	runParallel(
		func() { reads, rerr = run(plan.fsReads) },
		func() { writes, werr = run(plan.fsWrites) },
	)
	if err := firstStateError(rerr, werr); err != nil {
		return fsData{}, err
	}
	if hasPoints(reads) || hasPoints(writes) {
		known.Store(fsIO)
		return fsData{source: FilesystemIO, reads: reads, writes: writes}, nil
	}
	return fsData{}, nil
}

func hasPoints(s []promSeries) bool {
	for _, x := range s {
		if len(x.points) > 0 {
			return true
		}
	}
	return false
}

// ObjectRange implements RangeProvider. For a pod it sends what q.Want asks for, all at once: CPU
// and memory per container (two queries), the pod's network received and transmitted (two), the
// filesystem per container (one, two with the read and write rates, three while neither
// filesystem metric is known to exist); a zero Want asks for all of them, five queries. With the
// usage asked for, the CPU and memory queries must succeed and a failing network or filesystem
// query leaves its series out; without it, the query fails only when every query failed.
func (p *Prometheus) ObjectRange(ctx context.Context, q RangeQuery) (*ObjectSeries, error) {
	if q.Step <= 0 || !q.To.After(q.From) {
		return nil, fmt.Errorf("invalid range %s..%s step %s", q.From, q.To, q.Step)
	}
	var plan rangePlan
	switch q.Kind {
	case KindPods:
		plan = podPlan(q.Namespace, q.Name, rangeRate(q.Step))
	case KindNodes:
		plan = nodePlan(q.Name, rangeRate(q.Step))
	default:
		return nil, fmt.Errorf("kind %q has no metrics", q.Kind)
	}
	want := q.Want
	if want == 0 {
		want = WantAll
	}
	run := func(expr string) ([]promSeries, error) { return p.queryRange(ctx, expr, q.From, q.To, q.Step) }

	var (
		cpu, mem, rx, tx    []promSeries
		cpuErr, memErr      error
		rxErr, txErr, fsErr error
		fs                  fsData
		jobs                []func()
	)
	if want.Has(WantUsage) {
		jobs = append(jobs, func() { cpu, cpuErr = run(plan.cpu) }, func() { mem, memErr = run(plan.mem) })
	}
	if want.Has(WantNetwork) {
		jobs = append(jobs, func() { rx, rxErr = run(plan.rx) }, func() { tx, txErr = run(plan.tx) })
	}
	if want.Has(WantFilesystem) {
		jobs = append(jobs, func() { fs, fsErr = p.rangeFilesystem(q.Kind, plan, run) })
	}
	runParallel(jobs...)
	if want.Has(WantUsage) {
		if err := firstStateError(cpuErr, memErr); err != nil {
			return nil, err
		}
	} else {
		netFailed := !want.Has(WantNetwork) || rxErr != nil || txErr != nil
		fsFailed := !want.Has(WantFilesystem) || fsErr != nil
		if netFailed && fsFailed {
			return nil, firstStateError(rxErr, txErr, fsErr)
		}
	}
	for _, e := range []error{rxErr, txErr, fsErr} {
		if e != nil && ctx.Err() == nil {
			p.opts.Logger.Debug("metrics: an optional query failed", "service", p.target.String(), "class", errorClass(e))
		}
	}

	out := &ObjectSeries{Targets: map[string]*TargetSeries{}, From: q.From, To: q.To, Step: q.Step}
	target := func(name string) *TargetSeries {
		t := out.Targets[name]
		if t == nil {
			t = &TargetSeries{}
			out.Targets[name] = t
		}
		return t
	}

	cpuBy, memBy := groupByContainer(cpu), groupByContainer(mem)
	if plan.sumWhole {
		cpuBy[""], memBy[""] = sumPoints(cpuBy), sumPoints(memBy)
	}
	for _, name := range unionKeys(cpuBy, memBy) {
		if u := mergeUsage(cpuBy[name], memBy[name]); len(u) > 0 {
			target(name).Usage = u
		}
	}

	if want.Has(WantNetwork) && rxErr == nil && txErr == nil {
		if n := mergeNetwork(groupByContainer(rx)[""], groupByContainer(tx)[""]); len(n) > 0 {
			target("").Network = n
		}
	}

	if want.Has(WantFilesystem) && fsErr == nil && fs.source != "" {
		out.FilesystemSource = fs.source
		var a, b map[string][]point // usage, or reads and writes
		if fs.source == FilesystemUsage {
			a = groupByContainer(fs.usage)
		} else {
			a, b = groupByContainer(fs.reads), groupByContainer(fs.writes)
		}
		if plan.sumWhole {
			a[""] = sumPoints(a)
			if b != nil {
				b[""] = sumPoints(b)
			}
		}
		for _, name := range unionKeys(a, b) {
			if f := mergeFilesystem(fs.source, a[name], b[name]); len(f) > 0 {
				target(name).Filesystem = f
			}
		}
	}
	return out, nil
}

// groupByContainer groups a result by its container label; a series without one (the pod's
// network, a node) is under "". Two series of one container (it should not happen after the
// queries' sum by) are summed.
func groupByContainer(series []promSeries) map[string][]point {
	out := map[string][]point{}
	for _, s := range series {
		name := s.labels["container"]
		if prev, ok := out[name]; ok {
			out[name] = addPoints(prev, s.points)
		} else {
			out[name] = s.points
		}
	}
	return out
}

// sumPoints adds the series of every key at equal timestamps: a pod's total from its containers.
func sumPoints(by map[string][]point) []point {
	var out []point
	for _, k := range sortedKeys(by) {
		if k == "" { // an earlier whole, not a container
			continue
		}
		out = addPoints(out, by[k])
	}
	return out
}

// addPoints returns the sum of two series, by timestamp, oldest first; a timestamp only one has
// counts as is.
func addPoints(a, b []point) []point {
	m := make(map[int64]float64, len(a)+len(b))
	for _, p := range a {
		m[p.t] += p.v
	}
	for _, p := range b {
		m[p.t] += p.v
	}
	out := make([]point, 0, len(m))
	for t, v := range m {
		out = append(out, point{t, v})
	}
	slices.SortFunc(out, func(x, y point) int { return int(x.t - y.t) })
	return out
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

func unionKeys[V any](a, b map[string]V) []string {
	m := map[string]struct{}{}
	for k := range a {
		m[k] = struct{}{}
	}
	for k := range b {
		m[k] = struct{}{}
	}
	return sortedKeys(m)
}

// join walks two series by timestamp (the union, oldest first); a missing side is 0.
func join(a, b []point, emit func(t time.Time, a, b float64)) {
	joinHas(a, b, func(t time.Time, a, b float64, _, _ bool) { emit(t, a, b) })
}

// joinHas walks two series by timestamp and says which side has a point at each time.
func joinHas(a, b []point, emit func(t time.Time, a, b float64, hasA, hasB bool)) {
	type side struct {
		v   [2]float64
		has [2]bool
	}
	ma := make(map[int64]side, len(a)+len(b))
	for _, p := range a {
		e := ma[p.t]
		e.v[0], e.has[0] = p.v, true
		ma[p.t] = e
	}
	for _, p := range b {
		e := ma[p.t]
		e.v[1], e.has[1] = p.v, true
		ma[p.t] = e
	}
	ts := make([]int64, 0, len(ma))
	for t := range ma {
		ts = append(ts, t)
	}
	slices.Sort(ts)
	for _, t := range ts {
		e := ma[t]
		emit(time.UnixMilli(t), e.v[0], e.v[1], e.has[0], e.has[1])
	}
}

// mergeUsage joins CPU (cores) and memory (bytes) into the Metrics Server's units. A time only one
// of them has is a sample with the other part missing, not 0: a chart leaves that point out of the
// other variant.
func mergeUsage(cpu, mem []point) []model.UsageSample {
	var out []model.UsageSample
	joinHas(cpu, mem, func(t time.Time, c, m float64, hasC, hasM bool) {
		u := model.Usage{CPUMilli: seconds2milli(c), MemoryBytes: int64(m)}
		if !hasC {
			u.Missing |= model.UsageCPU
		}
		if !hasM {
			u.Missing |= model.UsageMemory
		}
		out = append(out, model.UsageSample{At: t, Usage: u})
	})
	return out
}

func mergeNetwork(rx, tx []point) []model.NetworkSample {
	var out []model.NetworkSample
	join(rx, tx, func(t time.Time, r, x float64) {
		out = append(out, model.NetworkSample{At: t, ReceiveBytesPerSec: r, TransmitBytesPerSec: x})
	})
	return out
}

func mergeFilesystem(source string, a, b []point) []model.FilesystemSample {
	var out []model.FilesystemSample
	if source == FilesystemUsage {
		for _, p := range a {
			out = append(out, model.FilesystemSample{At: time.UnixMilli(p.t), UsageBytes: p.v})
		}
		return out
	}
	join(a, b, func(t time.Time, r, w float64) {
		out = append(out, model.FilesystemSample{At: t, ReadBytesPerSec: r, WriteBytesPerSec: w})
	})
	return out
}
