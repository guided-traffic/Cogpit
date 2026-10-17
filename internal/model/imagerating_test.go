package model

import (
	"encoding/json"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// trivyReport is a VulnerabilityReport shaped as Trivy Operator 0.35.0 writes it (kind,
// 2026-10-17): synthetic names, the vulnerabilities left out.
func trivyReport(kind, workload, container string, summary map[string]any) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "aquasecurity.github.io/v1alpha1", "kind": "VulnerabilityReport",
		"metadata": map[string]any{
			"name": "replicaset-web-5d9c-app", "namespace": "shop", "creationTimestamp": "2026-10-17T09:00:00Z",
			"labels": map[string]any{
				"app.kubernetes.io/managed-by": "trivy-operator", "resource-spec-hash": "85f6f8bfb6",
				TrivyLabelContainerName: container, TrivyLabelResourceKind: kind,
				TrivyLabelResourceName: workload, TrivyLabelResourceNamespace: "shop",
			},
			"ownerReferences": []any{map[string]any{
				"apiVersion": "apps/v1", "kind": kind, "name": workload, "uid": "u-1", "controller": true,
			}},
		},
		"report": map[string]any{
			"artifact": map[string]any{"repository": "library/nginx", "tag": "1.16.1", "digest": "sha256:d20a"},
			"registry": map[string]any{"server": "index.docker.io"},
			"scanner":  map[string]any{"name": "Trivy", "vendor": "Aqua Security", "version": "0.75.0"},
			"summary":  summary, "updateTimestamp": "2026-10-17T09:55:22Z",
			"vulnerabilities": []any{},
		},
	}}
}

func TestImageRatings(t *testing.T) {
	cases := []struct {
		c     VulnerabilityCounts
		want  ImageRating
		text  string
		st    Status
		chips []string
	}{
		{VulnerabilityCounts{Critical: 39, High: 123, Medium: 147, Low: 23, Unknown: 31}, RatingCritical, "Critical · 39", StatusError,
			[]string{"39 Critical", "123 High", "147 Medium", "23 Low", "31 Unknown"}},
		{VulnerabilityCounts{High: 10, Medium: 16, Low: 1, Unknown: 152}, RatingHigh, "High · 10", StatusWarning,
			[]string{"10 High", "16 Medium", "1 Low", "152 Unknown"}},
		{VulnerabilityCounts{Medium: 2}, RatingMedium, "Medium · 2", StatusPending, []string{"2 Medium"}},
		{VulnerabilityCounts{Low: 1}, RatingLow, "Low · 1", StatusSucceeded, []string{"1 Low"}},
		{VulnerabilityCounts{Unknown: 4}, RatingUnknown, "Unknown · 4", StatusUnknown, []string{"4 Unknown"}},
		{VulnerabilityCounts{}, RatingClean, "Clean", StatusOK, nil},
	}
	for _, c := range cases {
		r := c.c.Rating()
		if r != c.want || RatingText(r, c.c) != c.text || r.Status() != c.st {
			t.Errorf("%+v: %v %q %q", c.c, r, RatingText(r, c.c), r.Status())
		}
		var chips []string
		for _, ch := range c.c.Chips() {
			chips = append(chips, ch.Text)
			if ch.Status == StatusNone {
				t.Errorf("chip %q without a colour", ch.Text)
			}
		}
		if len(chips) != len(c.chips) {
			t.Errorf("%+v chips %v", c.c, chips)
		}
		for i := range chips {
			if chips[i] != c.chips[i] {
				t.Errorf("%+v chips %v", c.c, chips)
			}
		}
	}
	if RatingText(RatingNotScanned, VulnerabilityCounts{}) != "Not scanned" || RatingNotScanned.Status() != StatusNone {
		t.Error("Not scanned")
	}
	// The order: a worse rating is greater.
	order := []ImageRating{RatingNotScanned, RatingClean, RatingUnknown, RatingLow, RatingMedium, RatingHigh, RatingCritical}
	for i := 1; i < len(order); i++ {
		if order[i] <= order[i-1] {
			t.Errorf("%v is not worse than %v", order[i], order[i-1])
		}
	}
	if ImageRating(42).String() != "Not scanned" || ImageRating(-1).Status() != StatusNone {
		t.Error("an out-of-range rating")
	}
	b, _ := json.Marshal(map[string]ImageRating{"r": RatingHigh})
	var back map[string]ImageRating
	if string(b) != `{"r":"high"}` || json.Unmarshal(b, &back) != nil || back["r"] != RatingHigh {
		t.Errorf("JSON %s, %v", b, back)
	}
}

func TestParseVulnerabilityReport(t *testing.T) {
	obj := trivyReport("ReplicaSet", "web-5d9c", "app", map[string]any{
		"criticalCount": int64(1), "highCount": int64(2), "mediumCount": float64(3), "lowCount": int64(-4), "noneCount": int64(0),
	})
	r, ok := ParseVulnerabilityReport(obj)
	want := VulnerabilityCounts{Critical: 1, High: 2, Medium: 3}
	if !ok || r.Kind != "ReplicaSet" || r.Workload != "web-5d9c" || r.Container != "app" || r.Namespace != "shop" ||
		r.Name != "replicaset-web-5d9c-app" || r.Counts != want || r.Registry != "index.docker.io" ||
		r.Image() != "library/nginx:1.16.1" || !r.Scanned.Equal(time.Date(2026, 10, 17, 9, 55, 22, 0, time.UTC)) {
		t.Fatalf("%+v, %v", r, ok)
	}

	// A workload name too long for a label: the annotation of the same key.
	long := trivyReport("StatefulSet", "", "db", nil)
	labels := long.GetLabels()
	delete(labels, TrivyLabelResourceName)
	labels[TrivyLabelResourceNameHash] = "6b8f"
	long.SetLabels(labels)
	long.SetAnnotations(map[string]string{TrivyLabelResourceName: "a-statefulset-whose-name-is-much-longer-than-sixty-three-characters-x"})
	if r, ok := ParseVulnerabilityReport(long); !ok || r.Workload != "a-statefulset-whose-name-is-much-longer-than-sixty-three-characters-x" ||
		r.Counts != (VulnerabilityCounts{}) || r.Counts.Rating() != RatingClean {
		t.Errorf("long name: %+v, %v", r, ok)
	}
	// Neither: the report's controller of that kind.
	noName := trivyReport("StatefulSet", "", "db", nil)
	unstructured.RemoveNestedField(noName.Object, "metadata", "labels", TrivyLabelResourceName)
	noName.SetOwnerReferences(nil)
	if _, ok := ParseVulnerabilityReport(noName); ok {
		t.Error("a report that names no workload was read")
	}
	owned := trivyReport("StatefulSet", "db", "db", nil)
	unstructured.RemoveNestedField(owned.Object, "metadata", "labels", TrivyLabelResourceName)
	if r, ok := ParseVulnerabilityReport(owned); !ok || r.Workload != "db" {
		t.Errorf("by its controller: %+v, %v", r, ok)
	}
	// No update time: the creation time.
	unstructured.RemoveNestedField(owned.Object, "report", "updateTimestamp")
	if r, _ := ParseVulnerabilityReport(owned); !r.Scanned.Equal(time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)) {
		t.Errorf("scanned %v", r.Scanned)
	}
	// Hostile shapes never panic.
	for _, o := range []map[string]any{
		{}, {"report": "x"}, {"report": map[string]any{"summary": []any{1}}},
		{"metadata": map[string]any{"labels": map[string]any{TrivyLabelResourceKind: 1}}},
	} {
		ParseVulnerabilityReport(&unstructured.Unstructured{Object: o})
	}
	if _, ok := ParseVulnerabilityReport(nil); ok {
		t.Error("nil")
	}
}

func TestImageMatches(t *testing.T) {
	for _, c := range []struct {
		image, repo, tag string
	}{
		{"nginx:1.16.1", "nginx", "1.16.1"},
		{"docker.io/library/nginx:1.16.1", "nginx", "1.16.1"},
		{"nginx", "nginx", "latest"},
		{"nginx@sha256:abc", "nginx", ""},
		{"registry.example:5000/team/app:2.0@sha256:abc", "team/app", "2.0"},
		{"localhost/app", "app", "latest"},
		{"ghcr.io/org/app", "org/app", "latest"},
		{"bitnami/redis:7", "bitnami/redis", "7"},
	} {
		if repo, tag := ImageRepository(c.image); repo != c.repo || tag != c.tag {
			t.Errorf("%s: %q %q", c.image, repo, tag)
		}
	}
	r := VulnerabilityReport{Repository: "library/nginx", Tag: "1.16.1"}
	for image, want := range map[string]bool{
		"nginx:1.16.1": true, "docker.io/library/nginx:1.16.1": true, "nginx@sha256:abc": true,
		"nginx:1.17": false, "nginx": false, "httpd:1.16.1": false, "": true,
	} {
		if r.Matches(image) != want {
			t.Errorf("%q matches: %v", image, !want)
		}
	}
	if !(VulnerabilityReport{Repository: "org/app", Digest: "sha256:abc"}).Matches("ghcr.io/org/app@sha256:abc") {
		t.Error("by digest")
	}
}
