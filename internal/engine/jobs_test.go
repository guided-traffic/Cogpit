package engine

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// jobTabOf returns a dock tab's job as the UI state holds it.
func jobTabOf(s *Session, id string) (*JobTab, bool) {
	t, ok := s.UIState().DockTab(id)
	if !ok || t.Job == nil {
		return nil, false
	}
	return t.Job, true
}

// connectedJobSession is a session connected to "test".
func connectedJobSession(t *testing.T) *Session {
	t.Helper()
	s, _ := newTestSession(t, testkit.NewMemorySource(), 5*time.Millisecond)
	connectTest(t, s)
	return s
}

// TestJobRunsInItsTab (A91): a job opens its own dock tab, active, with its steps pending; its Run
// moves them, and the tab shows how it ended: done with its outcome, the time it ended.
func TestJobRunsInItsTab(t *testing.T) {
	s := connectedJobSession(t)
	ref := model.ObjectRef{Context: "test", Kind: "pods", Namespace: "default", Name: "web-0"}
	step := make(chan struct{})
	tab, err := s.StartJob(JobSpec{
		Command: "test.job", Title: "A job", Context: "test",
		Steps: []JobStep{{Ref: &ref, Text: "first"}, {Text: "second"}},
		Run: func(ctx context.Context, j *Job) (string, error) {
			j.SetStep(0, JobRunning, "under way")
			<-step
			j.SetStep(0, JobDone, "fine")
			j.SetStep(1, JobDone, "")
			return "All done.", nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if u := s.UIState(); u.ActiveDock != tab.ID || tab.Type != DockJob || tab.Title != "A job" {
		t.Errorf("tab %+v, active %q", tab, u.ActiveDock)
	}
	waitFor(t, "the first step under way", func() bool {
		j, ok := jobTabOf(s, tab.ID)
		return ok && j.State == JobRunning && j.Steps[0].State == JobRunning && j.Steps[0].Detail == "under way" && j.Steps[1].State == JobPending
	})
	close(step)
	waitFor(t, "the job's end", func() bool { j, ok := jobTabOf(s, tab.ID); return ok && j.State.Ended() })
	j, _ := jobTabOf(s, tab.ID)
	if j.State != JobDone || j.Outcome != "All done." || j.Ended.IsZero() || j.Steps[0].Detail != "fine" || j.Steps[0].Ref == nil || j.Command != "test.job" {
		t.Errorf("job %+v", j)
	}
	// The UI state's copies are its own.
	u := s.UIState()
	u.Dock[0].Job.Steps[0].Text = "changed"
	if j, _ := jobTabOf(s, tab.ID); j.Steps[0].Text != "first" {
		t.Error("a clone of the UI state shares the job's steps")
	}
}

// TestJobFailsWithItsError (A91): a Run that returns an error fails the job with the error as a
// sentence; one that panics fails it too, and the session goes on.
func TestJobFailsWithItsError(t *testing.T) {
	s := connectedJobSession(t)
	failing, _ := s.StartJob(JobSpec{Title: "failing", Context: "test", Run: func(context.Context, *Job) (string, error) {
		return "", errors.New("the source is not ready")
	}})
	panicking, _ := s.StartJob(JobSpec{Title: "panicking", Context: "test", Run: func(context.Context, *Job) (string, error) {
		panic("boom")
	}})
	waitFor(t, "both ends", func() bool {
		a, _ := jobTabOf(s, failing.ID)
		b, _ := jobTabOf(s, panicking.ID)
		return a != nil && b != nil && a.State.Ended() && b.State.Ended()
	})
	a, _ := jobTabOf(s, failing.ID)
	b, _ := jobTabOf(s, panicking.ID)
	if a.State != JobFailed || a.Outcome != "The source is not ready." {
		t.Errorf("failing %+v", a)
	}
	if b.State != JobFailed || b.Outcome != "An internal error ended it." {
		t.Errorf("panicking %+v", b)
	}
}

// TestJobStops (A91): closing a job's tab cancels its work, and so do disconnecting its cluster and
// closing the session; the tab, while it is open, says why it stopped.
func TestJobStops(t *testing.T) {
	s := connectedJobSession(t)
	causes := make(chan error, 3)
	waiting := func(ctx context.Context, _ *Job) (string, error) {
		<-ctx.Done()
		causes <- context.Cause(ctx)
		return "", ctx.Err()
	}
	closed, _ := s.StartJob(JobSpec{Title: "closed", Context: "test", Run: waiting})
	kept, _ := s.StartJob(JobSpec{Title: "kept", Context: "test", Run: waiting})
	s.UpdateUI(func(u *UIState) {
		for i := range u.Dock {
			if u.Dock[i].ID == closed.ID {
				u.Dock = append(u.Dock[:i], u.Dock[i+1:]...)
				return
			}
		}
	})
	if err := <-causes; !errors.Is(err, errJobTabClosed) {
		t.Errorf("closing the tab: %v", err)
	}
	if j, ok := jobTabOf(s, kept.ID); !ok || j.State != JobRunning {
		t.Errorf("the other job %+v", j)
	}

	if err := s.Disconnect(); err != nil {
		t.Fatal(err)
	}
	if err := <-causes; !errors.Is(err, errJobClusterChanged) {
		t.Errorf("disconnecting: %v", err)
	}
	waitFor(t, "the kept tab's end", func() bool { j, ok := jobTabOf(s, kept.ID); return !ok || j.State.Ended() })
	if j, ok := jobTabOf(s, kept.ID); ok && (j.State != JobCancelled || j.Outcome != "Stopped: the connected cluster changed.") {
		t.Errorf("after disconnecting %+v", j)
	}
	if _, err := s.StartJob(JobSpec{Title: "late", Context: "test", Run: waiting}); !errors.Is(err, ErrWrongCluster) {
		t.Errorf("a job on a cluster not connected: %v", err)
	}

	connectTest(t, s)
	last, _ := s.StartJob(JobSpec{Title: "last", Context: "test", Run: waiting})
	s.Close()
	if err := <-causes; !errors.Is(err, errJobSessionClosed) {
		t.Errorf("closing the session: %v", err)
	}
	if _, err := s.StartJob(JobSpec{Title: "after", Context: "test", Run: waiting}); err == nil {
		t.Error("a job after the session closed")
	}
	_ = last
}

// TestJobNeedsWork: a job without a Run is refused.
func TestJobNeedsWork(t *testing.T) {
	s := connectedJobSession(t)
	if _, err := s.StartJob(JobSpec{Title: "idle", Context: "test"}); err == nil || !strings.Contains(err.Error(), "work") {
		t.Errorf("a job without work: %v", err)
	}
}

// TestJobsRace starts, updates and closes jobs from several goroutines. Run with -race.
func TestJobsRace(t *testing.T) {
	s := connectedJobSession(t)
	done := make(chan string, 20)
	for range 10 {
		go func() {
			tab, err := s.StartJob(JobSpec{Title: "race", Context: "test", Steps: []JobStep{{Text: "a"}, {Text: "b"}},
				Run: func(ctx context.Context, j *Job) (string, error) {
					for i := range 20 {
						j.SetStep(i%2, JobRunning, "step")
						if ctx.Err() != nil {
							return "", ctx.Err()
						}
					}
					return "ok", nil
				}})
			if err != nil {
				done <- ""
				return
			}
			done <- tab.ID
		}()
	}
	for range 10 {
		id := <-done
		if id == "" {
			continue
		}
		s.UpdateUI(func(u *UIState) {
			for i := range u.Dock {
				if u.Dock[i].ID == id {
					u.Dock = append(u.Dock[:i], u.Dock[i+1:]...)
					return
				}
			}
		})
	}
	waitFor(t, "every job's end", func() bool {
		s.jobs.mu.Lock()
		defer s.jobs.mu.Unlock()
		return len(s.jobs.running) == 0
	})
}

// TestJobCancelKeepsItsTab (G4): job.cancel's CancelJob stops a job's work and keeps its tab, whose
// outcome says it was cancelled (by an agent, when one did) and adds what Run said it did and did
// not do; the steps' counts follow the steps. A job that ended, a tab that is no job and an unknown
// tab cannot be cancelled.
func TestJobCancelKeepsItsTab(t *testing.T) {
	s := connectedJobSession(t)
	started := make(chan struct{}, 2)
	run := func(ctx context.Context, j *Job) (string, error) {
		j.SetStep(0, JobDone, "Deleted")
		j.SetStep(1, JobRunning, "")
		started <- struct{}{}
		<-ctx.Done()
		j.SetStep(1, JobDone, "Deleted")
		j.SetStep(2, JobCancelled, "Not attempted: "+context.Cause(ctx).Error())
		return "", errors.New("deleted: 2 of 3 objects; 1 not attempted")
	}
	steps := []JobStep{{Text: "a"}, {Text: "b"}, {Text: "c"}}
	byUser, _ := s.StartJob(JobSpec{Title: "user's", Context: "test", Steps: steps, Run: run})
	byAgent, _ := s.StartJob(JobSpec{Title: "agent's", Context: "test", Steps: steps, Run: run})
	<-started
	<-started
	if j, _ := jobTabOf(s, byUser.ID); j.Counts != (JobCounts{Total: 3, Done: 1, Running: 1, Pending: 1}) {
		t.Errorf("counts while running %+v", j.Counts)
	}

	tab, err := s.CancelJob(context.Background(), byUser.ID, OriginGUI)
	if err != nil {
		t.Fatal(err)
	}
	if tab.Job == nil || tab.Job.State != JobCancelled ||
		tab.Job.Outcome != "Stopped: cancelled. Deleted: 2 of 3 objects; 1 not attempted." ||
		tab.Job.Counts != (JobCounts{Total: 3, Done: 2, Cancelled: 1}) {
		t.Errorf("cancelled by the user %+v", tab.Job)
	}
	if u := s.UIState(); len(u.Dock) != 2 {
		t.Errorf("the cancelled job's tab closed: %+v", u.Dock)
	}
	tab, err = s.CancelJob(context.Background(), byAgent.ID, OriginExternalAgent)
	if err != nil || tab.Job == nil || !strings.HasPrefix(tab.Job.Outcome, "Stopped: cancelled by an agent. ") {
		t.Errorf("cancelled by an agent: %v %+v", err, tab.Job)
	}

	if _, err := s.CancelJob(context.Background(), byUser.ID, OriginGUI); !errors.Is(err, ErrJobEnded) {
		t.Errorf("cancelling an ended job: %v", err)
	}
	if err := s.JobCancellable("job-99"); err == nil || !strings.Contains(err.Error(), "no dock tab") {
		t.Errorf("an unknown tab: %v", err)
	}
	s.UpdateUI(func(u *UIState) { u.Dock = append(u.Dock, DockTab{ID: "logs-1", Type: DockLogs}) })
	if err := s.JobCancellable("logs-1"); err == nil || !strings.Contains(err.Error(), "not a job") {
		t.Errorf("a logs tab: %v", err)
	}
}

// TestJobEnd: how a job ends: done when Run returned no error, even after a stop that came after
// its last step; stopped with why and what Run said; failed with Run's error.
func TestJobEnd(t *testing.T) {
	for _, tc := range []struct {
		outcome    string
		err, cause error
		state      JobState
		text       string
	}{
		{"All 12 deleted.", nil, nil, JobDone, "All 12 deleted."},
		{"All 12 deleted.", nil, errJobCancelled, JobDone, "All 12 deleted."},
		{"", context.Canceled, errJobTabClosed, JobCancelled, "Stopped: its tab was closed."},
		{"", errors.New("deleted: 3 of 12 objects; 9 not attempted"), errJobClusterChanged, JobCancelled,
			"Stopped: the connected cluster changed. Deleted: 3 of 12 objects; 9 not attempted."},
		{"", errors.New("the source is not ready"), nil, JobFailed, "The source is not ready."},
		{"", errors.New("the source is not ready"), context.Canceled, JobFailed, "The source is not ready."},
	} {
		if state, text := jobEnd(tc.outcome, tc.err, tc.cause); state != tc.state || text != tc.text {
			t.Errorf("jobEnd(%q, %v, %v) = %s %q, want %s %q", tc.outcome, tc.err, tc.cause, state, text, tc.state, tc.text)
		}
	}
}

type jobProbeParams struct {
	Steps int `json:"steps"`
}

// TestJobKeepsItsCommandsContext (G4): a job started by a command keeps the command's context
// values (its origin, the confirmation that bound it) but outlives the call; an agent's job gets an
// "ended" audit line after its command's "requested" and "done" lines, with its outcome.
func TestJobKeepsItsCommandsContext(t *testing.T) {
	audit := &lockedBuffer{}
	fc := &fakeConnector{src: testkit.NewMemorySource(), profiles: testProfiles()}
	s := New(Options{Connector: fc, SnapshotInterval: 5 * time.Millisecond, Audit: audit})
	t.Cleanup(s.Close)
	connectTest(t, s)
	s.SetAgentWrites("test", true)
	type seen struct {
		origin  Origin
		context string
	}
	seenIn := make(chan seen, 1)
	release := make(chan struct{})
	err := Register(s.Registry(), Spec[jobProbeParams]{
		ID: "probe.job", Label: "Probe", Danger: DangerMutating, Verbs: []Verb{{Verb: "patch", Resource: "pods"}},
		Targets: func(*Session, jobProbeParams) ([]model.ObjectRef, error) { return nil, nil },
		Run: func(ctx context.Context, s *Session, p jobProbeParams) (any, error) {
			return s.StartJob(JobSpec{Command: "probe.job", Title: "Probe", Context: "test", Parent: ctx,
				Steps: make([]JobStep, p.Steps),
				Run: func(jctx context.Context, j *Job) (string, error) {
					req, _ := Confirmed(jctx)
					seenIn <- seen{OriginOf(jctx), req.Context}
					<-release
					if jctx.Err() != nil {
						return "", jctx.Err()
					}
					return "Probed.", nil
				}})
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	res, err := s.Dispatch(ctx, Invocation{ID: "probe.job", Params: json.RawMessage(`{"steps":2}`), Origin: OriginExternalAgent})
	if err != nil {
		t.Fatal(err)
	}
	cancel() // the call ended: the job goes on
	got := <-seenIn
	if got.origin != OriginExternalAgent || got.context != "test" {
		t.Errorf("the job's context: %+v", got)
	}
	close(release)
	id := res.(DockTab).ID
	waitFor(t, "the job's end", func() bool { j, ok := jobTabOf(s, id); return ok && j.State.Ended() })
	if j, _ := jobTabOf(s, id); j.State != JobDone || j.Outcome != "Probed." {
		t.Errorf("the job after its call ended: %+v", j)
	}
	waitFor(t, "the ended line", func() bool { return strings.Contains(audit.String(), `"phase":"ended"`) })
	var phases []string
	for line := range strings.SplitSeq(strings.TrimSpace(audit.String()), "\n") {
		var e AuditEntry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatal(err)
		}
		if e.Command == "probe.job" {
			phases = append(phases, e.Phase+" "+e.Context+" "+strconv.FormatBool(e.OK)+" "+string(e.Params))
		}
	}
	want := []string{`requested test true {"steps":2}`, `done test true {"steps":2}`, `ended test true {"steps":2}`}
	if !slices.Equal(phases, want) {
		t.Errorf("audit lines %q, want %q", phases, want)
	}
}

// TestJobStopsWhenInterrupted: a connection that stops answering stops its cluster's jobs with
// their own reason (D-58, as after a crash: nothing continues by itself after the reconnect).
func TestJobStopsWhenInterrupted(t *testing.T) {
	s := connectedJobSession(t)
	causes := make(chan error, 1)
	tab, _ := s.StartJob(JobSpec{Title: "waits", Context: "test", Run: func(ctx context.Context, _ *Job) (string, error) {
		<-ctx.Done()
		causes <- context.Cause(ctx)
		return "", ctx.Err()
	}})
	s.UpdateUI(func(u *UIState) { u.Connection.State = Interrupted })
	if err := <-causes; !errors.Is(err, errJobInterrupted) {
		t.Errorf("interrupted: %v", err)
	}
	waitFor(t, "the job's end", func() bool { j, ok := jobTabOf(s, tab.ID); return ok && j.State.Ended() })
	if j, _ := jobTabOf(s, tab.ID); j.State != JobCancelled || !strings.Contains(j.Outcome, "interrupted") {
		t.Errorf("after the interruption %+v", j)
	}
}
