package engine

import (
	"sync"
	"testing"
	"time"
)

// gateJournal blocks its first Remove until gate closes, so that a test can run another journal
// operation in the middle of one.
type gateJournal struct {
	*MemoryHelmJournal
	entered, gate chan struct{}
	once          sync.Once
}

func newGateJournal() *gateJournal {
	return &gateJournal{MemoryHelmJournal: NewMemoryHelmJournal(), entered: make(chan struct{}), gate: make(chan struct{})}
}

func (g *gateJournal) Remove(context, namespace, release string) error {
	g.once.Do(func() { close(g.entered); <-g.gate })
	return g.MemoryHelmJournal.Remove(context, namespace, release)
}

// beginDuring starts a Begin while another journal operation is blocked in the gate, gives it
// time to finish (it must wait for that operation instead), opens the gate and returns Begin's
// error once both are done.
func beginDuring(t *testing.T, h *HelmOps, j *gateJournal, op PendingHelmOp, other <-chan error) error {
	t.Helper()
	<-j.entered
	begun := make(chan error, 1)
	go func() { begun <- h.Begin(op) }()
	select {
	case err := <-begun: // the race: Begin ran in the middle of the other operation
		close(j.gate)
		if oerr := <-other; oerr != nil {
			t.Fatalf("the other operation: %v", oerr)
		}
		return err
	case <-time.After(200 * time.Millisecond):
	}
	close(j.gate)
	if err := <-other; err != nil {
		t.Fatalf("the other operation: %v", err)
	}
	return <-begun
}

// TestHelmOpsEndDoesNotDropTheNextAction: an action that starts while the previous one on the
// release is being recorded as finished keeps its journal entry (D-11): End's removal of the old
// entry must not remove the new one, or a crash during the new action leaves nothing to offer.
func TestHelmOpsEndDoesNotDropTheNextAction(t *testing.T) {
	j := newGateJournal()
	h := NewHelmOps(j)
	op := PendingHelmOp{Context: "c", Namespace: "ns", Release: "web", Action: HelmUninstall}
	if err := h.Begin(op); err != nil {
		t.Fatal(err)
	}
	ended := make(chan error, 1)
	go func() { ended <- h.End(op, nil) }()
	if err := beginDuring(t, h, j, op, ended); err != nil {
		t.Fatalf("the next action: %v", err)
	}
	if _, ok, _ := h.Pending("c", "ns", "web"); !h.Running("c", "ns", "web") || !ok {
		t.Errorf("running %v, journal entry %v: an action runs without its journal entry", h.Running("c", "ns", "web"), ok)
	}
}

// TestHelmOpsDismissDoesNotDropARunningAction: Dismiss of a failed action and Finish of the same
// release at once (the user's banner and an agent) never leave the running action without its
// journal entry.
func TestHelmOpsDismissDoesNotDropARunningAction(t *testing.T) {
	j := newGateJournal()
	op := PendingHelmOp{Context: "c", Namespace: "ns", Release: "web", Action: HelmRollback, Revision: 1, FromRevision: 2, State: HelmOpFailed}
	if err := j.Put(op); err != nil {
		t.Fatal(err)
	}
	h := NewHelmOps(j)
	forgot := make(chan error, 1)
	go func() { forgot <- h.Forget("c", "ns", "web") }()
	if err := beginDuring(t, h, j, op, forgot); err != nil {
		t.Fatalf("finish: %v", err)
	}
	if _, ok, _ := h.Pending("c", "ns", "web"); !h.Running("c", "ns", "web") || !ok {
		t.Errorf("running %v, journal entry %v: an action runs without its journal entry", h.Running("c", "ns", "web"), ok)
	}
}
