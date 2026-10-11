package kinds_test

import (
	"testing"

	"github.com/guided-traffic/cogpit/internal/model"
)

func FuzzReplicationControllers(f *testing.F) { fuzzProfile(f, "replicationcontrollers") }

func TestReplicationControllerTexts(t *testing.T) {
	checkStatusTexts(t, []statusText{
		{"replicationcontrollers", "healthy", "replicas", "3", model.StatusNone, model.StatusOK},
		{"replicationcontrollers", "healthy", "desiredReplicas", "3", model.StatusNone, model.StatusOK},
		{"replicationcontrollers", "healthy", "selector", "app=frontend, version=v1", model.StatusNone, model.StatusOK},
		// A ReplicaFailure condition (quota) is an error, as for replica sets.
		{"replicationcontrollers", "failing", "replicas", "2", model.StatusNone, model.StatusError},
		{"replicationcontrollers", "not-ready", "replicas", "3", model.StatusNone, model.StatusWarning},
		{"replicationcontrollers", "scaled-to-zero", "desiredReplicas", "0", model.StatusNone, model.StatusNone},
		// The API defaults spec.replicas to 1: still to be created.
		{"replicationcontrollers", "minimal", "desiredReplicas", "1", model.StatusNone, model.StatusPending},
		{"replicationcontrollers", "minimal", "selector", "", model.StatusNone, model.StatusPending},
		{"replicationcontrollers", "terminating", "replicas", "1", model.StatusNone, model.StatusTerminating},
	})
	// Documented columns (not captured): no Age.
	assertColumnOrder(t, "replicationcontrollers", []model.Column{
		{ID: "name", Title: "Name"},
		{ID: "namespace", Title: "Namespace"},
		{ID: "replicas", Title: "Replicas"},
		{ID: "desiredReplicas", Title: "Desired Replicas"},
		{ID: "selector", Title: "Selector"},
	})
}
