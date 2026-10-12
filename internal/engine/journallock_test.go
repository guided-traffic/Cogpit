package engine

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// journalHelperEnv makes the test binary a journal-writing helper process (TestJournalHelper).
const journalHelperEnv = "COGPIT_JOURNAL_HELPER"

// TestJournalHelper is not a test: started by the tests below with journalHelperEnv set, it plays
// a second Cogpit process on a shared journal directory.
//
//	write:<dir>:<id>:<n>  puts n drain and n Helm entries of its own, removing every second one
//	hold:<dir>:<node>     puts one running drain, says "ready" and waits to be killed
func TestJournalHelper(t *testing.T) {
	spec := os.Getenv(journalHelperEnv)
	if spec == "" {
		t.Skip("a helper process for the journal tests")
	}
	parts := strings.Split(spec, "|")
	fail := func(err error) {
		fmt.Fprintln(os.Stderr, "helper:", err)
		os.Exit(2)
	}
	switch parts[0] {
	case "write":
		journalSync = false // a stress test of the lock, not of the disk
		dir, id := parts[1], parts[2]
		n, _ := strconv.Atoi(parts[3])
		drains, helm := NewFileDrainJournal(dir), NewFileHelmJournal(dir)
		for i := range n {
			node := fmt.Sprintf("%s-%03d", id, i)
			if err := drains.Put(drainEntry("c", node)); err != nil {
				fail(err)
			}
			if err := helm.Put(PendingHelmOp{Context: "c", Namespace: "ns", Release: node, Action: HelmRollback, State: HelmOpRunning}); err != nil {
				fail(err)
			}
			if i%2 == 1 {
				if err := drains.Remove("c", node); err != nil {
					fail(err)
				}
				if err := helm.Remove("c", "ns", node); err != nil {
					fail(err)
				}
			}
		}
		os.Exit(0)
	case "hold":
		if err := NewFileDrainJournal(parts[1]).Put(drainEntry("c", parts[2])); err != nil {
			fail(err)
		}
		fmt.Println("ready")
		_, _ = io.Copy(io.Discard, os.Stdin) // until killed
		os.Exit(0)
	}
	fail(fmt.Errorf("unknown helper %q", spec))
}

func journalHelper(t *testing.T, spec string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestJournalHelper$", "-test.count=1")
	cmd.Env = append(os.Environ(), journalHelperEnv+"="+spec)
	cmd.Stderr = os.Stderr
	return cmd
}

// TestJournalsKeepEveryProcessesEntries: four Cogpit processes writing both journals of one
// directory at once lose none of each other's entries (H-22; without the lock a review's test
// kept 201 of 400, and this one, with the lock taken out, 85 of 120).
func TestJournalsKeepEveryProcessesEntries(t *testing.T) {
	if testing.Short() {
		t.Skip("starts processes")
	}
	dir := filepath.Join(t.TempDir(), "Cogpit")
	const procs, each = 4, 60
	var wg sync.WaitGroup
	errs := make(chan error, procs)
	start := time.Now()
	for p := range procs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cmd := journalHelper(t, fmt.Sprintf("write|%s|p%d|%d", dir, p, each))
			if err := cmd.Run(); err != nil {
				errs <- fmt.Errorf("process %d: %w", p, err)
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	t.Logf("%d processes, %d journal writes each, in %s", procs, 3*each, time.Since(start).Round(time.Millisecond))

	drains, err := NewFileDrainJournal(dir).List()
	if err != nil {
		t.Fatal(err)
	}
	ops, err := NewFileHelmJournal(dir).List()
	if err != nil {
		t.Fatal(err)
	}
	want := procs * each / 2
	if len(drains) != want || len(ops) != want {
		t.Fatalf("%d drain and %d Helm entries kept, want %d each", len(drains), len(ops), want)
	}
	owners := map[string]bool{}
	for _, e := range drains {
		if e.Owner == nil || !validOwnerID(e.Owner.ID) || e.Owner.PID == os.Getpid() {
			t.Fatalf("entry %s has owner %+v", e.Node, e.Owner)
		}
		owners[e.Owner.ID] = true
		if e.RunsElsewhere() {
			t.Errorf("the drain of %s runs elsewhere, but its process ended", e.Node)
		}
	}
	if len(owners) != procs {
		t.Errorf("%d owners, want one per process", len(owners))
	}
	// The processes ended; the next registration removes their owner files.
	if err := NewFileDrainJournal(dir).Put(drainEntry("c", "after")); err != nil {
		t.Fatal(err)
	}
	if files, _ := os.ReadDir(filepath.Join(dir, journalOwnersDir)); len(files) != 1 {
		t.Errorf("owner files after the sweep: %d, want this process's only", len(files))
	}
}

// TestJournalEntryOfAKilledProcess: a drain recorded by a live process is that process's; once it
// is killed, the entry is interrupted and this process may finish or dismiss it.
func TestJournalEntryOfAKilledProcess(t *testing.T) {
	if testing.Short() {
		t.Skip("starts a process")
	}
	dir := filepath.Join(t.TempDir(), "Cogpit")
	cmd := journalHelper(t, "hold|"+dir+"|worker-1")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer stdin.Close()
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	line, err := bufio.NewReader(out).ReadString('\n')
	if err != nil || strings.TrimSpace(line) != "ready" {
		_ = cmd.Process.Kill()
		t.Fatalf("helper: %q, %v", line, err)
	}

	j := NewFileDrainJournal(dir)
	list, err := j.List()
	if err != nil || len(list) != 1 || !list[0].RunsElsewhere() || list[0].Owner.PID != cmd.Process.Pid {
		_ = cmd.Process.Kill()
		t.Fatalf("while its process lives: %+v, %v", list, err)
	}
	var elsewhere *ElsewhereError
	if err := j.Remove("c", "worker-1"); !errors.As(err, &elsewhere) || elsewhere.PID != cmd.Process.Pid {
		t.Errorf("dismissing another process's running drain: %v", err)
	}
	if err := j.Put(drainEntry("c", "worker-1")); !errors.Is(err, ErrRunsElsewhere) {
		t.Errorf("starting it again: %v", err)
	}

	if err := cmd.Process.Kill(); err != nil { // SIGKILL: no cleanup runs, as in a crash
		t.Fatal(err)
	}
	_ = cmd.Wait()
	list, err = j.List()
	if err != nil || len(list) != 1 || list[0].RunsElsewhere() {
		t.Fatalf("after the process was killed: %+v, %v", list, err)
	}
	if err := j.Put(drainEntry("c", "worker-1")); err != nil {
		t.Fatalf("finishing the interrupted drain: %v", err)
	}
	if list, _ := j.List(); len(list) != 1 || list[0].Owner == nil || list[0].Owner.PID != os.Getpid() {
		t.Errorf("finishing took no ownership: %+v", list)
	}
}

// TestJournalOwnership plays two processes in one: an entry another live process runs is kept
// from this one; its failed entries are not; an entry without an owner is interrupted.
func TestJournalOwnership(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Cogpit")
	a, b := newJournalDir(dir), newJournalDir(dir)
	ja, jb := newFileDrainJournal(a), newFileDrainJournal(b)
	ha, hb := newFileHelmJournal(a), newFileHelmJournal(b)

	if err := ja.Put(drainEntry("c", "worker-1")); err != nil {
		t.Fatal(err)
	}
	op := PendingHelmOp{Context: "c", Namespace: "apps", Release: "web", Action: HelmUpgrade, FromRevision: 3, State: HelmOpRunning}
	if err := ha.Put(op); err != nil {
		t.Fatal(err)
	}
	if list, _ := ja.List(); len(list) != 1 || list[0].RunsElsewhere() {
		t.Errorf("a's own drain: %+v", list)
	}
	if list, _ := jb.List(); len(list) != 1 || !list[0].RunsElsewhere() || list[0].Owner.ID != a.selfID() {
		t.Errorf("a's drain as b sees it: %+v", list)
	}
	if ops, _ := hb.List(); len(ops) != 1 || !ops[0].RunsElsewhere() {
		t.Errorf("a's upgrade as b sees it: %+v", ops)
	}
	if err := jb.Put(drainEntry("c", "worker-1")); !errors.Is(err, ErrRunsElsewhere) {
		t.Errorf("b replaced a's running drain: %v", err)
	}
	if err := hb.Remove("c", "apps", "web"); !errors.Is(err, ErrRunsElsewhere) || !strings.Contains(err.Error(), "the upgrade of release apps/web") {
		t.Errorf("b removed a's running upgrade: %v", err)
	}
	if err := jb.Put(drainEntry("c", "worker-2")); err != nil { // b's own entries are b's
		t.Fatal(err)
	}

	failed := drainEntry("c", "worker-1")
	failed.State, failed.Error = DrainFailed, "timed out"
	if err := ja.Put(failed); err != nil {
		t.Fatal(err)
	}
	if list, _ := jb.List(); drainOf(list, "worker-1").RunsElsewhere() {
		t.Error("a failed drain counts as running elsewhere")
	}
	if err := jb.Put(drainEntry("c", "worker-1")); err != nil { // Finish drain of a's failed drain
		t.Fatalf("b finishing a's failed drain: %v", err)
	}
	if list, _ := ja.List(); !drainOf(list, "worker-1").RunsElsewhere() {
		t.Error("b's running drain is not b's for a")
	}

	// a ends: its lock goes with its file.
	_ = a.owner.f.Close()
	if ops, _ := hb.List(); len(ops) != 1 || ops[0].RunsElsewhere() {
		t.Errorf("an ended process's upgrade: %+v", ops)
	}
	if err := hb.Remove("c", "apps", "web"); err != nil {
		t.Errorf("dismissing an ended process's upgrade: %v", err)
	}
	c := newJournalDir(dir)
	if err := newFileDrainJournal(c).Put(drainEntry("c", "worker-3")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, journalOwnersDir, a.owner.ID+ownerFileSuffix)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the ended owner's file was not swept: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, journalOwnersDir, b.owner.ID+ownerFileSuffix)); err != nil {
		t.Errorf("a live owner's file was swept: %v", err)
	}

	// Entries written before owners existed are interrupted, as they were.
	legacy := `{"version":1,"drains":[{"context":"c","node":"old","started":"2026-10-11T09:30:00Z","startedBy":"gui","state":"running","settings":{}}]}`
	if err := os.WriteFile(filepath.Join(dir, DrainJournalFile), []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	if list, err := jb.List(); err != nil || len(list) != 1 || list[0].Owner != nil || list[0].RunsElsewhere() {
		t.Errorf("an entry without an owner: %+v, %v", list, err)
	}
	// An owner id that is not one Cogpit makes never becomes a path.
	forged := `{"version":1,"drains":[{"context":"c","node":"x","state":"running","settings":{},"owner":{"id":"../../journals","pid":1}}]}`
	if err := os.WriteFile(filepath.Join(dir, DrainJournalFile), []byte(forged), 0o600); err != nil {
		t.Fatal(err)
	}
	if list, err := jb.List(); err != nil || len(list) != 1 || list[0].RunsElsewhere() {
		t.Errorf("a forged owner: %+v, %v", list, err)
	}
}

func drainOf(list []PendingDrain, node string) PendingDrain {
	for _, e := range list {
		if e.Node == node {
			return e
		}
	}
	return PendingDrain{}
}

// TestJournalsConcurrentProcessesInOne: goroutines playing eight processes (eight directory
// views, eight owners) keep every entry, under the race detector.
func TestJournalsConcurrentProcessesInOne(t *testing.T) {
	journalSync = false
	defer func() { journalSync = true }()
	dir := filepath.Join(t.TempDir(), "Cogpit")
	const procs, each = 6, 20
	var wg sync.WaitGroup
	for p := range procs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			d := newJournalDir(dir)
			j, h := newFileDrainJournal(d), newFileHelmJournal(d)
			for i := range each {
				node := fmt.Sprintf("p%d-%02d", p, i)
				if err := j.Put(drainEntry("c", node)); err != nil {
					t.Error(err)
				}
				if err := h.Put(PendingHelmOp{Context: "c", Namespace: "ns", Release: node, Action: HelmUninstall, State: HelmOpFailed}); err != nil {
					t.Error(err)
				}
				if _, err := j.List(); err != nil {
					t.Error(err)
				}
			}
		}()
	}
	wg.Wait()
	drains, err1 := NewFileDrainJournal(dir).List()
	ops, err2 := NewFileHelmJournal(dir).List()
	if err1 != nil || err2 != nil || len(drains) != procs*each || len(ops) != procs*each {
		t.Fatalf("%d drains (%v), %d Helm entries (%v); want %d each", len(drains), err1, len(ops), err2, procs*each)
	}
}

// TestJournalLockTimesOut: a process that holds the lock for too long makes a write fail with
// ErrJournalBusy instead of hanging; the write goes through once the lock is free.
func TestJournalLockTimesOut(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Cogpit")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	held, err := lockPath(filepath.Join(dir, journalLockFile), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	old := journalLockWait
	journalLockWait = 100 * time.Millisecond
	defer func() { journalLockWait = old }()

	j := newFileDrainJournal(newJournalDir(dir))
	if err := j.Put(drainEntry("c", "worker-1")); !errors.Is(err, ErrJournalBusy) {
		t.Fatalf("a write while another process holds the lock: %v", err)
	}
	_ = unlockFile(held)
	_ = held.Close()
	journalLockWait = 5 * time.Second
	if err := j.Put(drainEntry("c", "worker-1")); err != nil {
		t.Fatalf("after the lock was released: %v", err)
	}
}

// TestDrainsOfAnotherProcess: a drain another live process runs is reported running elsewhere,
// is not offered as unfinished, and cannot be dismissed or started here.
func TestDrainsOfAnotherProcess(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Cogpit")
	other := newFileDrainJournal(newJournalDir(dir))
	if err := other.Put(drainEntry("c", "worker-1")); err != nil {
		t.Fatal(err)
	}
	d := NewDrains(newFileDrainJournal(newJournalDir(dir)))
	s := New(Options{Drains: d})
	defer s.Close()
	infos, err := s.PendingDrains(true)
	if err != nil || len(infos) != 1 || infos[0].Status != DrainStatusRunning || !infos[0].Elsewhere || infos[0].Progress != nil {
		t.Fatalf("pending drains: %+v, %v", infos, err)
	}
	if err := d.Forget("c", "worker-1"); !errors.Is(err, ErrRunsElsewhere) {
		t.Errorf("Dismiss: %v", err)
	}
	if err := d.Begin(drainEntry("c", "worker-1")); !errors.Is(err, ErrRunsElsewhere) || d.Running("c", "worker-1") {
		t.Errorf("Finish drain: %v (running %v)", err, d.Running("c", "worker-1"))
	}
}
