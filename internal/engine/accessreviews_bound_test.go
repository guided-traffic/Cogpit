package engine

import (
	"fmt"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// TestAccessReviewsAreBounded: the namespaces and questions the access reviews keep answers for
// are bounded. Every command an agent dispatches asks Allowed in the namespace its parameters
// name, existing or not, and each new namespace used to cost a rules review, an entry kept until
// a refresh and a goroutine waiting for a review slot: an agent naming thousands of namespaces
// made thousands of reviews. Past the bound the answer stays unknown (the API server decides),
// and the namespaces reviewed first keep their answers.
func TestAccessReviewsAreBounded(t *testing.T) {
	rev := newFakeReviewer(map[string][]cluster.AccessRule{"": {rule("delete", "", "pods")}})
	s := newAccessSession(t, testkit.NewMemorySource(), rev, "default")
	a := s.Backend().Access.(*accessReviews)
	if !waitAllowed(t, a, "default", deletePods) {
		t.Fatal("delete pods in default denied")
	}
	for i := range 3000 {
		a.Allowed(fmt.Sprintf("bogus-%d", i), deletePods)
		a.Allowed(fmt.Sprintf("bogus-%d", i), Verb{Verb: "*", Resource: "pods"}) // an access review each
	}
	time.Sleep(200 * time.Millisecond) // let the reviews that were started run
	rules, access := rev.calls()
	a.mu.Lock()
	nRules, nChecks := len(a.rules), len(a.checks)
	a.mu.Unlock()
	if nRules > accessRulesLimit || len(rules) > accessRulesLimit {
		t.Errorf("%d namespaces kept, %d rules reviews made; at most %d", nRules, len(rules), accessRulesLimit)
	}
	if nChecks > accessChecksLimit || len(access) > accessChecksLimit {
		t.Errorf("%d questions kept, %d access reviews made; at most %d", nChecks, len(access), accessChecksLimit)
	}
	if allowed, known := a.Allowed("default", deletePods); !allowed || !known {
		t.Errorf("default lost its answer: allowed %v, known %v", allowed, known)
	}
	if _, known := a.Allowed("bogus-2999", deletePods); known {
		t.Error("a namespace past the bound has an answer")
	}
}
