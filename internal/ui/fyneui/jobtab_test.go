package fyneui

import (
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
)

// reconcileJob is Reconcile with Source's job tab (A91) as the engine holds it while the source
// is done and the Kustomization is waited for; a third object's source failed.
func reconcileJob() engine.DockTab {
	git := model.ObjectRef{Context: "demo", Kind: "gitrepositories.source.toolkit.fluxcd.io", Namespace: "flux-system", Name: "flux-system"}
	ks := model.ObjectRef{Context: "demo", Kind: "kustomizations.kustomize.toolkit.fluxcd.io", Namespace: "flux-system", Name: "apps"}
	oci := model.ObjectRef{Context: "demo", Kind: "ocirepositories.source.toolkit.fluxcd.io", Namespace: "platform", Name: "platform"}
	infra := model.ObjectRef{Context: "demo", Kind: "kustomizations.kustomize.toolkit.fluxcd.io", Namespace: "flux-system", Name: "infra"}
	return engine.DockTab{
		ID: "job-1", Type: engine.DockJob, Title: "Reconcile with Source: 2 objects", Context: "demo",
		Job: &engine.JobTab{
			Command: "flux.reconcileWithSourceMany", State: engine.JobRunning, Started: fixedNow.Add(-40 * time.Second),
			Steps: []engine.JobStep{
				{Ref: &git, Text: "GitRepository flux-system/flux-system", State: engine.JobDone, Detail: "Ready: stored artifact for revision 'main@sha1:7a1e2c4'"},
				{Ref: &ks, Text: "Kustomization flux-system/apps", State: engine.JobRunning, Detail: "Waiting for its controller: Reconciling: Detecting drift for revision main@sha1:7a1e2c4"},
				{Ref: &oci, Text: "OCIRepository platform/platform", State: engine.JobFailed, Detail: "Not ready: failed to pull artifact: 401 Unauthorized"},
				{Ref: &infra, Text: "Kustomization flux-system/infra", State: engine.JobFailed, Detail: "Not asked to reconcile: its source OCIRepository platform/platform is not ready: failed to pull artifact: 401 Unauthorized"},
			},
		},
	}
}

// jobTabOf returns the dock's job tab of that id.
func jobTabOf(sh *shell, id string) *jobTab {
	if c, ok := sh.dock.contents[id]; ok {
		if j, ok := c.editor.(*jobTab); ok {
			return j
		}
	}
	return nil
}

// TestJobTabFollowsTheEngine (A91): a job tab shows the job's state and steps as the engine holds
// them, each in its state's colour, and follows them as they move, whoever moved them; the tab's
// close button closes it through dock.close, which stops the job.
func TestJobTabFollowsTheEngine(t *testing.T) {
	f := connectedFake(t, 20)
	f.ui.Dock, f.ui.ActiveDock = []engine.DockTab{reconcileJob()}, "job-1"
	sh, _, q := newTestShell(t, f, Options{})
	q.until(t, "the job tab", func() bool { return jobTabOf(sh, "job-1") != nil })
	j := jobTabOf(sh, "job-1")
	if j.state.text != "Running" || j.state.color != colorAccent || len(j.rows) != 4 {
		t.Fatalf("header %q, %d rows", j.state.text, len(j.rows))
	}
	for i, want := range []struct {
		text  string
		color string
	}{
		{"GitRepository flux-system/flux-system", string(colorMuted)},
		{"Kustomization flux-system/apps", string(colorAccent)},
		{"OCIRepository platform/platform", string(colorStatusError)},
		{"Kustomization flux-system/infra", string(colorStatusError)},
	} {
		if r := j.rows[i]; r.text.text != want.text || string(r.detail.color) != want.color {
			t.Errorf("row %d: %q in %s", i, r.text.text, r.detail.color)
		}
	}

	// The engine ends the job.
	f.update(func(u *engine.UIState) {
		job := u.Dock[0].Job
		job.Steps[1].State, job.Steps[1].Detail = engine.JobDone, "Ready: Applied revision: main@sha1:7a1e2c4"
		job.State, job.Ended, job.Outcome = engine.JobFailed, fixedNow, "2 of 4 objects reconciled and ready; the steps say why the others are not."
	})
	q.until(t, "the end", func() bool { return j.state.text == "Failed" })
	if j.outcome.text != "2 of 4 objects reconciled and ready; the steps say why the others are not." || j.outcome.color != colorStatusError ||
		j.rows[1].detail.text != "Ready: Applied revision: main@sha1:7a1e2c4" {
		t.Errorf("after the end: %q, row %q", j.outcome.text, j.rows[1].detail.text)
	}

	// Closing the tab asks the engine.
	b := sh.dock.tabRow.Objects[0].(*tabButton)
	b.close.OnTapped()
	q.until(t, "dock.close", func() bool { return len(f.callsOf("dock.close")) == 1 })
}

// TestRenderJobTab: Reconcile with Source's job tab in the dock while it waits for a
// Kustomization's controller, dark and light (A91).
func TestRenderJobTab(t *testing.T) {
	for _, mode := range []ThemeMode{ThemeDark, ThemeLight} {
		f := connectedFake(t, 200)
		f.ui.Dock, f.ui.ActiveDock = []engine.DockTab{reconcileJob()}, "job-1"
		sh, w, q := newTestShell(t, f, Options{Theme: mode})
		q.until(t, "the job tab", func() bool { return jobTabOf(sh, "job-1") != nil && sh.dock.open() })
		q.drain()
		savePNG(t, w, "job_tab_"+string(mode)+".png")
	}
}
