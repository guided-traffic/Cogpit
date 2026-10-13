package engine

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// pspGVR is where clusters before Kubernetes 1.25 serve Pod Security Policies.
var pspGVR = schema.GroupVersionResource{Group: "policy", Version: "v1beta1", Resource: "podsecuritypolicies"}

// offersPSP reports whether profiles hold the Pod Security Policies view, and at which version.
func offersPSP(profiles []*model.KindProfile) (string, bool) {
	for _, p := range profiles {
		if p.Key == "podsecuritypolicies" {
			return p.Version, true
		}
	}
	return "", false
}

// TestPodSecurityPoliciesOnlyWhereServed checks the Pod Security Policies view on the discovery
// of two clusters, as Lens's documentation has it ("available only with clusters prior to
// Kubernetes 1.25"): a 1.24 cluster serves policy/v1beta1 podsecuritypolicies next to
// policy/v1 poddisruptionbudgets and gets the view; a 1.25 or later cluster serves policy/v1 only
// and gets none. First on discovery results, then through client-go's discovery of a fake API
// server, whose policies then fill the view's cache. kind runs 1.36, which serves none, so no
// cluster test covers it.
func TestPodSecurityPoliciesOnlyWhereServed(t *testing.T) {
	pdb := schema.GroupVersionResource{Group: "policy", Version: "v1", Resource: "poddisruptionbudgets"}
	if v, ok := offersPSP(servedProfiles(kinds.Builtin(), map[schema.GroupVersionResource]bool{pspGVR: true, pdb: true})); !ok || v != "v1beta1" {
		t.Errorf("1.24: offered %v at %q", ok, v)
	}
	if _, ok := offersPSP(servedProfiles(kinds.Builtin(), map[schema.GroupVersionResource]bool{pdb: true})); ok {
		t.Error("1.25: Pod Security Policies offered on a cluster that does not serve them")
	}

	for _, tt := range []struct {
		name   string
		served bool
	}{{"serves policy/v1beta1", true}, {"does not serve them", false}} {
		t.Run(tt.name, func(t *testing.T) {
			resources := append([]testkit.FakeResource(nil), testkit.DefaultFakeResources...)
			if tt.served {
				resources = append(resources, testkit.FakeResource{GVR: pspGVR, Kind: "PodSecurityPolicy"})
			}
			srv := testkit.NewFakeAPIServer(testkit.FakeAPIOptions{Resources: resources})
			defer srv.Close()
			if tt.served {
				srv.Set(pspGVR, testkit.Object("policy/v1beta1", "PodSecurityPolicy", "", "restricted",
					map[string]any{"spec": map[string]any{"privileged": false, "volumes": []any{"secret", "configMap"}}}))
			}
			conn := &ClusterConnector{KubeconfigPaths: []string{srv.WriteKubeconfig(t, "fake", "")}}
			b, err := conn.Connect(context.Background(), "fake")
			if err != nil {
				t.Fatal(err)
			}
			defer b.close(slog.Default())
			_, ok := offersPSP(b.Profiles)
			if ok != tt.served {
				t.Fatalf("offered %v, want %v", ok, tt.served)
			}
			if !tt.served {
				return
			}
			p, _ := kinds.ByKey("podsecuritypolicies")
			h := b.Acquire(p)
			defer h.Release()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if !h.WaitForSync(ctx) {
				t.Fatalf("the view's cache did not sync: %v", h.Err())
			}
			objs := KindList(h, p)
			if len(objs) != 1 {
				t.Fatalf("%d policies", len(objs))
			}
			row := model.BuildRow(p, "fake", objs[0])
			if got := []string{row.Cells[0].Text, row.Cells[1].Text, row.Cells[2].Text}; got[0] != "restricted" || got[1] != "false" || got[2] != "secret, configMap" {
				t.Errorf("row %q", got)
			}
		})
	}
}
