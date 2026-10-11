package kinds_test

import (
	"testing"

	"github.com/guided-traffic/cogpit/internal/model"
)

func FuzzHorizontalPodAutoscalers(f *testing.F) { fuzzProfile(f, "horizontalpodautoscalers") }

func TestHorizontalPodAutoscalersStatusTexts(t *testing.T) {
	const (
		none    = model.StatusNone
		ok      = model.StatusOK
		pending = model.StatusPending
		warning = model.StatusWarning
		bad     = model.StatusError
	)
	// The columns are Lens's documented ones; the forms of Metrics (kubectl's TARGETS with each
	// metric named) and Status (the True conditions) and the status rule are inferred.
	checkStatusTexts(t, []statusText{
		{"horizontalpodautoscalers", "healthy", "metrics", "cpu: 45%/80%, memory: 301989888/512Mi", none, ok},
		{"horizontalpodautoscalers", "healthy", "minPods", "2", none, ok},
		{"horizontalpodautoscalers", "healthy", "maxPods", "10", none, ok},
		{"horizontalpodautoscalers", "healthy", "replicas", "4", none, ok},
		{"horizontalpodautoscalers", "healthy", "status", "AbleToScale ScalingActive", ok, ok},
		{"horizontalpodautoscalers", "failing", "metrics", "cpu: <unknown>/70%", none, bad},
		{"horizontalpodautoscalers", "failing", "status", "AbleToScale", bad, bad},
		{"horizontalpodautoscalers", "at-maximum", "metrics", "cpu (app): 96%/75%, packets-per-second: 1450/1k + 2 more...", none, warning},
		{"horizontalpodautoscalers", "at-maximum", "status", "AbleToScale ScalingActive ScalingLimited", warning, warning},
		// Held at the minimum is the normal idle state, not a warning.
		{"horizontalpodautoscalers", "terminating", "status", "AbleToScale ScalingActive ScalingLimited", ok, model.StatusTerminating},
		{"horizontalpodautoscalers", "scaling-disabled", "status", "AbleToScale", none, none},
		{"horizontalpodautoscalers", "scaling-disabled", "replicas", "0", none, none},
		{"horizontalpodautoscalers", "minimal", "minPods", "1", none, pending},
		{"horizontalpodautoscalers", "minimal", "metrics", "<none>", none, pending},
		{"horizontalpodautoscalers", "minimal", "status", "", pending, pending},
	})
}
