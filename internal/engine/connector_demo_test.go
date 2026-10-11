package engine

import (
	"slices"
	"testing"

	"github.com/guided-traffic/cogpit/internal/demo"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

// TestDemoProfilesFollowTheDemo checks that the demo cluster offers exactly the built-in kinds it
// serves, as discovery decides for a real cluster: a profile the demo has no objects for (Ingresses,
// Config Maps) must not become a view that only shows a 404.
func TestDemoProfilesFollowTheDemo(t *testing.T) {
	served := demo.Resources()
	offered := demoProfiles()
	if len(offered) == 0 {
		t.Fatal("the demo offers no kinds")
	}
	for _, p := range kinds.Builtin() {
		if want, got := slices.Contains(served, p.GVR()), slices.Contains(offered, p); got != want {
			t.Errorf("%s: offered %v, served by the demo %v", p.Key, got, want)
		}
	}
}
