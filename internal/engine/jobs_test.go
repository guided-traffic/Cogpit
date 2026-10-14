package engine

import (
	"context"
	"errors"
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
