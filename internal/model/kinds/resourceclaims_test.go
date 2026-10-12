package kinds_test

import (
	"strings"
	"testing"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

func FuzzResourceClaims(f *testing.F)         { fuzzProfile(f, "resourceclaims") }
func FuzzResourceClaimTemplates(f *testing.F) { fuzzProfile(f, "resourceclaimtemplates") }
func FuzzDeviceClasses(f *testing.F)          { fuzzProfile(f, "deviceclasses") }
func FuzzResourceSlices(f *testing.F)         { fuzzProfile(f, "resourceslices") }

// TestDeviceResourcesColumns pins the columns of Lens's documentation (read 2026-10-13, not
// captured) for the four Device Resources views; Resource Claims' Devices and Reserved For are
// Cogpit's own hidden columns.
func TestDeviceResourcesColumns(t *testing.T) {
	assertColumnOrder(t, "resourceclaims", []model.Column{
		{ID: "name", Title: "Name"}, {ID: "warning", Title: "Warning"}, {ID: "namespace", Title: "Namespace"},
		{ID: "state", Title: "State"}, {ID: "deviceClasses", Title: "Device Classes"},
		{ID: "devices", Title: "Devices"}, {ID: "reservedFor", Title: "Reserved For"}, {ID: "age", Title: "Age"},
	})
	assertColumnOrder(t, "resourceclaimtemplates", []model.Column{
		{ID: "name", Title: "Name"}, {ID: "namespace", Title: "Namespace"},
		{ID: "deviceClasses", Title: "Device Classes"}, {ID: "age", Title: "Age"},
	})
	assertColumnOrder(t, "deviceclasses", []model.Column{
		{ID: "name", Title: "Name"}, {ID: "driver", Title: "Driver"}, {ID: "age", Title: "Age"},
	})
	assertColumnOrder(t, "resourceslices", []model.Column{
		{ID: "name", Title: "Name"}, {ID: "driver", Title: "Driver"}, {ID: "pool", Title: "Pool"},
		{ID: "node", Title: "Node"}, {ID: "devices", Title: "Devices"}, {ID: "allocated", Title: "Allocated"},
		{ID: "age", Title: "Age"},
	})
	p, _ := kinds.ByKey("resourceclaims")
	for _, id := range []string{"devices", "reservedFor"} {
		if !p.Columns[p.ColumnIndex(id)].Hidden {
			t.Errorf("Resource Claims' %s column is shown; Lens's documentation lists no such column", id)
		}
	}
	for _, key := range []string{"resourceclaims", "resourceclaimtemplates", "deviceclasses", "resourceslices"} {
		p, _ := kinds.ByKey(key)
		if p.Group != "resource.k8s.io" || p.Version != "v1" || len(p.FallbackVersions) != 0 {
			t.Errorf("%s: %s/%s %v: Lens reads resource.k8s.io/v1 only", key, p.Group, p.Version, p.FallbackVersions)
		}
	}
}

func TestResourceClaimsStatusTexts(t *testing.T) {
	const (
		none    = model.StatusNone
		ok      = model.StatusOK
		pending = model.StatusPending
		warning = model.StatusWarning
		term    = model.StatusTerminating
	)
	checkStatusTexts(t, []statusText{
		// State as Lens's documentation words it: Pending, Allocated, Allocated (reserved: N).
		{"resourceclaims", "healthy", "state", "Allocated (reserved: 1)", ok, ok},
		{"resourceclaims", "shared", "state", "Allocated (reserved: 2)", ok, ok},
		{"resourceclaims", "pending", "state", "Pending", pending, pending},
		{"resourceclaims", "minimal", "state", "Pending", pending, pending},
		{"resourceclaims", "terminating", "state", "Allocated", ok, term},
		// The warning icon: a driver reports an unhealthy device (which conditions count is inferred).
		{"resourceclaims", "failing", "warning", "Device gpu-3 (gpu.example.com) reports Ready=False: Xid 79: GPU has fallen off the bus", warning, warning},
		{"resourceclaims", "failing", "state", "Allocated (reserved: 1)", ok, warning},
		{"resourceclaims", "healthy", "warning", "", none, ok},
		{"resourceclaims", "healthy", "deviceClasses", "gpu.example.com", none, ok},
		{"resourceclaims", "pending", "deviceClasses", "gpu-large.example.com, gpu.example.com, nic.example.net", none, pending},
		{"resourceclaims", "shared", "devices", "gpu-0, gpu-1, gpu-2 + 2 more", none, ok},
		{"resourceclaims", "shared", "reservedFor", "worker-0, worker-1", none, ok},
		{"resourceclaims", "healthy", "reservedFor", "trainer-0", none, ok},

		{"resourceclaimtemplates", "healthy", "deviceClasses", "gpu.example.com", none, none},
		{"resourceclaimtemplates", "failing", "deviceClasses", "gpu-large.example.com, gpu.example.com, nic.example.net", none, none},
		{"resourceclaimtemplates", "minimal", "deviceClasses", "", none, none},
		{"resourceclaimtemplates", "terminating", "deviceClasses", "nic.example.net", none, term},

		// Driver from the selectors' comparisons of device.driver, else the configuration (inferred).
		{"deviceclasses", "healthy", "driver", "gpu.example.com", none, none},
		{"deviceclasses", "failing", "driver", "nic.example.net", none, none},
		{"deviceclasses", "terminating", "driver", "fpga.example.org", none, term},
		{"deviceclasses", "two-drivers", "driver", "gpu.example.com, gpu.example.org", none, none},
		{"deviceclasses", "minimal", "driver", "", none, none},

		{"resourceslices", "healthy", "node", "worker-a-01", none, none},
		{"resourceslices", "healthy", "devices", "2", none, none},
		{"resourceslices", "healthy", "allocated", "-", none, none}, // the claims are not listed here
		{"resourceslices", "all-nodes", "node", "All nodes", none, none},
		{"resourceslices", "node-selector", "node", "Node selector", none, none},
		{"resourceslices", "minimal", "node", "", none, none},
		{"resourceslices", "terminating", "pool", "worker-b-01", none, term},
	})
}

// TestResourceSliceAllocated: Allocated counts the slice's devices that a cached claim holds,
// through ClaimsByPool; a device of another pool or driver, or one the slice does not publish,
// does not count; the drawer links each held device to its claim.
func TestResourceSliceAllocated(t *testing.T) {
	ix := kinds.ClaimsByPool
	if ix.Kind != "resourceclaims" {
		t.Fatalf("ClaimsByPool indexes %q", ix.Kind)
	}
	healthy := recorded(t, "resourceclaims", "healthy") // gpu-0 of gpu.example.com/worker-a-01
	shared := recorded(t, "resourceclaims", "shared")   // gpu-0..3 and a NIC of worker-a-01
	if got := ix.Keys(shared); strings.Join(got, ",") != "gpu.example.com/worker-a-01,nic.example.net/worker-a-01" {
		t.Errorf("keys of a claim with two drivers: %q", got)
	}
	if got := ix.Keys(recorded(t, "resourceclaims", "pending")); len(got) != 0 {
		t.Errorf("keys of a pending claim: %q", got)
	}

	p, _ := kinds.ByKey("resourceslices")
	if len(p.Needs) != 1 || p.Needs[0].ID() != ix.ID() {
		t.Fatalf("Needs = %+v", p.Needs)
	}
	slice := recorded(t, "resourceslices", "healthy") // gpu-0 and gpu-1 of worker-a-01
	cases := []struct {
		name string
		rel  model.Related
		want string
	}{
		{"not listed", model.NoRelated, "-"},
		{"no claims", fakeRelated{"resourceclaims": nil}, "0"},
		{"one claim", fakeRelated{"resourceclaims": {healthy}}, "1"},
		// gpu-0 is held by both (a recorded overlap); gpu-2 and gpu-3 are not in the slice.
		{"two claims", fakeRelated{"resourceclaims": {healthy, shared}}, "2"},
	}
	i := p.ColumnIndex("allocated")
	for _, c := range cases {
		row := model.BuildRowWith(p, "test", slice, c.rel)
		if got := model.CellText(p.Columns[i], row.Cells[i], now); got != c.want {
			t.Errorf("%s: Allocated %q, want %q", c.name, got, c.want)
		}
	}

	sections := p.DetailSections(slice, fakeRelated{"resourceclaims": {shared, healthy}})
	var devices *model.DetailTable
	for _, s := range sections {
		if s.Title == "Devices" {
			devices = s.Table
		}
	}
	if devices == nil || len(devices.Rows) != 2 {
		t.Fatalf("Devices section %+v", sections)
	}
	// gpu-0 goes to the first claim by namespace and name, both in ml: shared-accel before trainer-gpu.
	if l := devices.Link(0, 4); l == nil || l.Kind != "resourceclaims" || l.Namespace != "ml" || l.Name != "shared-accel" {
		t.Errorf("gpu-0's claim link %+v", l)
	}
	if devices.Rows[0][1] != "driverVersion: 1.2.3\nindex: 0\nmig: false\nmodel: A100" || devices.Rows[0][2] != "memory: 40Gi" {
		t.Errorf("gpu-0's attributes and capacity %q", devices.Rows[0])
	}
}

// TestPodResourceClaimsSection: the pod drawer lists each of the pod's claims, the claim it
// resolves to (a generated one from status.resourceClaimStatuses), its template, and the
// containers that request it; a pod without claims has no such section.
func TestPodResourceClaimsSection(t *testing.T) {
	p, _ := kinds.ByKey("pods")
	find := func(name string) *model.DetailTable {
		for _, s := range p.DetailSections(recorded(t, "pods", name), nil) {
			if s.Title == "Resource Claims" {
				return s.Table
			}
		}
		return nil
	}
	if find("healthy") != nil {
		t.Error("a pod without claims shows a Resource Claims section")
	}
	tbl := find("resource-claims")
	if tbl == nil || len(tbl.Rows) != 3 {
		t.Fatalf("Resource Claims table %+v", tbl)
	}
	want := [][]string{
		{"gpu", "trainer-0-gpu-x2v9k", "single-gpu", "warmup\ntrainer"},
		{"shared", "shared-accel", "", "trainer (nic)"},
		{"scratch", "(not generated yet)", "scratch-nvme", ""},
	}
	for i, row := range want {
		if strings.Join(tbl.Rows[i], "|") != strings.Join(row, "|") {
			t.Errorf("row %d = %q, want %q", i, tbl.Rows[i], row)
		}
	}
	if l := tbl.Link(0, 1); l == nil || l.Kind != "resourceclaims" || l.Namespace != "ml" || l.Name != "trainer-0-gpu-x2v9k" {
		t.Errorf("generated claim link %+v", l)
	}
	if l := tbl.Link(0, 2); l == nil || l.Kind != "resourceclaimtemplates" || l.Name != "single-gpu" {
		t.Errorf("template link %+v", l)
	}
	if l := tbl.Link(2, 1); l != nil {
		t.Errorf("a claim not generated yet links to %+v", l)
	}
}

// TestResourceClaimLinks: a single consumer pod and the allocation's node are links.
func TestResourceClaimLinks(t *testing.T) {
	p, _ := kinds.ByKey("resourceclaims")
	row := model.BuildRow(p, "test", recorded(t, "resourceclaims", "healthy"))
	if l := row.Cells[p.ColumnIndex("reservedFor")].Link; l == nil || l.Kind != "pods" || l.Namespace != "ml" || l.Name != "trainer-0" {
		t.Errorf("Reserved For link %+v", l)
	}
	shared := model.BuildRow(p, "test", recorded(t, "resourceclaims", "shared"))
	if l := shared.Cells[p.ColumnIndex("reservedFor")].Link; l != nil {
		t.Errorf("two consumers are text, got link %+v", l)
	}
	var node *model.ObjectRef
	for _, s := range p.DetailSections(recorded(t, "resourceclaims", "healthy"), nil) {
		for _, it := range s.Items {
			if it.Name == "Node" {
				node = it.Link
			}
		}
	}
	if node == nil || node.Kind != "nodes" || node.Name != "worker-a-01" {
		t.Errorf("the drawer's Node link %+v", node)
	}
}
