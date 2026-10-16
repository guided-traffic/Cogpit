package commands

import (
	"context"
	"errors"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// AccessResult is cluster.refreshAccess's result.
type AccessResult struct {
	Context string `json:"context"`
	// Namespaces are those whose rules are reviewed at once: the context's default namespace and
	// the namespace filter's. Others follow as menus, the drawer and the sidebar ask.
	Namespaces []string `json:"namespaces"`
}

// The access reviews' API verb and group.
const (
	reviewVerb  = "create"
	reviewGroup = "authorization.k8s.io"
)

// reviewVerbs are the verbs of the access reviews. Creating a review writes nothing: the API
// server answers it and keeps nothing.
var reviewVerbs = []engine.Verb{
	{Verb: reviewVerb, Group: reviewGroup, Resource: "selfsubjectrulesreviews"},
	{Verb: reviewVerb, Group: reviewGroup, Resource: "selfsubjectaccessreviews"},
}

func registerAccess(g *registrar) {
	add(g, engine.Spec[engine.OnCluster]{
		ID: "cluster.refreshAccess", Label: "Refresh Access Reviews", MenuPath: "Cluster/Refresh Access Reviews",
		Danger: engine.DangerRead, Verbs: reviewVerbs,
		Description: "Asks the API server of a connected cluster (context, else the shown one) again what the user may do (SelfSubjectRulesReview per namespace, " +
			"SelfSubjectAccessReview where rules cannot answer) and drops the answers Cogpit kept since connecting: use it " +
			"after the user's roles changed. Until the new answers arrive, commands are offered and kinds shown; then menus " +
			"grey what the user may not do (\"not permitted: ...\") and the sidebar hides kinds they may not list. Returns " +
			"{context, namespaces}: the namespaces reviewed at once. Writes nothing.",
		Enabled: func(s *engine.Session, p engine.OnCluster) error { _, err := reviewable(s, p.Context); return err },
		Run: func(_ context.Context, s *engine.Session, p engine.OnCluster) (any, error) {
			b, err := reviewable(s, p.Context)
			if err != nil {
				return nil, err
			}
			namespaces, ok := b.RefreshAccess()
			if !ok {
				return nil, errNoAccessReviews
			}
			return AccessResult{Context: b.Context, Namespaces: namespaces}, nil
		},
	})
}

// errNoAccessReviews disables cluster.refreshAccess on a backend without access reviews.
var errNoAccessReviews = errors.New("this cluster has no access reviews to refresh (the demo cluster allows everything)")

// reviewable is the Enabled check of cluster.refreshAccess: a connected cluster with access reviews.
func reviewable(s *engine.Session, contextID string) (*engine.Backend, error) {
	b, err := connectedOn(s, contextID)
	if err != nil {
		return nil, err
	}
	if !b.HasAccessReviews() {
		return nil, errNoAccessReviews
	}
	return b, nil
}
