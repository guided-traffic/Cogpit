package commands_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

func mustJSON(t testing.TB, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestCordonAndUncordonTakeEachOthersPlace: on a schedulable node Cordon applies and Uncordon does
// not, on a cordoned node the other way round (captured: the row menu and the drawer show one of
// the two), from spec.unschedulable in the cache, even where the cluster cannot be changed.
func TestCordonAndUncordonTakeEachOthersPlace(t *testing.T) {
	h, w := newActionsHarness(t, nil, engine.Options{})
	reg := h.s.Registry()
	cordon, _ := reg.Get("node.cordon")
	uncordon, _ := reg.Get("node.uncordon")
	schedulable, cordoned := mustJSON(t, actionNodeRef("worker-1")), mustJSON(t, actionNodeRef("worker-2"))

	if !cordon.Applicable(h.s, schedulable) || cordon.Applicable(h.s, cordoned) {
		t.Error("Cordon: applicable on a schedulable node only")
	}
	if uncordon.Applicable(h.s, schedulable) || !uncordon.Applicable(h.s, cordoned) {
		t.Error("Uncordon: applicable on a cordoned node only")
	}
	if err := cordon.Enabled(h.s, cordoned); !errors.Is(err, engine.ErrNotApplicable) || !strings.Contains(err.Error(), "cordoned already") {
		t.Errorf("Cordon on a cordoned node: %v", err)
	}
	if err := uncordon.Enabled(h.s, schedulable); !errors.Is(err, engine.ErrNotApplicable) {
		t.Errorf("Uncordon on a schedulable node: %v", err)
	}
	pod := mustJSON(t, map[string]any{"ref": model.ObjectRef{Kind: "pods", Namespace: "shop", Name: "web-0"}})
	if cordon.Applicable(h.s, pod) {
		t.Error("Cordon applies to a pod")
	}
	missing := mustJSON(t, actionNodeRef("gone"))
	if err := cordon.Enabled(h.s, missing); err == nil || errors.Is(err, engine.ErrNotApplicable) || !strings.Contains(err.Error(), "not found") {
		t.Errorf("Cordon on a missing node: %v (disabled with a reason, not hidden)", err)
	}

	// From the GUI, mutating commands run without a confirmation.
	var res commands.ActionResult
	h.must(t, engine.OriginGUI, "node.cordon", actionNodeRef("worker-1"), &res)
	h.must(t, engine.OriginGUI, "node.uncordon", actionNodeRef("worker-2"), nil)
	if got := w.Calls(); !slices.Equal(got, []string{"cordon worker-1", "uncordon worker-2"}) {
		t.Errorf("writes %v", got)
	}
	if res.Action != "cordoned" || res.Ref.Context != "test" || res.Ref.Namespace != "" {
		t.Errorf("result %+v", res)
	}
	// The cache follows the cluster: the two commands swap places.
	waitApplicable(t, h.s, uncordon, schedulable, true)
	waitApplicable(t, h.s, cordon, cordoned, true)
}

func waitApplicable(t *testing.T, s *engine.Session, c *engine.Command, params json.RawMessage, want bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if !testkit.WaitFor(ctx, func() bool { return c.Applicable(s, params) == want }) {
		t.Fatalf("%s applicable %v never became %v: %v", c.ID, !want, want, c.Enabled(s, params))
	}
}

// TestNodeCommandsConfirmationPerOrigin: Cordon is mutating, so agents need the user's acceptance
// unless agent writes are allowed for the context; Drain is destructive, so every origin needs it.
func TestNodeCommandsConfirmationPerOrigin(t *testing.T) {
	audit := &syncBuffer{}
	h, w := newActionsHarness(t, nil, engine.Options{Audit: audit, Drains: engine.NewDrains(engine.NewMemoryDrainJournal())})

	for _, origin := range []engine.Origin{engine.OriginExternalAgent, engine.OriginBuiltInAgent} {
		if err := h.run(t, origin, "node.cordon", actionNodeRef("worker-1"), nil); !errors.Is(err, engine.ErrConfirmationUnavailable) {
			t.Errorf("%s cordon without a GUI: %v", origin, err)
		}
	}
	for _, origin := range []engine.Origin{engine.OriginGUI, engine.OriginCLI, engine.OriginExternalAgent, engine.OriginBuiltInAgent} {
		if err := h.run(t, origin, "node.drain", actionNodeRef("worker-1"), nil); !errors.Is(err, engine.ErrConfirmationUnavailable) {
			t.Errorf("%s drain without a GUI: %v", origin, err)
		}
	}
	if len(w.Calls()) != 0 {
		t.Fatalf("an unconfirmed command ran: %v", w.Calls())
	}

	c := &confirmations{answer: false}
	h.s.SetConfirmer(c)
	if err := h.run(t, engine.OriginBuiltInAgent, "node.drain", actionNodeRef("worker-1"), nil); !errors.Is(err, engine.ErrDeclined) {
		t.Errorf("declined drain: %v", err)
	}
	req := c.last(t)
	if req.Command != "node.drain" || req.Danger != engine.DangerDestructive || req.Origin != engine.OriginBuiltInAgent ||
		req.Context != "test" || len(req.Namespaces) != 0 || len(req.Objects) != 1 || req.Objects[0].Name != "worker-1" {
		t.Errorf("drain confirmation %+v", req)
	}
	if msg := req.Message(); !strings.Contains(msg, "nodes worker-1") || !strings.Contains(msg, "context test") || !strings.Contains(msg, "An agent asks") {
		t.Errorf("confirmation message %q", msg)
	}

	// Agent writes allowed for the context: Cordon runs without asking; Drain still asks.
	h.s.SetAgentWrites("test", true)
	asked := len(c.all())
	h.must(t, engine.OriginExternalAgent, "node.cordon", actionNodeRef("worker-1"), nil)
	if len(c.all()) != asked {
		t.Error("cordon asked although agent writes are allowed")
	}
	c.mu.Lock()
	c.answer = true
	c.mu.Unlock()
	h.must(t, engine.OriginExternalAgent, "node.drain", actionNodeRef("worker-1"), nil)
	if len(c.all()) != asked+1 || c.last(t).Danger != engine.DangerDestructive {
		t.Error("drain ran without the user's confirmation")
	}

	lines := audit.Lines()
	var phases []string
	for _, l := range lines {
		phases = append(phases, l.Command+":"+l.Phase)
		if l.Context != "test" {
			t.Errorf("audit line for %s names context %q", l.Command, l.Context)
		}
	}
	for _, want := range []string{"node.cordon:requested", "node.cordon:done", "node.drain:requested", "node.drain:done"} {
		if !slices.Contains(phases, want) {
			t.Errorf("audit lacks %s: %v", want, phases)
		}
	}
}

// TestDrainRecordsAndClearsItsJournalEntry: the journal holds the drain while it runs, with who
// started it and its settings, and drops it when the drain finished.
func TestDrainRecordsAndClearsItsJournalEntry(t *testing.T) {
	journal := engine.NewMemoryDrainJournal()
	h, w := newActionsHarness(t, nil, engine.Options{Drains: engine.NewDrains(journal), Confirmer: &confirmations{answer: true}})
	events := recordEvents(h.s)

	var during []engine.PendingDrain
	w.setDrainHook(func(_ context.Context, _ string, progress func(cluster.DrainProgress)) error {
		during, _ = journal.List()
		progress(cluster.DrainProgress{Step: cluster.DrainListed, Node: "worker-1", Total: 1})
		progress(cluster.DrainProgress{Step: cluster.DrainEvicted, Node: "worker-1", Pod: "shop/web-0", Done: 1, Total: 1})
		return nil
	})
	var res commands.DrainResult
	h.must(t, engine.OriginGUI, "node.drain", map[string]any{"ref": model.ObjectRef{Kind: "nodes", Name: "worker-1"}, "force": true, "gracePeriodSeconds": 30}, &res)
	if len(during) != 1 || during[0].Node != "worker-1" || during[0].Context != "test" || during[0].State != engine.DrainRunning ||
		during[0].StartedBy != engine.OriginGUI || during[0].Started.IsZero() || !during[0].Settings.Force ||
		during[0].Settings.IgnoreDaemonSets == nil || !*during[0].Settings.IgnoreDaemonSets {
		t.Errorf("journal while draining: %+v", during)
	}
	if after, _ := journal.List(); len(after) != 0 {
		t.Errorf("journal after the drain finished: %+v", after)
	}
	if res.Action != "drained" || res.Evicted != 1 || res.Ref.Name != "worker-1" {
		t.Errorf("result %+v", res)
	}
	opts := w.lastDrainOpts()
	if !opts.IgnoreDaemonSets || opts.DeleteEmptyDirData || !opts.Force || opts.GracePeriodSeconds != 30 || opts.Timeout != engine.DefaultDrainTimeout {
		t.Errorf("drain options %+v", opts)
	}
	if got := w.Calls(); !slices.Equal(got, []string{"drain worker-1", "cordon worker-1"}) {
		t.Errorf("writes %v", got)
	}
	events.waitEvent(t, engine.EventToast, "Node worker-1 drained: 1 pods evicted")
	if len(events.of(engine.EventDrains)) < 3 {
		t.Errorf("drain events: %+v", events.of(engine.EventDrains))
	}
}

// TestDrainDefaultsAndPrefill: the settings a dialog starts with, and the options an agent's
// drain without settings gets: DaemonSet pods left running, nothing that loses data, each pod's
// own grace period, five minutes.
func TestDrainDefaultsAndPrefill(t *testing.T) {
	h, w := newActionsHarness(t, nil, engine.Options{Drains: engine.NewDrains(engine.NewMemoryDrainJournal()), Confirmer: &confirmations{answer: true}})
	cmd, _ := h.s.Registry().Get("node.drain")
	raw, err := cmd.Prefill(h.s, mustJSON(t, actionNodeRef("worker-1")))
	if err != nil {
		t.Fatal(err)
	}
	var pre map[string]any
	if err := json.Unmarshal(raw, &pre); err != nil {
		t.Fatal(err)
	}
	if pre["ignoreDaemonSets"] != true || pre["timeoutSeconds"] != float64(300) || pre["force"] != nil || pre["deleteEmptyDirData"] != nil || pre["gracePeriodSeconds"] != nil {
		t.Errorf("prefill %s", raw)
	}
	h.must(t, engine.OriginCLI, "node.drain", actionNodeRef("worker-1"), nil)
	if opts := w.lastDrainOpts(); opts != (cluster.DrainOptions{IgnoreDaemonSets: true, GracePeriodSeconds: -1, Timeout: 5 * time.Minute}) {
		t.Errorf("default options %+v", opts)
	}
	h.must(t, engine.OriginCLI, "node.drain", map[string]any{"ref": model.ObjectRef{Kind: "nodes", Name: "worker-1"}, "ignoreDaemonSets": false, "deleteEmptyDirData": true, "timeoutSeconds": 60}, nil)
	if opts := w.lastDrainOpts(); opts != (cluster.DrainOptions{DeleteEmptyDirData: true, GracePeriodSeconds: -1, Timeout: time.Minute}) {
		t.Errorf("options %+v", opts)
	}
}

func TestDrainBadParams(t *testing.T) {
	h, w := newActionsHarness(t, nil, engine.Options{Drains: engine.NewDrains(engine.NewMemoryDrainJournal()), Confirmer: &confirmations{answer: true}})
	node := model.ObjectRef{Kind: "nodes", Name: "worker-1"}
	var perr *engine.ParamsError
	for _, params := range []map[string]any{
		{"ref": node, "gracePeriodSeconds": -1},
		{"ref": node, "timeoutSeconds": 70000},
		{"ref": node, "force": "yes"},
		{"ref": node, "evictEverything": true},
		{"node": "worker-1"},
	} {
		if err := h.run(t, engine.OriginGUI, "node.drain", params, nil); !errors.As(err, &perr) {
			t.Errorf("%v: %v, want a params error", params, err)
		}
	}
	var dis *engine.DisabledError
	if err := h.run(t, engine.OriginGUI, "node.drain", map[string]any{"ref": model.ObjectRef{Kind: "pods", Namespace: "shop", Name: "web-0"}}, nil); !errors.As(err, &dis) {
		t.Errorf("drain a pod: %v", err)
	}
	if len(w.Calls()) != 0 {
		t.Errorf("writes %v", w.Calls())
	}
}

func TestDrainNeedsAJournal(t *testing.T) {
	h, w := newActionsHarness(t, nil, engine.Options{Confirmer: &confirmations{answer: true}})
	var dis *engine.DisabledError
	if err := h.run(t, engine.OriginGUI, "node.drain", actionNodeRef("worker-1"), nil); !errors.As(err, &dis) || !strings.Contains(dis.Reason, "journal") {
		t.Errorf("drain without a journal: %v", err)
	}
	if _, err := h.s.PendingDrains(false); !errors.Is(err, engine.ErrNoDrainJournal) {
		t.Errorf("pending drains without a journal: %v", err)
	}
	if len(w.Calls()) != 0 {
		t.Errorf("writes %v", w.Calls())
	}
}

// TestDrainConfirmedForAnotherClusterDoesNotRun: the user switches clusters while the dialog is
// open; the drain neither runs nor leaves a journal entry.
func TestDrainConfirmedForAnotherClusterDoesNotRun(t *testing.T) {
	journal := engine.NewMemoryDrainJournal()
	h, w := newActionsHarness(t, nil, engine.Options{Drains: engine.NewDrains(journal)})
	c := &confirmations{answer: true}
	c.during = func(ctx context.Context) {
		if err := h.s.Connect(ctx, "other"); err != nil {
			t.Error(err)
		}
		warmActionKinds(t, h.s)
	}
	h.s.SetConfirmer(c)
	if err := h.run(t, engine.OriginGUI, "node.drain", actionNodeRef("worker-1"), nil); !errors.Is(err, engine.ErrWrongCluster) {
		t.Errorf("drain after a cluster switch: %v", err)
	}
	c.during = nil
	var dis *engine.DisabledError
	if err := h.run(t, engine.OriginGUI, "node.cordon", map[string]any{"ref": model.ObjectRef{Context: "test", Kind: "nodes", Name: "worker-1"}}, nil); !errors.As(err, &dis) || !strings.Contains(dis.Reason, "another cluster") {
		t.Errorf("cordon naming the old context: %v", err)
	}
	if len(w.Calls()) != 0 {
		t.Errorf("writes reached the other cluster: %v", w.Calls())
	}
	if entries, _ := journal.List(); len(entries) != 0 {
		t.Errorf("journal %+v", entries)
	}
}

// TestFailedDrainKeepsItsEntry: a drain that fails while Cogpit runs keeps its journal entry as
// failed and says that the node stays cordoned and partly drained; while it runs, a second drain,
// an uncordon and a dismiss of the node are refused.
func TestFailedDrainKeepsItsEntry(t *testing.T) {
	journal := engine.NewMemoryDrainJournal()
	h, w := newActionsHarness(t, nil, engine.Options{Drains: engine.NewDrains(journal), Confirmer: &confirmations{answer: true}})
	events := recordEvents(h.s)

	release := make(chan struct{})
	running := make(chan struct{})
	w.setDrainHook(func(_ context.Context, name string, progress func(cluster.DrainProgress)) error {
		progress(cluster.DrainProgress{Step: cluster.DrainListed, Node: name, Total: 2})
		progress(cluster.DrainProgress{Step: cluster.DrainBlocked, Node: name, Pod: "shop/db-0", Total: 2,
			Message: "the PodDisruptionBudget db-pdb allows no disruption now"})
		close(running)
		<-release
		return errors.New("drain node worker-1: timed out after 5m0s: error when evicting pods/\"db-0\" -n \"shop\": global timeout reached")
	})
	done := make(chan error, 1)
	go func() { done <- h.run(t, engine.OriginGUI, "node.drain", actionNodeRef("worker-1"), nil) }()
	<-running

	events.waitEvent(t, engine.EventToast, "waits: the PodDisruptionBudget db-pdb")
	var pending []engine.PendingDrainInfo
	h.must(t, engine.OriginExternalAgent, "node.pendingDrains", nil, &pending)
	if len(pending) != 1 || pending[0].Status != engine.DrainStatusRunning || pending[0].Progress == nil ||
		pending[0].Progress.Step != cluster.DrainBlocked || pending[0].NodeState != engine.DrainNodeCordoned {
		t.Errorf("pending while running: %+v", pending)
	}
	var dis *engine.DisabledError
	if err := h.run(t, engine.OriginGUI, "node.drain", actionNodeRef("worker-1"), nil); !errors.As(err, &dis) || !strings.Contains(dis.Reason, "running") {
		t.Errorf("a second drain: %v", err)
	}
	if err := h.run(t, engine.OriginGUI, "node.uncordon", actionNodeRef("worker-1"), nil); !errors.As(err, &dis) || !strings.Contains(dis.Reason, "running") {
		t.Errorf("uncordon during the drain: %v", err)
	}
	if err := h.run(t, engine.OriginGUI, "node.dismissDrain", actionNodeRef("worker-1"), nil); err == nil || !strings.Contains(err.Error(), "running") {
		t.Errorf("dismiss during the drain: %v", err)
	}

	close(release)
	err := <-done
	if err == nil || !strings.Contains(err.Error(), "stays cordoned and partly drained") || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("failed drain: %v", err)
	}
	events.waitEvent(t, engine.EventToast, "did not finish; the node stays cordoned")
	entries, _ := journal.List()
	if len(entries) != 1 || entries[0].State != engine.DrainFailed || !strings.Contains(entries[0].Error, "timed out") {
		t.Fatalf("journal after the failure: %+v", entries)
	}
	var failed []engine.PendingDrainInfo
	h.must(t, engine.OriginGUI, "node.pendingDrains", map[string]any{}, &failed)
	if len(failed) != 1 || failed[0].Status != engine.DrainStatusFailed || failed[0].Progress != nil {
		t.Errorf("pending after the failure: %+v", failed)
	}

	// Finish: the same command again, now succeeding, clears the entry.
	w.setDrainHook(nil)
	h.must(t, engine.OriginGUI, "node.drain", actionNodeRef("worker-1"), nil)
	if entries, _ := journal.List(); len(entries) != 0 {
		t.Errorf("journal after finishing: %+v", entries)
	}
}

// TestDrainPanicIsRecoveredAndRecorded: a panic inside the drain is the drain goroutine's boundary
// (D-11): an error, the journal entry kept as failed, the session alive.
func TestDrainPanicIsRecoveredAndRecorded(t *testing.T) {
	journal := engine.NewMemoryDrainJournal()
	h, w := newActionsHarness(t, nil, engine.Options{Drains: engine.NewDrains(journal), Confirmer: &confirmations{answer: true}, Logger: quietLogger})
	w.setDrainHook(func(context.Context, string, func(cluster.DrainProgress)) error { panic("library bug") })
	err := h.run(t, engine.OriginGUI, "node.drain", actionNodeRef("worker-1"), nil)
	if err == nil || !strings.Contains(err.Error(), "internal error") {
		t.Fatalf("err = %v", err)
	}
	if entries, _ := journal.List(); len(entries) != 1 || entries[0].State != engine.DrainFailed {
		t.Errorf("journal %+v", entries)
	}
	if h.s.Drains().Running("test", "worker-1") {
		t.Error("the panicked drain still counts as running")
	}
}
