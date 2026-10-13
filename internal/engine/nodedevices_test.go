package engine

import (
	"slices"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

var resourceSlicesGVR = schema.GroupVersionResource{Group: "resource.k8s.io", Version: "v1", Resource: "resourceslices"}

// sliceOn is a resource slice of driver publishing devices for node.
func sliceOn(name, node, driver string, devices ...string) *unstructured.Unstructured {
	devs := make([]any, 0, len(devices))
	for _, d := range devices {
		devs = append(devs, map[string]any{"name": d})
	}
	return testkit.Object("resource.k8s.io/v1", "ResourceSlice", "", name, map[string]any{"spec": map[string]any{
		"driver": driver, "nodeName": node, "devices": devs,
		"pool": map[string]any{"name": node, "generation": int64(1), "resourceSliceCount": int64(1)},
	}})
}

// nodeDevices returns the drivers of a drawer model's Devices sections.
func nodeDeviceDrivers(m *DetailModel) []string {
	var out []string
	for _, sec := range m.Sections {
		if sec.Group == "Devices" {
			out = append(out, sec.Title)
		}
	}
	return out
}

// TestNodeDrawerDevicesFollowSlices: on a cluster that serves resource.k8s.io/v1, a node's drawer
// lists the resource slices that name the node from the cache and follows them as they come and
// go, a slice of another node never; opening a node's drawer over the Nodes view, which holds the
// slices already, costs no list or watch (once the drawers' events are warm).
func TestNodeDrawerDevicesFollowSlices(t *testing.T) {
	src := testkit.NewMemorySource()
	src.Set(testkit.NodesGVR, testkit.Object("v1", "Node", "", "node-1", nil))
	src.Set(testkit.NodesGVR, testkit.Object("v1", "Node", "", "node-2", nil))
	src.Set(resourceSlicesGVR, sliceOn("node-1-gpu", "node-1", "gpu.example.com", "gpu-0", "gpu-1"))
	src.Set(resourceSlicesGVR, sliceOn("node-2-gpu", "node-2", "gpu.example.com", "gpu-0"))
	s, fc := newTestSession(t, src, 5*time.Millisecond)
	nodes := builtinProfile(t, "nodes")
	fc.profiles = append(testProfiles(), nodes, builtinProfile(t, "resourceslices"))
	connectTest(t, s)
	s.UpdateUI(func(u *UIState) { u.View, u.Sort = nodes.Key, s.ViewSort(nodes.Key) })
	waitTable(t, s, nodes.Key, func(t *TableSnapshot) bool { return t.Synced && len(t.Rows) == 2 })
	waitFor(t, "the view's handle on the slices", func() bool { return handlesOn(s, "resourceslices") == 1 })
	waitFor(t, "the informers to settle", func() bool { return testkit.Settled(s.Backend().Cache, src) })

	// The first drawer starts the events' informer (the drawer's own); the second costs nothing.
	drivers := func(ref model.ObjectRef) []string {
		m, err := s.Detail(ref)
		if err != nil {
			t.Fatal(err)
		}
		return nodeDeviceDrivers(m)
	}
	other := model.ObjectRef{Context: "test", Kind: nodes.Key, Name: "node-2"}
	s.UpdateUI(func(u *UIState) { u.Drawer = &other })
	waitFor(t, "node-2's devices", func() bool { return slices.Equal(drivers(other), []string{"gpu.example.com"}) })
	waitFor(t, "the informers to settle", func() bool { return testkit.Settled(s.Backend().Cache, src) })
	lists, watches := src.Lists(), src.Watches()
	ref := model.ObjectRef{Context: "test", Kind: nodes.Key, Name: "node-1"}
	s.UpdateUI(func(u *UIState) { u.Drawer = &ref })
	waitFor(t, "the node's devices", func() bool { return slices.Equal(drivers(ref), []string{"gpu.example.com"}) })
	m, _ := s.Detail(ref)
	for _, sec := range m.Sections {
		if sec.Group == "Devices" && (sec.Table == nil || len(sec.Table.Rows) != 2) {
			t.Errorf("node-1's GPU slice: %+v", sec)
		}
	}
	if dl, dw := src.Lists()-lists, src.Watches()-watches; dl != 0 || dw != 0 || src.OpenWatches(resourceSlicesGVR) != 1 {
		t.Errorf("the drawer cost %d lists and %d watches; %d watches on the slices", dl, dw, src.OpenWatches(resourceSlicesGVR))
	}

	src.Set(resourceSlicesGVR, sliceOn("node-1-nic", "node-1", "nic.example.net", "vf-0"))
	waitFor(t, "a new slice", func() bool { return slices.Equal(drivers(ref), []string{"gpu.example.com", "nic.example.net"}) })
	src.Delete(resourceSlicesGVR, "", "node-1-gpu")
	waitFor(t, "a slice gone", func() bool { return slices.Equal(drivers(ref), []string{"nic.example.net"}) })
	src.Set(resourceSlicesGVR, sliceOn("node-2-nic", "node-2", "nic.example.net", "vf-1"))
	time.Sleep(50 * time.Millisecond)
	if got := drivers(ref); !slices.Equal(got, []string{"nic.example.net"}) {
		t.Errorf("another node's slice reached node-1: %q", got)
	}
}

// TestNodeDrawerWithoutDeviceResources: on a cluster that does not serve resource.k8s.io/v1 the
// node's drawer has no Devices and nothing asks for resource slices.
func TestNodeDrawerWithoutDeviceResources(t *testing.T) {
	src := testkit.NewMemorySource()
	src.Set(testkit.NodesGVR, testkit.Object("v1", "Node", "", "node-1", nil))
	src.Set(resourceSlicesGVR, sliceOn("node-1-gpu", "node-1", "gpu.example.com", "gpu-0"))
	s, fc := newTestSession(t, src, 5*time.Millisecond)
	nodes := builtinProfile(t, "nodes")
	fc.profiles = append(testProfiles(), nodes) // no resourceslices: the cluster does not serve them
	connectTest(t, s)
	ref := model.ObjectRef{Context: "test", Kind: nodes.Key, Name: "node-1"}
	s.UpdateUI(func(u *UIState) { u.View, u.Sort, u.Drawer = nodes.Key, s.ViewSort(nodes.Key), &ref })
	waitFor(t, "the node's drawer", func() bool {
		m, err := s.Detail(ref)
		return err == nil && len(m.Sections) > 0
	})
	m, _ := s.Detail(ref)
	if d := nodeDeviceDrivers(m); len(d) != 0 {
		t.Errorf("Devices on a cluster without them: %q", d)
	}
	if handlesOn(s, "resourceslices") != 0 || src.Watching(cache.Scope{GVR: resourceSlicesGVR}) {
		t.Error("a handle or watch on resource slices")
	}
}
