package engine

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/guided-traffic/cogpit/internal/model"
)

// Access is what the access reviews of the connected user answer: SelfSubjectAccessReview and
// SelfSubjectRulesReview, cached per context and namespace, refreshed on connect and on demand,
// never polled (gap G2/G4). The connector sets it on the Backend; nil allows everything, and the
// API server decides.
//
// The registry asks it in Command.Enabled, for every Verb the command declares, whenever the
// command's parameters name an object ("ref"): such a command is greyed out "not permitted" in
// every menu and refused by Dispatch for every origin. Menus only read Enabled. A command whose own
// Enabled says ErrNotApplicable stays hidden instead; for every other one the denial comes first.
type Access interface {
	// Allowed reports whether the user may use v in namespace: "" asks for cluster-scoped
	// resources, and for namespaced ones in every namespace. It must not block or call the API
	// server: an answer it does not hold yet returns known false (and may start a review in the
	// background); Cogpit then offers the command and lets the API server decide. v.Resource is
	// never "*": the registry resolves it from the object's kind first.
	Allowed(namespace string, v Verb) (allowed, known bool)
}

// ErrNotPermitted marks a command the user's access reviews do not permit. Enabled wraps it; the
// text says which verb on which resource.
var ErrNotPermitted = errors.New("not permitted")

// permitted is the registry's generic access check (Command.Enabled): every declared verb, for
// the object the parameters name. Commands without a "ref" parameter, kinds Cogpit makes itself
// (local kinds), a verb on "*" for a kind that is no API object (a virtual kind: Edit and Delete
// do not apply to it), and answers the reviews do not have pass.
func (s *Session) permitted(c *Command, p any) error {
	if len(c.Verbs) == 0 {
		return nil
	}
	b := s.Backend()
	if b == nil || b.Access == nil {
		return nil
	}
	ref, ok := paramsRef(p)
	if !ok {
		return nil
	}
	prof, hasProf := b.Profile(ref.Kind)
	if hasProf && prof.Local {
		return nil
	}
	for _, v := range c.Verbs {
		ns := ref.Namespace
		if v.Resource == "*" {
			if !hasProf || !prof.APIObject() {
				// A kind without a profile: discovery would be needed, so the server decides. A
				// virtual kind: "*" names no API resource, and its own commands declare their verbs.
				continue
			}
			v.Group, v.Resource = prof.Group, prof.Resource
			if !prof.Namespaced {
				ns = ""
			}
		}
		if allowed, known := b.Access.Allowed(ns, v); known && !allowed {
			return fmt.Errorf("%w: you may not %s", ErrNotPermitted, describeVerb(v, ns))
		}
	}
	return nil
}

// describeVerb says "patch deployments.apps in team-a", "create pods/eviction".
func describeVerb(v Verb, namespace string) string {
	res := v.Resource
	if v.Subresource != "" {
		res += "/" + v.Subresource
	}
	if v.Group != "" {
		res += "." + v.Group
	}
	out := v.Verb + " " + res
	if namespace != "" {
		out += " in " + namespace
	}
	return out
}

// paramsRef returns the object a command's parameters name in their "ref" field.
func paramsRef(p any) (model.ObjectRef, bool) {
	raw, err := json.Marshal(p)
	if err != nil {
		return model.ObjectRef{}, false
	}
	var withRef struct {
		Ref *model.ObjectRef `json:"ref"`
	}
	if json.Unmarshal(raw, &withRef) != nil || withRef.Ref == nil || withRef.Ref.Kind == "" {
		return model.ObjectRef{}, false
	}
	return *withRef.Ref, true
}
