package testkit

import (
	"io"
	"net/http"
	"slices"
)

// FakeIdentity is who a FakeAPIServer says its user is: its answer to a SelfSubjectReview (what
// `kubectl auth whoami` asks), a create on authentication.k8s.io/v1 and v1beta1 that stores
// nothing. A server without one answers as DefaultFakeIdentity.
type FakeIdentity struct {
	Username string
	Groups   []string
	// UID and Extra are answered too, as a real API server does; Cogpit must keep neither.
	UID   string
	Extra map[string][]string
	// Status, when set, answers every review with it instead: http.StatusNotFound as an API server
	// older than 1.27 (or one that turned the API off), http.StatusForbidden as one whose RBAC
	// refuses it.
	Status int
	// BetaOnly serves v1beta1 alone, as a Kubernetes 1.27 API server: v1 answers 404.
	BetaOnly bool
}

// DefaultFakeIdentity is a FakeAPIServer's user unless told otherwise: an administrator as kubeadm
// writes one.
var DefaultFakeIdentity = FakeIdentity{
	Username: "kubernetes-admin",
	Groups:   []string{"kubeadm:cluster-admins", "system:authenticated"},
}

// SetIdentity changes who the server says its user is, for the reviews from now on.
func (s *FakeAPIServer) SetIdentity(id FakeIdentity) {
	s.mu.Lock()
	s.opts.Identity = &id
	s.mu.Unlock()
}

func (s *FakeAPIServer) identity() FakeIdentity {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.opts.Identity == nil {
		return DefaultFakeIdentity
	}
	id := *s.opts.Identity
	id.Groups = slices.Clone(id.Groups)
	return id
}

// serveWhoAmI answers POST /apis/authentication.k8s.io/<version>/selfsubjectreviews.
func (s *FakeAPIServer) serveWhoAmI(w http.ResponseWriter, r *http.Request, version string) {
	s.count("create", "selfsubjectreviews.authentication.k8s.io")
	_, _ = io.Copy(io.Discard, r.Body) // the review asks nothing: its spec is empty
	id := s.identity()
	switch {
	case id.Status == http.StatusForbidden:
		writeStatus(w, http.StatusForbidden, "Forbidden", `selfsubjectreviews.authentication.k8s.io is forbidden: User "`+id.Username+
			`" cannot create resource "selfsubjectreviews" in API group "authentication.k8s.io" at the cluster scope`)
		return
	case id.Status != 0:
		writeStatus(w, id.Status, http.StatusText(id.Status), "the server could not find the requested resource")
		return
	case version != "v1" && version != "v1beta1", version == "v1" && id.BetaOnly:
		writeStatus(w, http.StatusNotFound, "NotFound", "the server could not find the requested resource")
		return
	}
	user := map[string]any{"username": id.Username, "groups": id.Groups}
	if id.UID != "" {
		user["uid"] = id.UID
	}
	if len(id.Extra) > 0 {
		user["extra"] = id.Extra
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"kind": "SelfSubjectReview", "apiVersion": "authentication.k8s.io/" + version,
		"metadata": map[string]any{"creationTimestamp": nil},
		"status":   map[string]any{"userInfo": user},
	})
}
