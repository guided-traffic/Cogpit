package kinds_test

import (
	"slices"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

// TestPolicyReportResults reads both report shapes: Kyverno 1.10+'s, one object as the scope, and
// the older one, each result naming its objects; the violations only when asked.
func TestPolicyReportResults(t *testing.T) {
	scoped := recorded(t, kinds.PolicyReportsKey, "deployment-fail")
	all := kinds.PolicyReportResults(scoped.Object, nil)
	if len(all) != 3 {
		t.Fatalf("%d results", len(all))
	}
	fails := kinds.PolicyReportResults(scoped.Object, kinds.IsPolicyViolation)
	if len(fails) != 1 {
		t.Fatalf("violations %+v", fails)
	}
	f := fails[0]
	if f.Policy != "require-team-label" || f.Rule != "autogen-check-team" || f.Process != "background scan" ||
		f.Severity != "medium" || f.Category != "Best Practices" || !f.Time.Equal(time.Unix(1791720000, 0)) {
		t.Errorf("violation %+v", f)
	}
	if len(f.Subjects) != 1 || f.Subjects[0].UID != "6b1f7a52-1c1e-4d0b-9a55-0c2f8d1e0001" || f.Subjects[0].Group() != "apps" {
		t.Errorf("subjects %+v", f.Subjects)
	}
	if !kinds.KyvernoReport(scoped.Object) {
		t.Error("Kyverno's report not recognised")
	}

	wide := recorded(t, kinds.PolicyReportsKey, "namespace-wide")
	vs := kinds.PolicyReportResults(wide.Object, kinds.IsPolicyViolation)
	if len(vs) != 1 || vs[0].Result != kinds.PolicyWarn || len(vs[0].Subjects) != 2 || vs[0].Subjects[1].UID != "" {
		t.Errorf("an older report's violations %+v", vs)
	}
	if !kinds.KyvernoReport(wide.Object) {
		t.Error("an unlabelled report of Kyverno's source not recognised")
	}
	if kinds.IsPolicyViolation(kinds.PolicyError) || kinds.IsPolicyViolation(kinds.PolicyPass) || kinds.IsPolicyViolation(kinds.PolicySkip) {
		t.Error("an error, pass or skip counted as a violation")
	}
	if got := kinds.PolicyReportResults(recorded(t, kinds.PolicyReportsKey, "minimal").Object, nil); len(got) != 0 {
		t.Errorf("an empty report's results %+v", got)
	}
}

// TestPolicyRef links a result's policy by its source: a ClusterPolicy, a namespaced Policy, a
// ValidatingPolicy; nothing for a source Cogpit has not seen.
func TestPolicyRef(t *testing.T) {
	for _, tt := range []struct {
		r    kinds.PolicyResult
		want *model.ObjectRef
	}{
		{kinds.PolicyResult{Policy: "require-labels", Source: "kyverno"}, &model.ObjectRef{Kind: "clusterpolicies.kyverno.io", Name: "require-labels"}},
		{kinds.PolicyResult{Policy: "shop/no-latest", Source: "kyverno"}, &model.ObjectRef{Kind: "policies.kyverno.io", Namespace: "shop", Name: "no-latest"}},
		{kinds.PolicyResult{Policy: "service-team", Source: "KyvernoValidatingPolicy"}, &model.ObjectRef{Kind: "validatingpolicies.policies.kyverno.io", Name: "service-team"}},
		{kinds.PolicyResult{Policy: "vap", Source: "ValidatingAdmissionPolicy"}, nil},
		{kinds.PolicyResult{Source: "kyverno"}, nil},
	} {
		got := tt.r.PolicyRef()
		if (got == nil) != (tt.want == nil) || (got != nil && *got != *tt.want) {
			t.Errorf("%+v: %+v, want %+v", tt.r, got, tt.want)
		}
	}
}

// TestPolicyReportColumns pins the report views' columns: Kyverno's printer columns with the object
// as a link.
func TestPolicyReportColumns(t *testing.T) {
	for key, want := range map[string][]string{
		kinds.PolicyReportsKey:        {"name", "namespace", "kind", "object", "pass", "fail", "warn", "error", "skip", "age"},
		kinds.ClusterPolicyReportsKey: {"name", "kind", "object", "pass", "fail", "warn", "error", "skip", "age"},
	} {
		var got []string
		for _, c := range curatedProfile(t, key).Columns {
			got = append(got, c.ID)
		}
		if !slices.Equal(got, want) {
			t.Errorf("%s: %v", key, got)
		}
	}
}
