package kinds_test

import (
	"fmt"
	"path/filepath"
	"slices"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

// nodeDeviceSlices are the resource slices of the fixtures: worker-a-01's GPUs, NICs and an empty
// pool, a per-device slice with one device each on worker-a-01 and worker-a-02, worker-a-02's GPU
// and a pool every node reaches.
func nodeDeviceSlices(t testing.TB) []*unstructured.Unstructured {
	t.Helper()
	out := []*unstructured.Unstructured{
		recorded(t, "resourceslices", "healthy"),   // worker-a-01, gpu.example.com
		recorded(t, "resourceslices", "failing"),   // worker-a-02
		recorded(t, "resourceslices", "all-nodes"), // allNodes
	}
	for _, c := range loadCases(t, "nodedevices") {
		out = append(out, c.obj.DeepCopy())
	}
	return out
}

// nodeNamed is the recorded healthy node renamed.
func nodeNamed(t testing.TB, name string) *unstructured.Unstructured {
	n := recorded(t, "nodes", "healthy")
	n.SetName(name)
	return n
}

func sectionTitles(sections []model.DetailSection) []string {
	out := make([]string, len(sections))
	for i, s := range sections {
		out[i] = s.Group + "/" + s.Title
	}
	return out
}

// TestSlicesByNode: a slice is found under the node it names, a per-device slice under each of
// its devices' nodes, a slice for all nodes or a node selector under none.
func TestSlicesByNode(t *testing.T) {
	ix := kinds.SlicesByNode
	if ix.Kind != "resourceslices" {
		t.Fatalf("SlicesByNode indexes %q", ix.Kind)
	}
	for _, c := range []struct {
		obj  *unstructured.Unstructured
		want []string
	}{
		{recorded(t, "resourceslices", "healthy"), []string{"worker-a-01"}},
		{recorded(t, "resourceslices", "all-nodes"), nil},
		{recorded(t, "resourceslices", "node-selector"), nil},
		{recorded(t, "nodedevices", "per-device"), []string{"worker-a-01", "worker-a-02"}},
	} {
		if got := ix.Keys(c.obj); !slices.Equal(got, c.want) {
			t.Errorf("%s: keys %q, want %q", c.obj.GetName(), got, c.want)
		}
	}
	p, _ := kinds.ByKey("nodes")
	if !slices.ContainsFunc(p.Needs, func(n model.RelatedIndex) bool { return n.ID() == ix.ID() }) {
		t.Error("the Nodes profile does not declare SlicesByNode")
	}
}

// TestNodeDevicesSection: a node's drawer lists, after Allocatable and before the Pods, one section
// per resource slice that names the node, under a Devices heading, sorted by driver: its pool, a
// link to the slice, how many devices it publishes for the node and a table of them (a per-device
// slice only the node's own); a Taints column only when a device has taints. The golden file holds
// the sections. Without the slices (not served, not listed, none naming the node) there are none.
func TestNodeDevicesSection(t *testing.T) {
	p, _ := kinds.ByKey("nodes")
	node := nodeNamed(t, "worker-a-01")
	rel := fakeRelated{"resourceslices": nodeDeviceSlices(t), "pods": {podOn(t, "healthy", "ml", "trainer-0", "worker-a-01")}}
	sections := p.DetailSections(node, rel)
	want := []string{"/Properties", "/Capacity", "/Allocatable",
		"Devices/accel.example.com", "Devices/gpu.example.com", "Devices/nic.example.net", "Devices/tpu.example.com", "/Pods"}
	if got := sectionTitles(sections); !slices.Equal(got, want) {
		t.Fatalf("sections %q, want %q", got, want)
	}
	devices := sections[3:7]
	checkGolden(t, filepath.Join("testdata", "nodedevices", "worker-a-01.golden.json"), encode(t, devices))

	accel, gpu, nic, tpu := devices[0], devices[1], devices[2], devices[3]
	if accel.Table == nil || len(accel.Table.Rows) != 1 || accel.Table.Rows[0][0] != "accel-0" || findItemIn(accel, "Devices") != "1" {
		t.Errorf("per-device slice: %+v", accel)
	}
	slice := findLink(accel, "Slice")
	if slice == nil || *slice != (model.ObjectRef{Kind: "resourceslices", Name: "rack-7-accel.example.com-m2k8d"}) {
		t.Errorf("the slice's link %+v", slice)
	}
	if gpu.Table == nil || !slices.Equal(gpu.Table.Columns, []string{"Name", "Attributes", "Capacity"}) || len(gpu.Table.Rows) != 2 ||
		findItemIn(gpu, "Pool") != "worker-a-01" || findItemIn(gpu, "Devices") != "2" {
		t.Errorf("GPU slice: %+v", gpu)
	}
	if nic.Table == nil || !slices.Equal(nic.Table.Columns, []string{"Name", "Attributes", "Capacity", "Taints"}) ||
		nic.Table.Rows[1][3] != "nic.example.net/link-down:NoSchedule" {
		t.Errorf("NIC slice: %+v", nic.Table)
	}
	if tpu.Table != nil || findItemIn(tpu, "Devices") != "0" {
		t.Errorf("an empty pool: %+v", tpu)
	}

	// worker-a-02: its GPU slice and its device of the per-device slice; nothing of worker-a-01's.
	other := p.DetailSections(nodeNamed(t, "worker-a-02"), rel)
	if got := sectionTitles(other); !slices.Equal(got, []string{"/Properties", "/Capacity", "/Allocatable", "Devices/accel.example.com", "Devices/gpu.example.com"}) {
		t.Errorf("worker-a-02: %q", got)
	} else if other[3].Table.Rows[0][0] != "accel-1" {
		t.Errorf("worker-a-02's accelerator %q", other[3].Table.Rows)
	}

	// No slices: none listed, the kind not served, or none naming the node.
	for name, r := range map[string]model.Related{
		"not served":    fakeRelated{"pods": rel["pods"]},
		"no related":    nil,
		"another node":  fakeRelated{"resourceslices": nodeDeviceSlices(t)[1:3]},
		"empty listing": fakeRelated{"resourceslices": nil},
	} {
		for _, s := range p.DetailSections(node, r) {
			if s.Group == "Devices" {
				t.Errorf("%s: a Devices section %q", name, s.Title)
			}
		}
	}
}

// TestNodeDevicesBounded: past 32 slices the rest are counted in a last section.
func TestNodeDevicesBounded(t *testing.T) {
	p, _ := kinds.ByKey("nodes")
	base := recorded(t, "resourceslices", "healthy")
	var many []*unstructured.Unstructured
	for i := range 40 {
		s := base.DeepCopy()
		s.SetName(fmt.Sprintf("slice-%02d", i))
		many = append(many, s)
	}
	var devices []model.DetailSection
	for _, s := range p.DetailSections(nodeNamed(t, "worker-a-01"), fakeRelated{"resourceslices": many}) {
		if s.Group == "Devices" {
			devices = append(devices, s)
		}
	}
	if len(devices) != 33 || devices[32].Title != "More" || findItemIn(devices[32], "Not shown") != "8 more resource slices" {
		t.Fatalf("%d sections, the last %+v", len(devices), devices[len(devices)-1])
	}
}

func findItemIn(sec model.DetailSection, name string) string {
	for _, it := range sec.Items {
		if it.Name == name {
			return it.Value
		}
	}
	return ""
}

func findLink(sec model.DetailSection, name string) *model.ObjectRef {
	for _, it := range sec.Items {
		if it.Name == name {
			return it.Link
		}
	}
	return nil
}
