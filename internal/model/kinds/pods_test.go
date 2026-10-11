package kinds_test

import (
	"testing"

	"github.com/guided-traffic/cogpit/internal/model"
)

// TestPodWarningTexts pins the warning column of recorded pods. Lens's rule is not captured (an
// orange triangle on a pod with many restarts and a container not ready); this is Cogpit's.
func TestPodWarningTexts(t *testing.T) {
	checkStatusTexts(t, []statusText{
		{"pods", "healthy", "warning", "", model.StatusNone, model.StatusOK},
		{"pods", "failing", "warning", "CrashLoopBackOff; Containers not ready: worker; 7 restarts", model.StatusWarning, model.StatusError},
		{"pods", "not-ready", "warning", "Containers not ready: auth-proxy; 2 restarts", model.StatusWarning, model.StatusWarning},
		{"pods", "unschedulable", "warning", "Unschedulable", model.StatusWarning, model.StatusPending},
		{"pods", "evicted", "warning", "Evicted", model.StatusWarning, model.StatusError},
		{"pods", "completed-job", "warning", "", model.StatusNone, model.StatusSucceeded},
		{"pods", "terminating", "warning", "", model.StatusNone, model.StatusTerminating},
		{"pods", "container-creating", "warning", "", model.StatusNone, model.StatusPending},
		{"pods", "init-progress", "warning", "", model.StatusNone, model.StatusPending},
	})
}
