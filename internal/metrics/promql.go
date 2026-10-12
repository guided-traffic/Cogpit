package metrics

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// The PromQL the Prometheus provider sends (architecture §9). Every query is built here, and every
// label value that comes from user data (a pod's, a namespace's, a node's name) goes through
// quoteLabel, so a name can never end the string literal and add a matcher or a function. The
// queries use the cAdvisor metrics the kubelet exposes, which every Prometheus that scrapes a
// cluster has: container_cpu_usage_seconds_total, container_memory_working_set_bytes,
// container_network_*_bytes_total, container_fs_*.

// rateWindow is the range of the rate() queries of the current values (the lists, metrics.get) and
// the shortest of the charts'. Prometheus asks for at least four scrape intervals: the prometheus
// chart scrapes every minute, and cAdvisor stamps a sample with its own collection time, up to a
// minute before the scrape. On kind with that chart (2026-10-12) a [2m] range left 5 to 7 of 52
// points of a half-hour chart empty and the newest one or two minutes missing; [4m] left none.
const rateWindow = 4 * time.Minute

// rangeRate is the rate() range of a chart's query at step: every sample between two points counts
// (a step and a scrape interval, which Cogpit takes as a minute), and never less than rateWindow.
// A week's chart (a step of about ten minutes) so shows a spike between two of its points.
func rangeRate(step time.Duration) time.Duration { return max(rateWindow, step+time.Minute) }

// promDuration writes a range for PromQL in whole minutes or seconds: "4m", "665s".
func promDuration(d time.Duration) string {
	if d%time.Minute == 0 {
		return strconv.FormatInt(int64(d/time.Minute), 10) + "m"
	}
	return strconv.FormatInt(int64((d+time.Second-1)/time.Second), 10) + "s"
}

// quoteLabel returns s as a double-quoted PromQL string literal.
func quoteLabel(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 2)
	b.WriteByte('"')
	for _, r := range s {
		switch {
		case r == '\\':
			b.WriteString(`\\`)
		case r == '"':
			b.WriteString(`\"`)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case r == '\t':
			b.WriteString(`\t`)
		case r < 0x20 || r == 0x7f:
			fmt.Fprintf(&b, `\x%02x`, r)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// The metrics, by variant.
const (
	metricCPU      = "container_cpu_usage_seconds_total"
	metricMemory   = "container_memory_working_set_bytes"
	metricNetRx    = "container_network_receive_bytes_total"
	metricNetTx    = "container_network_transmit_bytes_total"
	metricFSUsage  = "container_fs_usage_bytes"
	metricFSReads  = "container_fs_reads_bytes_total"
	metricFSWrites = "container_fs_writes_bytes_total"
)

// containerMatchers keeps the series of real containers: the pod-level cgroup has container=""
// and the pause container "POD", and counting them would double every sum.
const containerMatchers = `container!="",container!="POD"`

// podMatchers selects a pod; namespace "" selects the pod in any namespace (never used for a
// named pod), name "" every pod of the namespace.
func podMatchers(namespace, pod string) string {
	var m []string
	if namespace != "" {
		m = append(m, "namespace="+quoteLabel(namespace))
	}
	if pod != "" {
		m = append(m, "pod="+quoteLabel(pod))
	}
	return strings.Join(m, ",")
}

// withContainers builds {matchers,container!="",container!="POD"}.
func withContainers(matchers string) string {
	if matchers == "" {
		return "{" + containerMatchers + "}"
	}
	return "{" + matchers + "," + containerMatchers + "}"
}

// podRate is the per-container rate of a counter over win: sum by (container) (rate(metric{...}[4m])).
func podRate(metric, namespace, pod string, win time.Duration) string {
	return fmt.Sprintf("sum by (container) (rate(%s%s[%s]))", metric, withContainers(podMatchers(namespace, pod)), promDuration(win))
}

// podGauge is the per-container value of a gauge.
func podGauge(metric, namespace, pod string) string {
	return fmt.Sprintf("sum by (container) (%s%s)", metric, withContainers(podMatchers(namespace, pod)))
}

// podNetwork is the pod's traffic: network metrics belong to the pod's network namespace, so they
// are summed over the pod's interfaces without a container matcher.
func podNetwork(metric, namespace, pod string, win time.Duration) string {
	return fmt.Sprintf("sum(rate(%s{%s}[%s]))", metric, podMatchers(namespace, pod), promDuration(win))
}

// podListRate and podListGauge are the list forms: one series per container of every pod of a
// namespace (or of every namespace).
func podListRate(metric, namespace string) string {
	return fmt.Sprintf("sum by (namespace, pod, container) (rate(%s%s[%s]))", metric, withContainers(podMatchers(namespace, "")), promDuration(rateWindow))
}

func podListGauge(metric, namespace string) string {
	return fmt.Sprintf("sum by (namespace, pod, container) (%s%s)", metric, withContainers(podMatchers(namespace, "")))
}

// nodeSelectors are the two ways the series of one node's root cgroup are labelled: with the
// kube-prometheus-stack's node label, or with the instance (the node's name, which the Prometheus
// chart's cAdvisor job gives, with or without the kubelet's port).
func nodeSelectors(metric, extra, node string) [2]string {
	root := `id="/"`
	if extra != "" {
		root += "," + extra
	}
	return [2]string{
		fmt.Sprintf(`%s{%s,node=%s}`, metric, root, quoteLabel(node)),
		fmt.Sprintf(`%s{%s,instance=~%s}`, metric, root, quoteLabel(regexp.QuoteMeta(node)+"(:[0-9]+)?")),
	}
}

// nodeDisks keeps the node's block devices among its root cgroup's filesystem series. cAdvisor
// reports every mount there, and the overlay mounts of the containers repeat the disk they live
// on, so a sum over all of them counts one disk many times (kind, 2026-10-12: nine overlays of
// /dev/vda1 summed to 503 GiB on a 56 GiB disk). /dev/shm is memory.
const nodeDisks = `device=~"/dev/.+",device!="/dev/shm"`

// nodeSum sums the root cgroup of one node: a gauge as is (win 0), a counter as its rate over win.
// The two selectors are joined by "or", which yields the second only when the first matches
// nothing.
func nodeSum(metric, node string, win time.Duration) string {
	return nodeSumWhere(metric, "", node, win)
}

// nodeSumWhere is nodeSum over the series that also match extra.
func nodeSumWhere(metric, extra, node string, win time.Duration) string {
	sel := nodeSelectors(metric, extra, node)
	f := func(s string) string {
		if win > 0 {
			return fmt.Sprintf("sum(rate(%s[%s]))", s, promDuration(win))
		}
		return "sum(" + s + ")"
	}
	return f(sel[0]) + " or " + f(sel[1])
}

// nodeListSum is the list form: one series per node, labelled by node and instance.
func nodeListSum(metric string, rate bool) string {
	if rate {
		return fmt.Sprintf(`sum by (node, instance) (rate(%s{id="/"}[%s]))`, metric, promDuration(rateWindow))
	}
	return fmt.Sprintf(`sum by (node, instance) (%s{id="/"})`, metric)
}
