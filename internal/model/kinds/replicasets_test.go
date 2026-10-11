package kinds_test

import (
	"testing"

	"github.com/guided-traffic/cogpit/internal/model"
)

func FuzzReplicaSets(f *testing.F) { fuzzProfile(f, "replicasets") }

func TestReplicaSetsStatusTexts(t *testing.T) {
	const none = model.StatusNone
	checkStatusTexts(t, []statusText{
		{"replicasets", "healthy", "desired", "3", none, model.StatusOK},
		{"replicasets", "healthy", "ready", "3", none, model.StatusOK},
		{"replicasets", "failing", "current", "1", none, model.StatusError}, // ReplicaFailure: quota exceeded
		{"replicasets", "not-ready", "ready", "0", none, model.StatusWarning},
		{"replicasets", "scaling", "current", "3", none, model.StatusPending},
		{"replicasets", "scaled-to-zero", "desired", "0", none, none},
		{"replicasets", "minimal", "desired", "1", none, model.StatusPending},
		{"replicasets", "terminating", "ready", "1", none, model.StatusTerminating},
	})
}
