package cluster

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/guided-traffic/cogpit/internal/testkit"
)

// whoAmIConn connects to a fake API server whose user is id; its kubeconfig's user entry is "fake".
func whoAmIConn(t *testing.T, id *testkit.FakeIdentity) (*Conn, *testkit.FakeAPIServer) {
	t.Helper()
	srv := testkit.NewFakeAPIServer(testkit.FakeAPIOptions{Token: secretToken, Identity: id})
	t.Cleanup(srv.Close)
	c, err := Connect(context.Background(), Options{KubeconfigPaths: []string{srv.WriteKubeconfig(t, "fake", "")}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.Close)
	return c, srv
}

// The API server says who the connection is logged in as: one SelfSubjectReview on
// authentication.k8s.io/v1, the username and the groups kept on the connection, the UID and the
// extra fields nowhere.
func TestReviewUserAsksTheAPIServer(t *testing.T) {
	c, srv := whoAmIConn(t, &testkit.FakeIdentity{
		Username: "alice@example.com", Groups: []string{"team-a", "system:authenticated", "team-a"},
		UID: "uid-must-not-be-kept", Extra: map[string][]string{"authentication.kubernetes.io/credential-id": {"JTI=must-not-be-kept"}},
	})
	if _, ok := c.User(); ok {
		t.Fatal("a user before the review")
	}
	before := c.Counter().Snapshot()
	u := c.ReviewUser(context.Background())
	if !u.Confirmed || u.Username != "alice@example.com" || !slices.Equal(u.Groups, []string{"team-a", "system:authenticated"}) || u.Reason != "" {
		t.Errorf("ReviewUser = %+v", u)
	}
	if kept, ok := c.User(); !ok || kept.Username != u.Username || !slices.Equal(kept.Groups, u.Groups) {
		t.Errorf("User = %+v, %v", kept, ok)
	}
	after := c.Counter().Snapshot()
	if n := after.ByVerb[VerbCreate] - before.ByVerb[VerbCreate]; n != 1 || after.Total-before.Total != 1 {
		t.Errorf("the review cost %d creates of %d requests, want 1 of 1", n, after.Total-before.Total)
	}
	if srv.Requests()["create selfsubjectreviews.authentication.k8s.io"] != 1 {
		t.Errorf("server saw %v", srv.Requests())
	}
	// The kept answer is a copy: changing what was returned changes nothing.
	u.Groups[0] = "changed"
	if kept, _ := c.User(); kept.Groups[0] != "team-a" {
		t.Errorf("the kept groups changed with the returned ones: %v", kept.Groups)
	}
}

// A Kubernetes 1.27 API server serves the review as v1beta1 only: asked there after v1's 404.
func TestReviewUserFallsBackToV1beta1(t *testing.T) {
	c, srv := whoAmIConn(t, &testkit.FakeIdentity{Username: "bob", Groups: []string{"devs"}, BetaOnly: true})
	u := c.ReviewUser(context.Background())
	if !u.Confirmed || u.Username != "bob" || !slices.Equal(u.Groups, []string{"devs"}) {
		t.Errorf("ReviewUser = %+v", u)
	}
	if n := srv.Requests()["create selfsubjectreviews.authentication.k8s.io"]; n != 2 {
		t.Errorf("%d reviews, want 2 (v1, then v1beta1)", n)
	}
}

// An API server that does not know the review (404, older than 1.27) or refuses it (403): the
// kubeconfig's user entry's name, not confirmed, with the reason; the connect is not the review's.
func TestReviewUserFallsBackToTheKubeconfigUser(t *testing.T) {
	for _, tc := range []struct {
		status  int
		reviews int64
		reason  string
	}{
		{http.StatusNotFound, 2, "does not serve SelfSubjectReview"},
		{http.StatusForbidden, 1, "refused the SelfSubjectReview (403 Forbidden)"},
		{http.StatusInternalServerError, 1, "the SelfSubjectReview failed"},
	} {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			c, srv := whoAmIConn(t, &testkit.FakeIdentity{Username: "never-shown", Status: tc.status})
			u := c.ReviewUser(context.Background())
			if u.Confirmed || u.Username != "fake" || len(u.Groups) != 0 || !strings.Contains(u.Reason, tc.reason) {
				t.Errorf("ReviewUser = %+v", u)
			}
			if strings.Contains(u.Reason, secretToken) {
				t.Errorf("the reason carries the token: %q", u.Reason)
			}
			if kept, ok := c.User(); !ok || kept.Username != "fake" || kept.Confirmed || len(kept.Groups) != 0 || kept.Reason != u.Reason {
				t.Errorf("User = %+v, %v", kept, ok)
			}
			if n := srv.Requests()["create selfsubjectreviews.authentication.k8s.io"]; n != tc.reviews {
				t.Errorf("%d reviews, want %d", n, tc.reviews)
			}
		})
	}
}

// The answer is renewed with every review: a login renewed as another user shows the new one.
func TestReviewUserRenews(t *testing.T) {
	c, srv := whoAmIConn(t, &testkit.FakeIdentity{Username: "alice", Groups: []string{"a"}})
	c.ReviewUser(context.Background())
	srv.SetIdentity(testkit.FakeIdentity{Username: "carol", Groups: []string{"c1", "c2"}})
	c.ReviewUser(context.Background())
	if u, _ := c.User(); u.Username != "carol" || !slices.Equal(u.Groups, []string{"c1", "c2"}) {
		t.Errorf("User after the second review = %+v", u)
	}
}
