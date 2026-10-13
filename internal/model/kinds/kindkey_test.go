package kinds_test

import (
	"testing"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

// TestReferencesLinkToEveryBuiltinKind: a reference to an object (an owner reference, an event's
// involved object, a Helm release's resource) carries an API version and a Kind, which
// model.BuiltinKindKey turns into the kind key its link opens. Every built-in view's kind resolves
// to its own key and scope, at its version and at each older one it falls back to, so that a
// reference to a Pod Security Policy, an admission policy or a Device Resources object links as
// one to a Pod does.
func TestReferencesLinkToEveryBuiltinKind(t *testing.T) {
	for _, p := range kinds.Builtin() {
		if !p.APIObject() {
			continue
		}
		for _, v := range append([]string{p.Version}, p.FallbackVersions...) {
			apiVersion := v
			if p.Group != "" {
				apiVersion = p.Group + "/" + v
			}
			key, namespaced, ok := model.BuiltinKindKey(apiVersion, p.Kind)
			if !ok || key != p.Key || namespaced != p.Namespaced {
				t.Errorf("BuiltinKindKey(%q, %q) = %q namespaced %v ok %v, want %q namespaced %v",
					apiVersion, p.Kind, key, namespaced, ok, p.Key, p.Namespaced)
			}
		}
		if r := model.RefTo(p.Group+"/"+p.Version, p.Kind, "ns", "x"); r == nil || r.Kind != p.Key {
			t.Errorf("RefTo(%s) = %+v, want a link to %s", p.Kind, r, p.Key)
		}
	}
}
