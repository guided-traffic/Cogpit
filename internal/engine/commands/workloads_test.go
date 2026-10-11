package commands_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/model"
)

func actionRef(kind, namespace, name string) model.ObjectRef {
	return model.ObjectRef{Kind: kind, Namespace: namespace, Name: name}
}

func scaleTo(r model.ObjectRef, replicas int) map[string]any {
	return map[string]any{"ref": r, "replicas": replicas}
}

// TestScalePrefillsTheCurrentCount: a scale dialog starts from spec.replicas in the cache (1, the
// API server's default, when it is not set).
func TestScalePrefillsTheCurrentCount(t *testing.T) {
	h, _ := newActionsHarness(t, nil, engine.Options{})
	for _, tc := range []struct {
		id   string
		ref  model.ObjectRef
		want uint16
	}{
		{"deployment.scale", actionRef("deployments", "shop", "web"), 3},
		{"deployment.scale", actionRef("deployments", "shop", "defaulted"), 1},
		{"statefulset.scale", actionRef("statefulsets", "shop", "db"), 2},
	} {
		cmd, _ := h.s.Registry().Get(tc.id)
		raw, err := cmd.Prefill(h.s, mustJSON(t, map[string]any{"ref": tc.ref}))
		if err != nil {
			t.Fatalf("%s %s: %v", tc.id, tc.ref.Name, err)
		}
		var p struct {
			Ref      model.ObjectRef `json:"ref"`
			Replicas *uint16         `json:"replicas"`
		}
		if err := json.Unmarshal(raw, &p); err != nil || p.Replicas == nil || *p.Replicas != tc.want || p.Ref.Name != tc.ref.Name {
			t.Errorf("%s %s: prefill %s (%v)", tc.id, tc.ref.Name, raw, err)
		}
		// A count the caller gave is kept.
		raw, _ = cmd.Prefill(h.s, mustJSON(t, scaleTo(tc.ref, 7)))
		if !strings.Contains(string(raw), `"replicas":7`) {
			t.Errorf("prefill replaced the given count: %s", raw)
		}
	}
	cmd, _ := h.s.Registry().Get("deployment.scale")
	if _, err := cmd.Prefill(h.s, mustJSON(t, map[string]any{"ref": actionRef("deployments", "shop", "gone")})); err == nil {
		t.Error("prefill of a missing deployment")
	}
}

// TestScaleToZeroIsConfirmedAsDestructive: Scale is mutating, but scaling to zero escalates to
// destructive for every origin (privilege-footprint.md), and the audit log records it.
func TestScaleToZeroIsConfirmedAsDestructive(t *testing.T) {
	audit := &syncBuffer{}
	h, w := newActionsHarness(t, nil, engine.Options{Audit: audit})
	web, db := actionRef("deployments", "shop", "web"), actionRef("statefulsets", "shop", "db")

	for _, id := range []string{"deployment.scale", "statefulset.scale"} {
		if c, _ := h.s.Registry().Get(id); !c.Escalates || c.Danger != engine.DangerMutating || c.Order == 0 {
			t.Errorf("%s: %+v", id, c)
		}
	}
	// From the GUI: scaling up runs at once; scaling to zero needs the confirmation.
	h.must(t, engine.OriginGUI, "deployment.scale", scaleTo(web, 5), nil)
	if err := h.run(t, engine.OriginGUI, "statefulset.scale", scaleTo(db, 0), nil); !errors.Is(err, engine.ErrConfirmationUnavailable) {
		t.Errorf("scale to zero without a GUI: %v", err)
	}
	c := &confirmations{answer: true}
	h.s.SetConfirmer(c)
	h.must(t, engine.OriginGUI, "statefulset.scale", scaleTo(db, 0), nil)
	if req := c.last(t); req.Danger != engine.DangerDestructive || req.Command != "statefulset.scale" ||
		!slices.Equal(req.Namespaces, []string{"shop"}) || req.Objects[0].Kind != "statefulsets" || req.Context != "test" {
		t.Errorf("scale-to-zero confirmation %+v", req)
	}

	// Agents with writes allowed: scaling runs without asking, scaling to zero still asks.
	h.s.SetAgentWrites("test", true)
	asked := len(c.all())
	for _, origin := range []engine.Origin{engine.OriginExternalAgent, engine.OriginBuiltInAgent} {
		h.must(t, origin, "deployment.scale", scaleTo(web, 2), nil)
		if len(c.all()) != asked {
			t.Errorf("%s: scaling up asked although agent writes are allowed", origin)
		}
		h.must(t, origin, "deployment.scale", scaleTo(web, 0), nil)
		asked++
		if len(c.all()) != asked || c.last(t).Origin != origin || c.last(t).Danger != engine.DangerDestructive {
			t.Errorf("%s: scale to zero not confirmed as destructive: %+v", origin, c.all())
		}
	}
	want := []string{
		"scale deployments shop/web 5", "scale statefulsets shop/db 0",
		"scale deployments shop/web 2", "scale deployments shop/web 0",
		"scale deployments shop/web 2", "scale deployments shop/web 0",
	}
	if got := w.Calls(); !slices.Equal(got, want) {
		t.Errorf("writes %v", got)
	}
	var zeros int
	for _, l := range audit.Lines() {
		if l.Command == "deployment.scale" && l.Phase == "done" && strings.Contains(string(l.Params), `"replicas":0`) {
			zeros++
		}
	}
	if zeros != 2 {
		t.Errorf("audit lines of agent scale-to-zero: %d, want 2: %+v", zeros, audit.Lines())
	}
}

func TestScaleBadParams(t *testing.T) {
	h, w := newActionsHarness(t, nil, engine.Options{})
	web := actionRef("deployments", "shop", "web")
	var perr *engine.ParamsError
	for _, params := range []map[string]any{
		{"ref": web, "replicas": -1},
		{"ref": web, "replicas": 65536},
		{"ref": web, "replicas": 1.5},
		{"ref": web, "replicas": "3"},
		{"ref": web, "replicas": 3, "force": true},
	} {
		if err := h.run(t, engine.OriginGUI, "deployment.scale", params, nil); !errors.As(err, &perr) {
			t.Errorf("%v: %v, want a params error", params, err)
		}
	}
	if err := h.run(t, engine.OriginGUI, "deployment.scale", map[string]any{"ref": web}, nil); err == nil || !strings.Contains(err.Error(), "replicas is required") {
		t.Errorf("scale without a count: %v", err)
	}
	var dis *engine.DisabledError
	if err := h.run(t, engine.OriginGUI, "deployment.scale", scaleTo(actionRef("statefulsets", "shop", "db"), 1), nil); !errors.As(err, &dis) || !strings.Contains(dis.Reason, "deployments") {
		t.Errorf("deployment.scale on a stateful set: %v", err)
	}
	if err := h.run(t, engine.OriginGUI, "statefulset.scale", scaleTo(actionRef("statefulsets", "shop", "gone"), 1), nil); !errors.As(err, &dis) || !strings.Contains(dis.Reason, "not found") {
		t.Errorf("scale a missing stateful set: %v", err)
	}
	if len(w.Calls()) != 0 {
		t.Errorf("writes %v", w.Calls())
	}
	cmd, _ := h.s.Registry().Get("deployment.scale")
	if !cmd.Applicable(h.s, mustJSON(t, map[string]any{"ref": web})) || cmd.Enabled(h.s, mustJSON(t, map[string]any{"ref": web})) != nil {
		t.Error("Scale must be enabled for a row without a count: the dialog asks for it")
	}
}

// TestSuspendAndResumeTakeEachOthersPlace (captured: Resume takes Suspend's place on a suspended
// cron job) and Trigger, which creates a Job from the cached cron job.
func TestCronJobCommands(t *testing.T) {
	h, w := newActionsHarness(t, nil, engine.Options{})
	reg := h.s.Registry()
	suspend, _ := reg.Get("cronjob.suspend")
	resume, _ := reg.Get("cronjob.resume")
	backup := mustJSON(t, map[string]any{"ref": actionRef("cronjobs", "ops", "backup")})
	nightly := mustJSON(t, map[string]any{"ref": actionRef("cronjobs", "ops", "nightly")}) // suspended
	if !suspend.Applicable(h.s, backup) || suspend.Applicable(h.s, nightly) || resume.Applicable(h.s, backup) || !resume.Applicable(h.s, nightly) {
		t.Error("Suspend applies to a running cron job only, Resume to a suspended one only")
	}
	if suspend.Order != 200 || resume.Order != 200 {
		t.Errorf("orders %d, %d", suspend.Order, resume.Order)
	}

	var res commands.TriggerResult
	h.must(t, engine.OriginGUI, "cronjob.trigger", map[string]any{"ref": actionRef("cronjobs", "ops", "nightly")}, &res)
	if res.Job != (model.ObjectRef{Context: "test", Kind: "jobs", Namespace: "ops", Name: "nightly-manual-x7k2p"}) || res.Action != "triggered" {
		t.Errorf("trigger result %+v", res)
	}
	h.must(t, engine.OriginGUI, "cronjob.suspend", map[string]any{"ref": actionRef("cronjobs", "ops", "backup")}, nil)
	h.must(t, engine.OriginGUI, "cronjob.resume", map[string]any{"ref": actionRef("cronjobs", "ops", "nightly")}, nil)
	want := []string{"trigger ops/nightly uid-ops-nightly", "suspend ops/backup true", "suspend ops/nightly false"}
	if got := w.Calls(); !slices.Equal(got, want) {
		t.Errorf("writes %v", got)
	}

	var dis *engine.DisabledError
	if err := h.run(t, engine.OriginGUI, "cronjob.resume", map[string]any{"ref": actionRef("cronjobs", "ops", "backup")}, nil); !errors.As(err, &dis) || !strings.Contains(dis.Reason, "not suspended") {
		t.Errorf("resume a running cron job: %v", err)
	}
	// From an agent, Trigger is a mutating command: the user accepts it first.
	if err := h.run(t, engine.OriginExternalAgent, "cronjob.trigger", map[string]any{"ref": actionRef("cronjobs", "ops", "backup")}, nil); !errors.Is(err, engine.ErrConfirmationUnavailable) {
		t.Errorf("agent trigger without a GUI: %v", err)
	}
	c := &confirmations{answer: true}
	h.s.SetConfirmer(c)
	h.must(t, engine.OriginBuiltInAgent, "cronjob.trigger", map[string]any{"ref": actionRef("cronjobs", "ops", "backup")}, nil)
	if req := c.last(t); req.Danger != engine.DangerMutating || req.Command != "cronjob.trigger" || req.Objects[0].Name != "backup" {
		t.Errorf("trigger confirmation %+v", req)
	}
}

// TestEvict: destructive for every origin; the cached pod's UID is the precondition; a
// PodDisruptionBudget's refusal reaches the caller as an error naming the budget.
func TestEvict(t *testing.T) {
	h, w := newActionsHarness(t, nil, engine.Options{})
	pod := map[string]any{"ref": actionRef("pods", "shop", "web-0")}
	for _, origin := range []engine.Origin{engine.OriginGUI, engine.OriginCLI, engine.OriginExternalAgent, engine.OriginBuiltInAgent} {
		if err := h.run(t, origin, "pod.evict", pod, nil); !errors.Is(err, engine.ErrConfirmationUnavailable) {
			t.Errorf("%s evict without a GUI: %v", origin, err)
		}
	}
	c := &confirmations{answer: true}
	h.s.SetConfirmer(c)
	var res commands.ActionResult
	h.must(t, engine.OriginGUI, "pod.evict", pod, &res)
	if req := c.last(t); req.Danger != engine.DangerDestructive || !slices.Equal(req.Namespaces, []string{"shop"}) {
		t.Errorf("evict confirmation %+v", req)
	}
	if got := w.Calls(); !slices.Equal(got, []string{"evict shop/web-0 uid-shop-web-0"}) || res.Action != "evicted" {
		t.Errorf("writes %v, result %+v", got, res)
	}

	w.mu.Lock()
	w.evictErr = fmt.Errorf("evict pod shop/web-0: %w", &cluster.EvictionBlockedError{Pod: "shop/web-0", Budget: "web-pdb",
		Detail: "The disruption budget web-pdb needs 2 healthy pods and has 2 currently"})
	w.mu.Unlock()
	err := h.run(t, engine.OriginExternalAgent, "pod.evict", pod, nil)
	var blocked *cluster.EvictionBlockedError
	if !errors.As(err, &blocked) || blocked.Budget != "web-pdb" || !strings.Contains(err.Error(), "PodDisruptionBudget web-pdb") {
		t.Errorf("blocked eviction: %v", err)
	}
	var dis *engine.DisabledError
	if err := h.run(t, engine.OriginGUI, "pod.evict", map[string]any{"ref": actionRef("deployments", "shop", "web")}, nil); !errors.As(err, &dis) {
		t.Errorf("evict a deployment: %v", err)
	}
}

// TestActionOrders: the captured and documented positions in the row menu and the drawer.
func TestActionOrders(t *testing.T) {
	r := engine.NewRegistry()
	if err := commands.RegisterAll(r); err != nil {
		t.Fatal(err)
	}
	for id, want := range map[string]int{
		"node.cordon": 200, "node.uncordon": 200, "node.drain": 300, "deployment.scale": 100, "statefulset.scale": 200,
		"cronjob.trigger": 100, "cronjob.suspend": 200, "cronjob.resume": 200, "pod.evict": 300,
		"node.pendingDrains": 0, "node.dismissDrain": 0,
	} {
		c, ok := r.Get(id)
		if !ok || c.Order != want {
			t.Errorf("%s: order %v, want %d", id, c, want)
		}
	}
}

// FuzzNodeAndWorkloadCommands feeds the node and workload commands adversarial parameters from
// every origin, on a cluster that serves their kinds: errors, never a panic.
func FuzzNodeAndWorkloadCommands(f *testing.F) {
	ids := []string{"node.cordon", "node.uncordon", "node.drain", "node.pendingDrains", "node.dismissDrain",
		"deployment.scale", "statefulset.scale", "cronjob.trigger", "cronjob.suspend", "cronjob.resume", "pod.evict"}
	seeds := []string{
		`{}`, `null`, ``, `[]`, `{"ref":{"kind":"nodes","name":"worker-1"}}`, `{"ref":{"kind":"nodes","name":"worker-2"}}`,
		`{"ref":{"kind":"nodes","name":"worker-1"},"ignoreDaemonSets":false,"force":true,"gracePeriodSeconds":65535,"timeoutSeconds":1}`,
		`{"ref":{"kind":"nodes","name":"worker-1"},"ignoreDaemonSets":null}`, `{"ref":{"kind":"nodes","namespace":"x","name":"worker-1"}}`,
		`{"ref":{"kind":"deployments","namespace":"shop","name":"web"},"replicas":0}`,
		`{"ref":{"kind":"statefulsets","namespace":"shop","name":"db"},"replicas":65535}`,
		`{"ref":{"kind":"statefulsets","namespace":"shop","name":"db"},"replicas":null}`,
		`{"ref":{"kind":"cronjobs","namespace":"ops","name":"backup"}}`, `{"ref":{"kind":"cronjobs","name":"backup"}}`,
		`{"ref":{"kind":"pods","namespace":"shop","name":"web-0"}}`, `{"all":true}`, `{"all":"yes"}`,
		`{"ref":{"context":"other","kind":"nodes","name":"worker-1"}}`, `{"ref":{"kind":"nodes","name":""}}`,
	}
	for i := range ids {
		for j, s := range seeds {
			if (i+j)%3 == 0 {
				f.Add(uint8(i), []byte(s), uint8(j))
			}
		}
	}
	catcher := &panicCatcher{}
	h, _ := newActionsHarness(f, nil, engine.Options{
		Logger: slog.New(catcher), Audit: io.Discard, Drains: engine.NewDrains(engine.NewMemoryDrainJournal()),
		Confirmer: engine.ConfirmFunc(func(context.Context, engine.ConfirmRequest) (bool, error) { return true, nil }),
	})
	origins := []engine.Origin{engine.OriginGUI, engine.OriginCLI, engine.OriginExternalAgent, engine.OriginBuiltInAgent}
	f.Fuzz(func(t *testing.T, idx uint8, params []byte, origin uint8) {
		id := ids[int(idx)%len(ids)]
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		res, err := h.s.Dispatch(ctx, engine.Invocation{ID: id, Params: params, Origin: origins[int(origin)%len(origins)]})
		if msgs := catcher.take(); len(msgs) > 0 {
			t.Fatalf("%s %q: internal error: %v", id, params, msgs)
		}
		if err == nil && res == nil {
			t.Fatalf("%s %q: no result and no error", id, params)
		}
	})
}
