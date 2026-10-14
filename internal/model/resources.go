package model

import (
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// UsageKind says which Metrics API objects a kind's CPU and Memory columns show.
type UsageKind string

// Kinds with usage columns.
const (
	UsagePods  UsageKind = "pods"
	UsageNodes UsageKind = "nodes"
)

// The ids of the usage columns of a kind with a [KindProfile.Usage].
const (
	ColumnCPU    = "cpu"
	ColumnMemory = "memory"
	// ColumnDisk is the Nodes list's Disk column ([KindProfile.DiskCell]): the root filesystem,
	// from Prometheus's node-exporter series.
	ColumnDisk = "disk"
)

// ParseQuantity parses a Kubernetes resource quantity ("250m", "64Mi", "1.5") with a guard: it
// refuses input longer than any quantity and decimal exponents beyond ±30, which
// resource.ParseQuantity spends unbounded time on.
func ParseQuantity(s string) (resource.Quantity, bool) {
	if s == "" || len(s) > 64 {
		return resource.Quantity{}, false
	}
	if i := strings.IndexAny(s, "eE"); i >= 0 {
		if rest := s[i+1:]; rest != "" && rest != "i" { // "E" and "Ei" are suffixes
			exp, err := strconv.Atoi(rest)
			if err != nil || exp < -30 || exp > 30 {
				return resource.Quantity{}, false
			}
		}
	}
	q, err := resource.ParseQuantity(s)
	return q, err == nil
}

// UsageOf reads a resource list ({"cpu": "250m", "memory": "64Mi"}) at the given path of an
// object as a Usage; a missing or unparseable entry is 0 ("not set").
func UsageOf(root any, fields ...string) Usage {
	list := NestedMap(root, fields...)
	var u Usage
	if q, ok := ParseQuantity(NestedIntOrString(list, "cpu")); ok {
		u.CPUMilli = q.MilliValue()
	}
	if q, ok := ParseQuantity(NestedIntOrString(list, "memory")); ok {
		u.MemoryBytes = q.Value()
	}
	return u
}

// ContainerResources are one container's CPU and memory requests and limits; 0 means not set.
type ContainerResources struct {
	Name     string `json:"name"`
	Requests Usage  `json:"requests"`
	Limits   Usage  `json:"limits"`
}

// PodResources returns the requests and limits of a pod's containers (spec.containers; init
// containers are not measured by the Metrics API), in spec order.
func PodResources(pod *unstructured.Unstructured) []ContainerResources {
	var out []ContainerResources
	for _, c := range NestedSlice(pod.Object, "spec", "containers") {
		name := NestedString(c, "name")
		if name == "" {
			continue
		}
		out = append(out, ContainerResources{
			Name:     name,
			Requests: UsageOf(c, "resources", "requests"),
			Limits:   UsageOf(c, "resources", "limits"),
		})
	}
	return out
}

// PodTotals returns what the pod as a whole requests and may use: the sum of its containers'
// requests, and the sum of their limits only when every container sets that limit (a container
// without a CPU limit makes the pod's CPU unbounded: no line to draw), for CPU and memory
// separately.
func PodTotals(cs []ContainerResources) (requests, limits Usage) {
	cpuLimited, memLimited := len(cs) > 0, len(cs) > 0
	for _, c := range cs {
		requests = requests.Add(c.Requests)
		limits = limits.Add(c.Limits)
		cpuLimited = cpuLimited && c.Limits.CPUMilli > 0
		memLimited = memLimited && c.Limits.MemoryBytes > 0
	}
	if !cpuLimited {
		limits.CPUMilli = 0
	}
	if !memLimited {
		limits.MemoryBytes = 0
	}
	return requests, limits
}

// NodeCapacity returns a node's allocatable and capacity (status.allocatable, status.capacity).
func NodeCapacity(node *unstructured.Unstructured) (allocatable, capacity Usage) {
	return UsageOf(node.Object, "status", "allocatable"), UsageOf(node.Object, "status", "capacity")
}
