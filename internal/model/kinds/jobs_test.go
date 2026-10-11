package kinds_test

import (
	"testing"

	"github.com/guided-traffic/cogpit/internal/model"
)

func FuzzJobs(f *testing.F) { fuzzProfile(f, "jobs") }

func TestJobsStatusTexts(t *testing.T) {
	const none = model.StatusNone
	checkStatusTexts(t, []statusText{
		{"jobs", "healthy", "conditions", "Complete SuccessCriteriaMet", model.StatusSucceeded, model.StatusSucceeded},
		{"jobs", "healthy", "completions", "1/1", none, model.StatusSucceeded},
		{"jobs", "failing", "conditions", "Failed FailureTarget", model.StatusError, model.StatusError},
		{"jobs", "failing", "completions", "0/1", none, model.StatusError},
		{"jobs", "suspended", "conditions", "Suspended", none, none},
		{"jobs", "suspended", "completions", "0/3", none, none},
		{"jobs", "running", "completions", "3/5", none, model.StatusPending},
		{"jobs", "running", "conditions", "", none, model.StatusPending},
		{"jobs", "work-queue", "completions", "0/1 of 2", none, model.StatusPending},
		{"jobs", "retrying", "completions", "0/1", none, model.StatusWarning},
		{"jobs", "minimal", "completions", "0/1", none, model.StatusPending},
		{"jobs", "terminating", "conditions", "Complete", model.StatusSucceeded, model.StatusTerminating},
	})
}
