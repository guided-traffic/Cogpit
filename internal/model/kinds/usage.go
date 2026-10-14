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
	// The Nodes list's columns draw a usage bar beside the share, so they are wider than the Pods
	// list's numbers (Cogpit's width: "100%" and a bar of about 60 points).
	nodeCPUColumn    = model.Column{ID: model.ColumnCPU, Title: titleCPU, Type: model.ColumnMetric, Width: nodeBarWidth}
	nodeMemoryColumn = model.Column{ID: model.ColumnMemory, Title: titleMemory, Type: model.ColumnMetric, Width: nodeBarWidth}
)

// nodeBarWidth is the default width of a Nodes list's usage-bar column.
const nodeBarWidth = 112

// podUsageCells shows a pod's usage as Lens's Pods list does: CPU in cores with three decimals
// ("0.012"), memory in binary units with one decimal (D-32: "19.5 MiB").
// Num is the sort key: millicores and bytes. A part the provider did not measure (Usage.Missing)
// is an empty cell, never "0".
func podUsageCells(_ *unstructured.Unstructured, u model.Usage) (cpu, memory model.Cell) {
	if u.HasCPU() {
		cpu = model.Cell{Text: model.FormatCPU(u.CPUMilli), Num: u.CPUMilli}
	}
	if u.HasMemory() {
		memory = model.Cell{Text: model.FormatBytes(u.MemoryBytes), Num: u.MemoryBytes}
	}
	return cpu, memory
}

// nodeUsageCells shows a node's usage as a share of what the node can allocate ("7.5%") drawn as
// a usage bar (Lens's Nodes list), the amounts behind it as the bar's detail ("0.300 of 4.000
// cores"); the share, in tenths of a percent, is the sort key. A node that reports no allocatable
// amount shows the absolute usage without a bar, as a pod does.
func nodeUsageCells(obj *unstructured.Unstructured, u model.Usage) (cpu, memory model.Cell) {
	alloc, _ := model.NodeCapacity(obj)
	cpu, memory = podUsageCells(obj, u)
	if alloc.CPUMilli > 0 && u.HasCPU() {
		cpu = model.ShareCell(float64(u.CPUMilli)/float64(alloc.CPUMilli),
			model.FormatCPU(u.CPUMilli)+" of "+model.FormatCPU(alloc.CPUMilli)+" cores")
	}
	if alloc.MemoryBytes > 0 && u.HasMemory() {
		memory = model.ShareCell(float64(u.MemoryBytes)/float64(alloc.MemoryBytes),
			model.FormatBytes(u.MemoryBytes)+" of "+model.FormatBytes(alloc.MemoryBytes))
	}
	return cpu, memory
}
