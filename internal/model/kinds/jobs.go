package kinds

import (
	"sort"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
)

// Job condition types. Since Kubernetes 1.31 a finished job carries SuccessCriteriaMet or
// FailureTarget first, while its pods terminate, and Complete or Failed once they have.
const (
	jobComplete      = "Complete"
	jobSuccessMet    = "SuccessCriteriaMet"
	jobFailed        = "Failed"
	jobFailureTarget = "FailureTarget"
	jobSuspended     = "Suspended"
)

var jobs = &model.KindProfile{
	Key:        "jobs",
	Group:      "batch",
	Version:    "v1",
	Resource:   "jobs",
	Kind:       "Job",
	Title:      "Jobs",
	Namespaced: true,
	Columns: []model.Column{
		nameColumn,
		namespaceColumn,
		{ID: "completions", Title: "Completions", Type: model.ColumnNumber}, // succeeded/completions, sorted by succeeded
		ageColumn,
		conditionsColumn,
	},
	Cells: jobCells,
	Fields: map[string]fieldFunc{
		idImage: func(obj *unstructured.Unstructured) []string {
			return templateImages(obj)
		},
		idSelector: func(obj *unstructured.Unstructured) []string {
			return labelSelector(obj, "spec", "selector")
		},
		idControlledBy: controlledByField,
		"condition": func(obj *unstructured.Unstructured) []string {
			return jobTrueConditions(obj)
		},
	},
	Detail: jobDetail,
}

func jobCells(obj *unstructured.Unstructured) ([]model.Cell, model.Status) {
	text, cell, row := jobState(obj)
	return []model.Cell{
		nameCell(obj),
		namespaceCell(obj),
		{Text: jobCompletions(obj), Num: nonNegative(obj, "status", "succeeded")},
		model.AgeCell(created(obj)),
		{Text: text, Status: cell},
	}, row
}

// jobCompletions renders the Completions column as kubectl does: "succeeded/completions"; a work
// queue job without completions reads "succeeded/1", or "succeeded/1 of parallelism" when more
// than one pod works on it.
func jobCompletions(obj any) string {
	succeeded := itoa(nonNegative(obj, "status", "succeeded"))
	if n, ok := model.NestedInt64(obj, "spec", "completions"); ok {
		return succeeded + "/" + itoa(max(n, 0))
	}
	if p, ok := model.NestedInt64(obj, "spec", "parallelism"); ok && p > 1 {
		return succeeded + "/1 of " + itoa(p)
	}
	return succeeded + "/1"
}

// jobTrueConditions returns the types of the job's conditions that are True, sorted: the
// Conditions column, as the Deployments view renders it.
func jobTrueConditions(obj any) []string {
	var out []string
	for _, e := range model.NestedSlice(obj, "status", "conditions") {
		if model.NestedString(e, "status") == condTrue {
			out = appendUnique(out, model.NestedString(e, "type"))
		}
	}
	sort.Strings(out)
	return out
}

// jobState returns the Conditions column's text and token and the row's token. A failed job is
// an error, a finished one succeeded (as a completed pod), a suspended one has nothing to colour.
// A job still running is pending, or a warning once some of its pods have failed and it retries.
func jobState(obj any) (text string, cell, row model.Status) {
	trueTypes := jobTrueConditions(obj)
	text = strings.Join(trueTypes, " ")
	suspended, _ := model.NestedBool(obj, "spec", "suspend")
	switch {
	case contains(trueTypes, jobFailed), contains(trueTypes, jobFailureTarget):
		return text, model.StatusError, model.StatusError
	case contains(trueTypes, jobComplete), contains(trueTypes, jobSuccessMet):
		return text, model.StatusSucceeded, model.StatusSucceeded
	case contains(trueTypes, jobSuspended), suspended:
		return text, model.StatusNone, model.StatusNone
	case nonNegative(obj, "status", "failed") > 0:
		return text, model.StatusNone, model.StatusWarning
	}
	return text, model.StatusNone, model.StatusPending
}

func jobDetail(obj *unstructured.Unstructured) []model.DetailSection {
	var d detailItems
	d.add("Completions", jobCompletions(obj))
	if n, ok := model.NestedInt64(obj, "spec", "parallelism"); ok {
		d.add("Parallelism", itoa(n))
	}
	d.add("Completion Mode", model.NestedString(obj, "spec", "completionMode"))
	if n, ok := model.NestedInt64(obj, "spec", "backoffLimit"); ok {
		d.add("Backoff Limit", itoa(n))
	}
	if n, ok := model.NestedInt64(obj, "spec", "activeDeadlineSeconds"); ok {
		d.add("Active Deadline", itoa(n)+"s")
	}
	if n, ok := model.NestedInt64(obj, "spec", "ttlSecondsAfterFinished"); ok {
		d.add("TTL After Finished", itoa(n)+"s")
	}
	if suspended, _ := model.NestedBool(obj, "spec", "suspend"); suspended {
		d.add("Suspend", condTrue)
	}
	d.add("Pods", itoa(nonNegative(obj, "status", "active"))+" active / "+
		itoa(nonNegative(obj, "status", "ready"))+" ready / "+
		itoa(nonNegative(obj, "status", "succeeded"))+" succeeded / "+
		itoa(nonNegative(obj, "status", "failed"))+" failed")
	start := model.NestedTime(obj, "status", "startTime")
	end := model.NestedTime(obj, "status", "completionTime")
	d.add("Started", timestamp(start))
	d.add("Completed", timestamp(end))
	if !start.IsZero() && !end.IsZero() && !end.Before(start) {
		d.add("Duration", end.Sub(start).Round(time.Second).String())
	}
	d.add("Selector", strings.Join(labelSelector(obj, "spec", "selector"), "\n"))
	d.add("Images", strings.Join(templateImages(obj), "\n"))
	sections := []model.DetailSection{{Title: "Job", Items: d}}
	if cs, ok := conditionSection(conditions(obj, "status", "conditions"), jobConditionToken); ok {
		sections = append(sections, cs)
	}
	return sections
}

func jobConditionToken(c condition) model.Status {
	switch c.Type {
	case jobFailed, jobFailureTarget:
		return conditionToken(c.Status, model.StatusError, model.StatusOK)
	case jobComplete, jobSuccessMet:
		return conditionToken(c.Status, model.StatusSucceeded, model.StatusNone)
	case jobSuspended:
		return conditionToken(c.Status, model.StatusNone, model.StatusNone)
	}
	return conditionToken(c.Status, model.StatusOK, model.StatusWarning)
}
