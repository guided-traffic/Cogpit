package kinds_test

import (
	"testing"

	"github.com/guided-traffic/cogpit/internal/model"
)

// TestEventInvolvedObjectText: the involved object reads "Kind: name" and links to the object.
func TestEventInvolvedObjectText(t *testing.T) {
	checkStatusTexts(t, []statusText{
		{"events", "failing", "involvedObject", "Pod: worker-6f8b9c7d4-q9m2z", model.StatusNone, model.StatusWarning},
	})
}
