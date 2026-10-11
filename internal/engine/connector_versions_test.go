package engine

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/guided-traffic/cogpit/internal/model"
)

// TestServedProfilesFallBackToAnOlderVersion checks how discovery picks a profile's version: the
// preferred one when the cluster serves it (the profile itself), else the first served fallback
// version (a copy at that version, the shared profile untouched), else no view at all.
func TestServedProfilesFallBackToAnOlderVersion(t *testing.T) {
	policies := &model.KindProfile{
		Key: "validatingadmissionpolicies", Group: "admissionregistration.k8s.io", Version: "v1",
		FallbackVersions: []string{"v1beta1", "v1alpha1"}, Resource: "validatingadmissionpolicies",
	}
	mutating := &model.KindProfile{
		Key: "mutatingadmissionpolicies", Group: "admissionregistration.k8s.io", Version: "v1",
		Resource: "mutatingadmissionpolicies",
	}
	gvr := func(resource, version string) schema.GroupVersionResource {
		return schema.GroupVersionResource{Group: "admissionregistration.k8s.io", Version: version, Resource: resource}
	}
	tests := []struct {
		name   string
		served []schema.GroupVersionResource
		want   map[string]string // kind key → version offered
	}{
		{"every version served", []schema.GroupVersionResource{
			gvr("validatingadmissionpolicies", "v1"), gvr("validatingadmissionpolicies", "v1beta1"),
			gvr("mutatingadmissionpolicies", "v1"), gvr("mutatingadmissionpolicies", "v1beta1"),
		}, map[string]string{policies.Key: "v1", mutating.Key: "v1"}},
		{"only beta served", []schema.GroupVersionResource{
			gvr("validatingadmissionpolicies", "v1beta1"), gvr("validatingadmissionpolicies", "v1alpha1"),
			gvr("mutatingadmissionpolicies", "v1beta1"),
		}, map[string]string{policies.Key: "v1beta1"}},
		{"only alpha served", []schema.GroupVersionResource{
			gvr("validatingadmissionpolicies", "v1alpha1"),
		}, map[string]string{policies.Key: "v1alpha1"}},
		{"none served", nil, map[string]string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			served := map[schema.GroupVersionResource]bool{}
			for _, g := range tt.served {
				served[g] = true
			}
			got := servedProfiles([]*model.KindProfile{policies, mutating}, served)
			if len(got) != len(tt.want) {
				t.Fatalf("offered %d kinds, want %d: %+v", len(got), len(tt.want), got)
			}
			for _, p := range got {
				if want, ok := tt.want[p.Key]; !ok || p.Version != want {
					t.Errorf("%s offered at %q, want %q (offered: %v)", p.Key, p.Version, want, ok)
				}
				if !served[p.GVR()] {
					t.Errorf("%s offered at %v, which the cluster does not serve", p.Key, p.GVR())
				}
				if p.Version == "v1" && p != policies && p != mutating {
					t.Errorf("%s: the preferred version should keep the shared profile", p.Key)
				}
			}
		})
	}
	if policies.Version != "v1" {
		t.Errorf("the shared profile changed to %q", policies.Version)
	}
}
