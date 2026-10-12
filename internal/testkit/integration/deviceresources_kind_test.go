//go:build kind

package integration_test

import (
	"context"
	"os"
	"testing"
	"time"

	resourcev1 "k8s.io/api/resource/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/model"
)

// Names of what TestKindDeviceResources creates on the shared kind cluster: its namespace, and the
// cluster-scoped device class and resource slice of a driver that does not exist, so that nothing
// ever allocates or prepares its devices.
const (
	draNamespace = "cogpit-dra"
	draDriver    = "cogpit-dra.example.com"
	draClass     = "cogpit-dra-gpu"
	draSlice     = "cogpit-dra-slice"
)

// TestKindDeviceResources fills the four Device Resources views from watches on the kind cluster
// (Kubernetes 1.36 serves resource.k8s.io/v1): a device class that selects a fake driver's
// devices, a resource slice that publishes two of them to every node, a claim template and a claim
// in its own namespace. No pod uses the claim, so it stays pending and the slice's devices stay
// unallocated. Everything it creates is deleted at the end.
func TestKindDeviceResources(t *testing.T) {
	kubeconfig, contextName := os.Getenv("COGPIT_KIND_KUBECONFIG"), os.Getenv("COGPIT_KIND_CONTEXT")
	if kubeconfig == "" || contextName == "" {
		t.Skip("COGPIT_KIND_KUBECONFIG and COGPIT_KIND_CONTEXT are not set; run make test-kind")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	conn, err := cluster.Connect(ctx, cluster.Options{KubeconfigPaths: []string{kubeconfig}, Context: contextName})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	kube, ns := conn.Kube(), draNamespace
	dra := kube.ResourceV1()
	freshNamespace(ctx, t, kube, ns)

	cleanup := func() {
		dctx, dcancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer dcancel()
		if err := dra.DeviceClasses().Delete(dctx, draClass, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			t.Errorf("delete device class: %v", err)
		}
		if err := dra.ResourceSlices().Delete(dctx, draSlice, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			t.Errorf("delete resource slice: %v", err)
		}
	}
	cleanup() // a leftover of an interrupted run
	t.Cleanup(cleanup)

	selector := []resourcev1.DeviceSelector{{CEL: &resourcev1.CELDeviceSelector{Expression: `device.driver == "` + draDriver + `"`}}}
	if _, err := dra.DeviceClasses().Create(ctx, &resourcev1.DeviceClass{
		ObjectMeta: metav1.ObjectMeta{Name: draClass},
		Spec:       resourcev1.DeviceClassSpec{Selectors: selector},
	}, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	allNodes := true
	if _, err := dra.ResourceSlices().Create(ctx, &resourcev1.ResourceSlice{
		ObjectMeta: metav1.ObjectMeta{Name: draSlice},
		Spec: resourcev1.ResourceSliceSpec{
			Driver:   draDriver,
			Pool:     resourcev1.ResourcePool{Name: "cogpit-pool", Generation: 1, ResourceSliceCount: 1},
			AllNodes: &allNodes,
			Devices:  []resourcev1.Device{{Name: "gpu-0"}, {Name: "gpu-1"}},
		},
	}, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	request := []resourcev1.DeviceRequest{{Name: "gpu", Exactly: &resourcev1.ExactDeviceRequest{DeviceClassName: draClass}}}
	if _, err := dra.ResourceClaimTemplates(ns).Create(ctx, &resourcev1.ResourceClaimTemplate{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: "one-gpu"},
		Spec:       resourcev1.ResourceClaimTemplateSpec{Spec: resourcev1.ResourceClaimSpec{Devices: resourcev1.DeviceClaim{Requests: request}}},
	}, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := dra.ResourceClaims(ns).Create(ctx, &resourcev1.ResourceClaim{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: "gpu"},
		Spec:       resourcev1.ResourceClaimSpec{Devices: resourcev1.DeviceClaim{Requests: request}},
	}, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}

	s := engine.New(engine.Options{Connector: &engine.ClusterConnector{KubeconfigPaths: []string{kubeconfig}}})
	defer s.Close()
	if err := commands.RegisterAll(s.Registry()); err != nil {
		t.Fatal(err)
	}
	if err := s.Connect(ctx, contextName); err != nil {
		t.Fatal(err)
	}
	d := dispatcher{t: t, ctx: ctx, s: s}
	for _, key := range []string{"resourceclaims", "resourceclaimtemplates", "deviceclasses", "resourceslices"} {
		if p, ok := s.Backend().Profile(key); !ok || p.Version != "v1" {
			t.Errorf("%s: offered %v at %+v; want resource.k8s.io/v1 on Kubernetes 1.36", key, ok, p)
		}
	}

	row := func(view string, ref model.ObjectRef, what string, cond func(cell func(id string) model.Cell, r model.Row) bool) {
		t.Helper()
		d.must("ui.openView", map[string]any{"view": view}, nil)
		waitFor(ctx, t, view+": "+what, func() bool {
			tbl := s.Table(view)
			if tbl == nil || !tbl.Synced || tbl.Error != "" {
				return false
			}
			i := tbl.IndexOf(ref)
			if i < 0 {
				return false
			}
			got := tbl.Rows[i]
			cell := func(id string) model.Cell {
				for j, c := range tbl.Columns {
					if c.ID == id && j < len(got.Cells) {
						return got.Cells[j]
					}
				}
				return model.Cell{}
			}
			return cond(cell, got)
		})
	}
	row("deviceclasses", model.ObjectRef{Kind: "deviceclasses", Name: draClass}, "the class's driver",
		func(cell func(string) model.Cell, _ model.Row) bool { return cell("driver").Text == draDriver })
	row("resourceslices", model.ObjectRef{Kind: "resourceslices", Name: draSlice}, "the slice's devices, none allocated",
		func(cell func(string) model.Cell, _ model.Row) bool {
			return cell("driver").Text == draDriver && cell("pool").Text == "cogpit-pool" && cell("node").Text == "All nodes" &&
				cell("devices").Num == 2 && cell("allocated").Text == "" && cell("allocated").Num == 0
		})
	row("resourceclaimtemplates", model.ObjectRef{Kind: "resourceclaimtemplates", Namespace: ns, Name: "one-gpu"}, "the template's class",
		func(cell func(string) model.Cell, _ model.Row) bool { return cell("deviceClasses").Text == draClass })
	row("resourceclaims", model.ObjectRef{Kind: "resourceclaims", Namespace: ns, Name: "gpu"}, "a pending claim",
		func(cell func(string) model.Cell, r model.Row) bool {
			return cell("state").Text == "Pending" && cell("deviceClasses").Text == draClass && r.Status == model.StatusPending
		})
	claim, err := s.DetailWait(ctx, model.ObjectRef{Kind: "resourceclaims", Namespace: ns, Name: "gpu"})
	if err != nil {
		t.Fatal(err)
	}
	if !hasTableRow(claim, "Requests", []string{"gpu", draClass, "1"}) {
		t.Errorf("claim drawer: %+v", claim.Sections)
	}
}
