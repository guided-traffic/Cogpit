package kinds_test

import (
	"testing"

	"github.com/guided-traffic/cogpit/internal/model"
)

func FuzzRuntimeClasses(f *testing.F) { fuzzProfile(f, "runtimeclasses") }

func TestRuntimeClassesStatusTexts(t *testing.T) {
	const none = model.StatusNone
	// A runtime class has no state: no row is coloured (inferred).
	checkStatusTexts(t, []statusText{
		{"runtimeclasses", "healthy", "handler", "runsc", none, none},
		{"runtimeclasses", "failing", "handler", "kata-qemu-not-installed", none, none},
		{"runtimeclasses", "minimal", "handler", "", none, none},
		{"runtimeclasses", "terminating", "handler", "runc", none, model.StatusTerminating},
	})
}
