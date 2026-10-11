package kinds_test

import (
	"testing"

	"github.com/guided-traffic/cogpit/internal/model"
)

func FuzzLeases(f *testing.F) { fuzzProfile(f, "leases") }

func TestLeasesStatusTexts(t *testing.T) {
	const none = model.StatusNone
	// Holder is spec.holderIdentity; a lease is not coloured (inferred).
	checkStatusTexts(t, []statusText{
		{"leases", "healthy", "holder", "control-plane-1_9c1d7f2e-4b6a-4f3e-8a21-6d0b5c9e1f34", none, none},
		{"leases", "failing", "holder", "", none, none},
		{"leases", "coordinated", "holder", "control-plane-2_1f2e3d4c-5b6a-4987-8a1b-2c3d4e5f6a7b", none, none},
		{"leases", "minimal", "holder", "", none, none},
		{"leases", "terminating", "holder", "worker-b-03", none, model.StatusTerminating},
	})
}
