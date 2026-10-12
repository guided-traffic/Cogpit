package kinds_test

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

func TestUsageColumns(t *testing.T) {
	var withUsage int
	for _, p := range kinds.Builtin() {
		if p.Usage == "" {
			continue
		}
		withUsage++
		cpu, mem := p.ColumnIndex(model.ColumnCPU), p.ColumnIndex(model.ColumnMemory)
		if cpu < 0 || mem < 0 || p.Columns[cpu].Type != model.ColumnMetric || p.Columns[mem].Type != model.ColumnMetric || p.UsageCells == nil {
			t.Errorf("%s: usage kind %q without CPU and Memory metric columns and UsageCells", p.Key, p.Usage)
		}
	}
	if withUsage != 2 {
		t.Errorf("%d kinds with usage, want pods and nodes", withUsage)
	}
}

func TestPodsListLensOrder(t *testing.T) {
	p := profile(t, "pods")
	var ids []string
	for _, c := range p.Columns[:5] {
		ids = append(ids, c.ID)
	}
	if got := ids; got[0] != "name" || got[2] != "namespace" || got[3] != "cpu" || got[4] != "memory" {
		t.Errorf("first columns = %v, want Name, warning, Namespace, CPU, Memory (Lens)", got)
	}
}

func TestUsageCells(t *testing.T) {
	pods := profile(t, "pods")
	cpu, mem := pods.UsageCells(&unstructured.Unstructured{}, model.Usage{CPUMilli: 12, MemoryBytes: 20_447_232})
	if cpu.Text != "0.012" || cpu.Num != 12 || mem.Text != "19.5MiB" || mem.Num != 20_447_232 {
		t.Errorf("pod cells = %+v, %+v", cpu, mem)
	}

	nodes := profile(t, "nodes")
	node := &unstructured.Unstructured{Object: map[string]any{"status": map[string]any{
		"allocatable": map[string]any{"cpu": "4", "memory": "8Gi"},
	}}}
	cpu, mem = nodes.UsageCells(node, model.Usage{CPUMilli: 300, MemoryBytes: 2 << 30})
	if cpu.Text != "7.5%" || cpu.Num != 75 || mem.Text != "25%" || mem.Num != 250 {
		t.Errorf("node cells = %+v, %+v", cpu, mem)
	}
	// No allocatable amount reported: absolute usage, not a division by zero.
	cpu, _ = nodes.UsageCells(&unstructured.Unstructured{}, model.Usage{CPUMilli: 300})
	if cpu.Text != "0.300" {
		t.Errorf("node without allocatable: cpu = %+v", cpu)
	}
}

func profile(t *testing.T, key string) *model.KindProfile {
	t.Helper()
	for _, p := range kinds.Builtin() {
		if p.Key == key {
			return p
		}
	}
	t.Fatalf("no profile %q", key)
	return nil
}
