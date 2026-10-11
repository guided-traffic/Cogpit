package cluster

import (
	"context"
	"testing"

	authorizationv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"
)

func TestAccessRulesMatch(t *testing.T) {
	rules := AccessRules{Rules: []AccessRule{
		{Verbs: []string{"get", "list", "watch"}, APIGroups: []string{""}, Resources: []string{"pods", "pods/log"}},
		{Verbs: []string{"*"}, APIGroups: []string{"apps"}, Resources: []string{"deployments"}},
		{Verbs: []string{"create"}, APIGroups: []string{""}, Resources: []string{"*/eviction"}},
		{Verbs: []string{"delete"}, APIGroups: []string{""}, Resources: []string{"pods"}, ResourceNames: []string{"web"}},
		{Verbs: []string{"update"}, APIGroups: []string{"*"}, Resources: []string{"*"}, ResourceNames: []string{"*"}},
	}}
	for _, tt := range []struct {
		a    AccessAttributes
		want RuleMatch
	}{
		{AccessAttributes{Verb: "list", Resource: "pods"}, RuleAllows},
		{AccessAttributes{Verb: "get", Resource: "pods", Subresource: "log"}, RuleAllows},
		{AccessAttributes{Verb: "get", Resource: "pods", Subresource: "exec"}, RuleNone},
		{AccessAttributes{Verb: "list", Group: "apps", Resource: "pods"}, RuleNone},
		{AccessAttributes{Verb: "patch", Group: "apps", Resource: "deployments"}, RuleAllows},
		{AccessAttributes{Verb: "patch", Group: "apps", Resource: "deployments", Subresource: "scale"}, RuleNone},
		{AccessAttributes{Verb: "create", Resource: "pods", Subresource: "eviction"}, RuleAllows},
		{AccessAttributes{Verb: "create", Resource: "pods"}, RuleNone},
		{AccessAttributes{Verb: "delete", Resource: "pods"}, RuleNamesOnly},
		{AccessAttributes{Verb: "update", Group: "batch", Resource: "jobs"}, RuleAllows},
		{AccessAttributes{Verb: "update", Group: "batch", Resource: "jobs", Subresource: "status"}, RuleAllows},
		{AccessAttributes{Verb: "delete", Resource: "nodes"}, RuleNone},
	} {
		if got := rules.Match(tt.a); got != tt.want {
			t.Errorf("%s: %d, want %d", tt.a, got, tt.want)
		}
	}
	if got := (AccessRules{}).Match(AccessAttributes{Verb: "list", Resource: "pods"}); got != RuleNone {
		t.Errorf("no rules: %d", got)
	}
}

func TestAccessAttributesString(t *testing.T) {
	for a, want := range map[AccessAttributes]string{
		{Verb: "list", Resource: "pods", Namespace: "team-a"}:                                "list pods in team-a",
		{Verb: "patch", Group: "apps", Resource: "deployments", Subresource: "scale"}:        "patch deployments.apps/scale",
		{Verb: "create", Resource: "selfsubjectrulesreviews", Group: "authorization.k8s.io"}: "create selfsubjectrulesreviews.authorization.k8s.io",
	} {
		if got := a.String(); got != want {
			t.Errorf("%q, want %q", got, want)
		}
	}
}

// The reviews go through the connection's clientset as creates of the review resources, with the
// namespace and attributes asked; evaluation errors make the answer uncertain.
func TestReviewRulesAndAccess(t *testing.T) {
	c, cs := fakeConn()
	cs.PrependReactor("create", "selfsubjectrulesreviews", func(a k8stesting.Action) (bool, runtime.Object, error) {
		review := a.(k8stesting.CreateAction).GetObject().(*authorizationv1.SelfSubjectRulesReview)
		review.Status = authorizationv1.SubjectRulesReviewStatus{
			ResourceRules: []authorizationv1.ResourceRule{{Verbs: []string{"list"}, APIGroups: []string{""}, Resources: []string{"pods"}}},
		}
		if review.Spec.Namespace == "broken" {
			review.Status.EvaluationError = "role.rbac.authorization.k8s.io \"gone\" not found"
		}
		return true, review, nil
	})
	cs.PrependReactor("create", "selfsubjectaccessreviews", func(a k8stesting.Action) (bool, runtime.Object, error) {
		review := a.(k8stesting.CreateAction).GetObject().(*authorizationv1.SelfSubjectAccessReview)
		ra := review.Spec.ResourceAttributes
		review.Status.Allowed = ra.Namespace == "team-a" && ra.Verb == "delete" && ra.Resource == "pods"
		if ra.Namespace == "webhook" {
			review.Status.EvaluationError = "webhook unreachable"
		}
		return true, review, nil
	})
	ctx := context.Background()

	rules, err := c.ReviewRules(ctx, "team-a")
	if err != nil || rules.Incomplete || rules.Match(AccessAttributes{Verb: "list", Resource: "pods"}) != RuleAllows {
		t.Errorf("ReviewRules = %+v, %v", rules, err)
	}
	if rules, err := c.ReviewRules(ctx, "broken"); err != nil || !rules.Incomplete {
		t.Errorf("an evaluation error leaves the rules complete: %+v, %v", rules, err)
	}
	for _, tt := range []struct {
		a              AccessAttributes
		allowed, known bool
	}{
		{AccessAttributes{Namespace: "team-a", Verb: "delete", Resource: "pods"}, true, true},
		{AccessAttributes{Namespace: "team-b", Verb: "delete", Resource: "pods"}, false, true},
		{AccessAttributes{Namespace: "webhook", Verb: "delete", Resource: "pods"}, false, false},
	} {
		allowed, known, err := c.ReviewAccess(ctx, tt.a)
		if err != nil || allowed != tt.allowed || known != tt.known {
			t.Errorf("%s: %v %v %v, want %v %v", tt.a, allowed, known, err, tt.allowed, tt.known)
		}
	}
	var creates int
	for _, a := range cs.Actions() {
		if a.GetVerb() != "create" || a.GetResource().Group != "authorization.k8s.io" {
			t.Errorf("unexpected request %s %s", a.GetVerb(), a.GetResource())
		}
		creates++
	}
	if creates != 5 {
		t.Errorf("%d reviews sent, want 5", creates)
	}
}
