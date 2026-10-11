package kinds_test

import (
	"testing"

	"github.com/guided-traffic/cogpit/internal/model"
)

func FuzzStatefulSets(f *testing.F) { fuzzProfile(f, "statefulsets") }

func TestStatefulSetsStatusTexts(t *testing.T) {
	const none = model.StatusNone
	checkStatusTexts(t, []statusText{
		{"statefulsets", "healthy", "pods", "3/3", none, model.StatusOK},
		{"statefulsets", "healthy", "replicas", "3", none, model.StatusOK},
		{"statefulsets", "failing", "pods", "1/3", none, model.StatusWarning},
		{"statefulsets", "rolling", "pods", "2/3", none, model.StatusPending},
		{"statefulsets", "partitioned", "pods", "3/3", none, model.StatusOK}, // the partition holds the rest back on purpose
		{"statefulsets", "scaled-to-zero", "pods", "0/0", none, none},
		{"statefulsets", "minimal", "replicas", "1", none, model.StatusPending},
		{"statefulsets", "terminating", "pods", "1/1", none, model.StatusTerminating},
	})
}
