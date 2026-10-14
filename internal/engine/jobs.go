package engine

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/guided-traffic/cogpit/internal/model"
)

// Background jobs (A91, the owner's decision of 2026-10-14 evening). A command whose work outlives
// its call, as Reconcile with Source does while it waits for Flux's controllers, starts a job: the
// command returns at once with the job's dock tab, and the job runs on a goroutine of the
// session's. The tab is engine state (DockTab.Job): the job's steps, each pending, running, done or
// failed with a line of detail, its state and its outcome, so that agents read its progress with
// ui_state as the window shows it in the dock.
//
// A job ends when its work ends, when the user closes its tab (closing stops the job, it undoes
// nothing written so far), when the connected cluster changes or goes, or when the session closes.
// Jobs are not journaled: a crash ends one, and what it had written stays; each job's command says
// why that is harmless (D-11). A job's goroutine recovers from a panic at its boundary, which fails
// the job.

// DockJob: a background job's progress (jobs.go).
const DockJob DockType = "job"

// JobState is the state of a job and of each of its steps.
type JobState string

// Job and step states.
const (
	JobPending   JobState = "pending"   // a step not started yet
	JobRunning   JobState = "running"   // a job, or a step, under way
	JobDone      JobState = "done"      // finished as asked
	JobFailed    JobState = "failed"    // ended with an error (a step: Detail says why)
	JobCancelled JobState = "cancelled" // the tab was closed, the cluster changed, or the session closed
)

// Ended reports whether a state is final.
func (st JobState) Ended() bool { return st == JobDone || st == JobFailed || st == JobCancelled }

// JobStep is one step of a job: what it does and how far it got.
type JobStep struct {
	// Ref is the object the step acts on, if any.
	Ref    *model.ObjectRef `json:"ref,omitempty"`
	Text   string           `json:"text"`
	State  JobState         `json:"state"`
	Detail string           `json:"detail,omitempty"`
}

// JobTab is a job tab's progress (DockTab.Job).
type JobTab struct {
	// Command is the command that started the job ("flux.reconcileWithSource").
	Command string    `json:"command"`
	State   JobState  `json:"state"`
	Steps   []JobStep `json:"steps"`
	Started time.Time `json:"started"`
	Ended   time.Time `json:"ended,omitzero"`
	// Outcome says in one sentence how the job ended; empty while it runs.
	Outcome string `json:"outcome,omitempty"`
}

func (j *JobTab) clone() *JobTab {
	if j == nil {
		return nil
	}
	c := *j
	c.Steps = slices.Clone(j.Steps)
	for i, st := range c.Steps {
		if st.Ref != nil {
			r := *st.Ref
			c.Steps[i].Ref = &r
		}
	}
	return &c
}

// JobSpec describes a job StartJob starts.
type JobSpec struct {
	// Command is the command that starts it; Title the tab's title; Context the cluster it acts
	// on, the connected one.
	Command, Title, Context string
	// Steps are its steps, shown pending until Run moves them.
	Steps []JobStep
	// Run does the job's work on a goroutine of the session's and returns its outcome, one
	// sentence, or the error that failed it. ctx ends when the job is cancelled. Run reports its
	// steps' progress with Job.SetStep.
	Run func(ctx context.Context, j *Job) (outcome string, err error)
}

// Job is a running job, as its Run sees it.
type Job struct {
	s  *Session
	id string
}

// ID is the job's dock tab id.
func (j *Job) ID() string { return j.id }

// SetStep moves step i to state with a line of detail; a step of a job that ended, or a tab that
// is gone, is left alone.
func (j *Job) SetStep(i int, state JobState, detail string) {
	j.s.updateJob(j.id, func(t *JobTab) {
		if i < 0 || i >= len(t.Steps) || t.State.Ended() {
			return
		}
		t.Steps[i].State, t.Steps[i].Detail = state, detail
	})
}

// Step returns step i as the tab holds it now, and whether the tab still holds the job.
func (j *Job) Step(i int) (JobStep, bool) {
	t, ok := j.s.UIState().DockTab(j.id)
	if !ok || t.Job == nil || i < 0 || i >= len(t.Job.Steps) {
		return JobStep{}, false
	}
	return t.Job.Steps[i], true
}

// Errors that cancel a job.
var (
	errJobTabClosed      = errors.New("its tab was closed")
	errJobClusterChanged = errors.New("the connected cluster changed")
	errJobSessionClosed  = errors.New("the app is closing")
)

// jobSet is a session's running jobs, by tab id: how to cancel each and the cluster it acts on.
type jobSet struct {
	mu      sync.Mutex
	running map[string]runningJob
	closed  bool
}

type runningJob struct {
	context string
	cancel  context.CancelCauseFunc
}

// StartJob opens a job's dock tab, makes it the active one, and runs the job. It returns the tab.
func (s *Session) StartJob(spec JobSpec) (DockTab, error) {
	if spec.Run == nil {
		return DockTab{}, errors.New("a job needs work to do")
	}
	u := s.UIState()
	if u.Connection.State != Connected || u.Context != spec.Context {
		return DockTab{}, fmt.Errorf("%w: the job acts on %q", ErrWrongCluster, spec.Context)
	}
	steps := slices.Clone(spec.Steps)
	for i := range steps {
		if steps[i].State == "" {
			steps[i].State = JobPending
		}
	}
	tab := DockTab{
		ID: s.NewDockID(DockJob), Type: DockJob, Title: spec.Title, Context: spec.Context,
		Job: &JobTab{Command: spec.Command, State: JobRunning, Steps: steps, Started: s.Now()},
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	s.jobs.mu.Lock()
	if s.jobs.closed {
		s.jobs.mu.Unlock()
		cancel(errJobSessionClosed)
		return DockTab{}, errJobSessionClosed
	}
	if s.jobs.running == nil {
		s.jobs.running = map[string]runningJob{}
	}
	s.jobs.running[tab.ID] = runningJob{context: spec.Context, cancel: cancel}
	s.jobs.mu.Unlock()
	s.UpdateUI(func(u *UIState) {
		u.Dock = append(u.Dock, tab)
		u.ActiveDock = tab.ID
	})
	j := &Job{s: s, id: tab.ID}
	s.goSafe("job "+tab.Title, func() {
		var (
			outcome string
			err     = errJobPanicked
		)
		defer func() {
			cancel(nil)
			s.jobs.forget(tab.ID)
			s.endJob(tab.ID, outcome, err, context.Cause(ctx))
		}()
		outcome, err = spec.Run(ctx, j)
	})
	return tab, nil
}

// errJobPanicked fails a job whose Run panicked (goSafe logs the panic and tells the user).
var errJobPanicked = errors.New("an internal error ended it")

// endJob records how a job ended: done with its outcome, failed with its error, or cancelled with
// why. Steps still pending or running are left as they are, so that the tab shows where it stopped.
func (s *Session) endJob(id, outcome string, err, cause error) {
	now := s.Now()
	s.updateJob(id, func(t *JobTab) {
		t.Ended = now
		switch {
		case cause != nil && !errors.Is(cause, context.Canceled):
			t.State, t.Outcome = JobCancelled, "Stopped: "+cause.Error()+"."
		case err != nil:
			t.State, t.Outcome = JobFailed, upperFirst(err.Error())
		default:
			t.State, t.Outcome = JobDone, outcome
		}
	})
}

// updateJob changes an open job tab's progress with fn.
func (s *Session) updateJob(id string, fn func(*JobTab)) {
	if t, ok := s.UIState().DockTab(id); !ok || t.Job == nil {
		return // closed: nothing to show it on, and no UI state change to announce
	}
	s.UpdateUI(func(u *UIState) {
		for i := range u.Dock {
			if u.Dock[i].ID == id && u.Dock[i].Job != nil {
				job := u.Dock[i].Job.clone()
				fn(job)
				u.Dock[i].Job = job
				return
			}
		}
	})
}

// followJobs cancels the jobs whose tab a UI state change closed, and every job when the connected
// cluster changed or went. UpdateUI calls it after the change.
func (s *Session) followJobs(old, now UIState) {
	s.jobs.mu.Lock()
	var cancels []func()
	for id, r := range s.jobs.running {
		has := func(dock []DockTab) bool {
			return slices.ContainsFunc(dock, func(t DockTab) bool { return t.ID == id })
		}
		switch {
		case has(old.Dock) && !has(now.Dock): // this change closed it (StartJob registers a job before its tab opens)
			cancels = append(cancels, func() { r.cancel(errJobTabClosed) })
		case now.Connection.State != Connected || now.Context != r.context:
			if old.Connection.State == Connected && old.Context == r.context {
				cancels = append(cancels, func() { r.cancel(errJobClusterChanged) })
			}
		}
	}
	s.jobs.mu.Unlock()
	for _, c := range cancels {
		c()
	}
}

// forget drops a job that ended.
func (js *jobSet) forget(id string) {
	js.mu.Lock()
	defer js.mu.Unlock()
	delete(js.running, id)
}

// stopAll cancels every job and takes no new ones: the session closes.
func (js *jobSet) stopAll() {
	js.mu.Lock()
	js.closed = true
	running := js.running
	js.running = nil
	js.mu.Unlock()
	for _, r := range running {
		r.cancel(errJobSessionClosed)
	}
}

// upperFirst writes an error as a sentence: its first letter upper case, a full stop at the end.
func upperFirst(s string) string {
	if s == "" {
		return s
	}
	if c := s[0]; c >= 'a' && c <= 'z' {
		s = string(c-'a'+'A') + s[1:]
	}
	if s[len(s)-1] != '.' {
		s += "."
	}
	return s
}
