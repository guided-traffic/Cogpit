package fyneui

import (
	"fmt"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
)

// drainJobBudget is what a drain's pod step says while a PodDisruptionBudget holds its eviction.
const drainJobBudget = "Waiting: the PodDisruptionBudget db-pdb allows no disruption now " +
	"(The disruption budget db-pdb needs 2 healthy pods and has 2 currently); retrying"

// drainJobTab is a drain of worker-2 as a job (D-63) while it runs: the node cordoned with its
// DaemonSet pod left running, three pods evicted, one whose eviction was accepted, one a budget
// holds, two not asked for yet.
func drainJobTab() engine.DockTab {
	node := model.ObjectRef{Context: "kind-cogpit", Kind: "nodes", Name: "worker-2"}
	steps := []engine.JobStep{{Ref: &node, Text: "Node worker-2", State: engine.JobDone,
		Detail: "Cordoned; ignoring DaemonSet-managed Pods: kube-system/kube-proxy-7xk2p"}}
	pods := []struct {
		name   string
		state  engine.JobState
		detail string
	}{
		{"web-7d9f-0", engine.JobDone, "Evicted"},
		{"web-7d9f-1", engine.JobDone, "Evicted"},
		{"web-7d9f-2", engine.JobDone, "Evicted"},
		{"cache-0", engine.JobRunning, "Evicted; waiting for the pod to end"},
		{"db-0", engine.JobRunning, drainJobBudget},
		{"db-1", engine.JobPending, ""},
		{"worker-5c8b-4", engine.JobPending, ""},
	}
	for _, p := range pods {
		ref := model.ObjectRef{Context: "kind-cogpit", Kind: "pods", Namespace: "shop", Name: p.name}
		steps = append(steps, engine.JobStep{Ref: &ref, Text: "Pod shop/" + p.name, State: p.state, Detail: p.detail})
	}
	return engine.DockTab{
		ID: "job-3", Type: engine.DockJob, Title: "Drain: node worker-2", Context: "kind-cogpit",
		Job: &engine.JobTab{Command: "node.drain", State: engine.JobRunning, Started: fixedNow.Add(-75 * time.Second),
			Steps: steps, Counts: engine.CountJobSteps(steps)},
	}
}

// withDrainJob puts the drain's job tab into the dock and the drain, running in this window, into
// node.pendingDrains, as the engine holds them while the drain runs.
func withDrainJob(d *drainEngine) {
	d.ui.Dock, d.ui.ActiveDock = []engine.DockTab{drainJobTab()}, "job-3"
	yes := true
	d.pending = []engine.PendingDrainInfo{{
		PendingDrain: engine.PendingDrain{Context: "kind-cogpit", Node: "worker-2", Started: fixedNow.Add(-75 * time.Second),
			StartedBy: engine.OriginGUI, State: engine.DrainRunning, Settings: engine.DrainSettings{IgnoreDaemonSets: &yes, TimeoutSeconds: 300}},
		Status: engine.DrainStatusRunning, NodeState: engine.DrainNodeCordoned, Job: "job-3",
	}}
}

// TestDrainJobTabWithoutABanner (D-63): a drain running in this window shows in its job tab only:
// the steps with the budget's wait in the pod's detail, the progress and Cancel in the header, and
// no row in the drain banner, which keeps the drains that wait for the user.
func TestDrainJobTabWithoutABanner(t *testing.T) {
	sh, _, q, d := newDrainTestShell(t, ThemeDark, withDrainJob)
	q.until(t, "the job tab", func() bool { return jobTabOf(sh, "job-3") != nil })
	// With a stopped drain beside it, the banner shows that one only.
	stopped := fixtureDrains()[2]
	d.set(append(d.pending, stopped), nil)
	q.until(t, "the stopped drain alone", func() bool {
		return sh.drains.root.Visible() && len(sh.drains.drains) == 1 && sh.drains.drains[0].Node == "worker-4"
	})
	d.set(d.pending[:1], nil)
	q.until(t, "no banner", func() bool { return !sh.drains.root.Visible() && len(sh.drains.drains) == 0 })
	j := jobTabOf(sh, "job-3")
	if len(j.rows) != 8 || j.outcome.text != "4 of 8 done" || !j.cancel.Visible() {
		t.Fatalf("job tab: %d rows, %q, cancel %v", len(j.rows), j.outcome.text, j.cancel.Visible())
	}
	if r := j.rows[5]; r.text.text != "Pod shop/db-0" || r.detail.text != drainJobBudget || r.detail.color != colorAccent {
		t.Errorf("the blocked pod's row: %q %q in %s", r.text.text, r.detail.text, r.detail.color)
	}
	if r := j.rows[0]; r.text.text != "Node worker-2" || r.state != engine.JobDone {
		t.Errorf("the cordon's row: %q %s", r.text.text, r.state)
	}
}

// TestRenderJobTabDrain: a drain as a job in the dock while a PodDisruptionBudget holds a pod's
// eviction, no drain banner above the view, dark and light (D-63).
func TestRenderJobTabDrain(t *testing.T) {
	for _, mode := range []ThemeMode{ThemeDark, ThemeLight} {
		t.Run(string(mode), func(t *testing.T) {
			sh, w, q, d := newDrainTestShell(t, mode, withDrainJob)
			q.until(t, "the job tab", func() bool { return jobTabOf(sh, "job-3") != nil && sh.dock.open() })
			d.set(d.pending, nil) // the banner keeps it out (TestDrainJobTabWithoutABanner)
			q.drain()
			savePNG(t, w, fmt.Sprintf("job_tab_drain_%s.png", mode))
		})
	}
}
