package kinds

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
)

// The CPU and Memory columns of the Pods and Nodes lists (resource-parity: Pods and Nodes; Lens
// places them after Namespace and after the warning icon). They show the usage the Metrics API
// reports and stay empty without it (architecture §9).
const (
	titleCPU    = "CPU"
	titleMemory = "Memory"
)

var (
	cpuColumn    = model.Column{ID: model.ColumnCPU, Title: titleCPU, Type: model.ColumnMetric}
	memoryColumn = model.Column{ID: model.ColumnMemory, Title: titleMemory, Type: model.ColumnMetric}
)

// podUsageCells shows a pod's usage as Lens's Pods list does: CPU in cores with three decimals
// ("0.012"), memory in binary units ("19.5MiB"). Num is the sort key: millicores and bytes.
func podUsageCells(_ *unstructured.Unstructured, u model.Usage) (cpu, memory model.Cell) {
	return model.Cell{Text: model.FormatCPU(u.CPUMilli), Num: u.CPUMilli},
		model.Cell{Text: model.FormatBytes(u.MemoryBytes), Num: u.MemoryBytes}
}

// nodeUsageCells shows a node's usage as a share of what the node can allocate ("7.5%"), the text
// of Lens's usage bars (which Cogpit does not draw yet); the share, in tenths of a percent, is the
// sort key. A node that reports no allocatable amount shows the absolute usage, as a pod does.
func nodeUsageCells(obj *unstructured.Unstructured, u model.Usage) (cpu, memory model.Cell) {
	alloc, _ := model.NodeCapacity(obj)
	cpu, memory = podUsageCells(obj, u)
	if alloc.CPUMilli > 0 {
		share := float64(u.CPUMilli) / float64(alloc.CPUMilli)
		cpu = model.Cell{Text: model.FormatPercent(share), Num: int64(share * 1000)}
	}
	if alloc.MemoryBytes > 0 {
		share := float64(u.MemoryBytes) / float64(alloc.MemoryBytes)
		memory = model.Cell{Text: model.FormatPercent(share), Num: int64(share * 1000)}
	}
	return cpu, memory
}
