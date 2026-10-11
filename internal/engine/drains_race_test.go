package engine

import (
	"sync"
	"testing"
	"time"
)

// gateDrainJournal blocks its first Remove until gate closes, so that a test can run another
// journal operation in the middle of one.
type gateDrainJournal struct {
	*MemoryDrainJournal
	entered, gate chan struct{}
	once          sync.Once
}

func newGateDrainJournal() *gateDrainJournal {
	return &gateDrainJournal{MemoryDrainJournal: NewMemoryDrainJournal(), entered: make(chan struct{}), gate: make(chan struct{})}
}

func (g *gateDrainJournal) Remove(context, node string) error {
	g.once.Do(func() { close(g.entered); <-g.gate })
	return g.MemoryDrainJournal.Remove(context, node)
}

// drainBeginDuring starts a Begin while another journal operation is blocked in the gate, gives it
// time to finish (it must wait for that operation instead), opens the gate and returns Begin's
// error once both are done.
func drainBeginDuring(t *testing.T, d *Drains, j *gateDrainJournal, p PendingDrain, other <-chan error) error {
	t.Helper()
	<-j.entered
	begun := make(chan error, 1)
	go func() { begun <- d.Begin(p) }()
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

func drainJournaled(t *testing.T, j DrainJournal, context, node string) bool {
	t.Helper()
	list, err := j.List()
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range list {
		if e.Context == context && e.Node == node {
			return true
		}
	}
	return false
}

// TestDrainsEndDoesNotDropTheNextDrain: a drain that starts while the previous one on the node is
// being recorded as finished keeps its journal entry (D-11).
func TestDrainsEndDoesNotDropTheNextDrain(t *testing.T) {
	j := newGateDrainJournal()
	d := NewDrains(j)
	p := PendingDrain{Context: "c", Node: "worker-1"}
	if err := d.Begin(p); err != nil {
		t.Fatal(err)
	}
	ended := make(chan error, 1)
	go func() { ended <- d.End(p, nil) }()
	if err := drainBeginDuring(t, d, j, p, ended); err != nil {
		t.Fatalf("the next drain: %v", err)
	}
	if !d.Running("c", "worker-1") || !drainJournaled(t, j, "c", "worker-1") {
		t.Errorf("running %v, journaled %v: a drain runs without its journal entry", d.Running("c", "worker-1"), drainJournaled(t, j, "c", "worker-1"))
	}
}

// TestDrainsForgetDoesNotDropARunningDrain: Dismiss and Finish of the same node at once never
// leave the running drain without its journal entry.
func TestDrainsForgetDoesNotDropARunningDrain(t *testing.T) {
	j := newGateDrainJournal()
	p := PendingDrain{Context: "c", Node: "worker-1", State: DrainFailed}
	if err := j.Put(p); err != nil {
		t.Fatal(err)
	}
	d := NewDrains(j)
	forgot := make(chan error, 1)
	go func() { forgot <- d.Forget("c", "worker-1") }()
	if err := drainBeginDuring(t, d, j, p, forgot); err != nil {
		t.Fatalf("finish: %v", err)
	}
	if !d.Running("c", "worker-1") || !drainJournaled(t, j, "c", "worker-1") {
		t.Errorf("running %v, journaled %v: a drain runs without its journal entry", d.Running("c", "worker-1"), drainJournaled(t, j, "c", "worker-1"))
	}
}
