package commands

import (
	"testing"

	"github.com/guided-traffic/cogpit/internal/cluster"
)

// TestDrainLeft: what a drain that did not finish says it left of the node, from its last step
// (TestFailedDrainSaysWhatItLeft runs the first two through the command).
func TestDrainLeft(t *testing.T) {
	for _, tc := range []struct {
		cordoned bool
		last     cluster.DrainProgress
		want     string
	}{
		{false, cluster.DrainProgress{}, "is unchanged: the cordon did not go through"},
		{true, cluster.DrainProgress{Step: cluster.DrainCordoned}, "stays cordoned; no pod was evicted"},
		{true, cluster.DrainProgress{Step: cluster.DrainBlocked, Done: 0, Total: 3}, "stays cordoned; no pod was evicted"},
		{true, cluster.DrainProgress{Step: cluster.DrainEvicted, Done: 1, Total: 3}, "stays cordoned and partly drained (1 of 3 pods evicted)"},
	} {
		if got := drainLeft(tc.cordoned, tc.last); got != tc.want {
			t.Errorf("drainLeft(%v, %+v) = %q, want %q", tc.cordoned, tc.last, got, tc.want)
		}
	}
}
