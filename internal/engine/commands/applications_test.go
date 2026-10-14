package commands_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// appsConnector is the workload connector with the Applications view, as the cluster connector
// offers it next to the workload kinds.
type appsConnector struct{ workloadConnector }

func (c *appsConnector) Connect(ctx context.Context, name string) (*engine.Backend, error) {
	b, err := c.workloadConnector.Connect(ctx, name)
	if err != nil {
		return nil, err
	}
	b.Profiles = append(b.Profiles, kinds.Applications())
	return b, nil
}

// labelled sets a workload's app.kubernetes.io/instance label.
func labelled(t *testing.T, h *harness, kind, ns, name, instance string) {
	t.Helper()
	obj, err := h.s.Object(context.Background(), model.ObjectRef{Kind: kind, Namespace: ns, Name: name})
	if err != nil {
		t.Fatal(err)
	}
	obj = obj.DeepCopy()
	obj.SetLabels(map[string]string{kinds.LabelAppInstance: instance})
	p, _ := kinds.ByKey(kind)
	h.src.Set(p.GVR(), obj)
}

// TestApplicationLogsOpensOneMergedTab (A81): Logs of an application (Lens's one documented action,
// at Logs' place) opens one logs tab on the application, which merges its workloads' logs, and
// activates it when it is open; it is not offered for an application without a workload a tab can
// follow (a Cron Job's alone).
func TestApplicationLogsOpensOneMergedTab(t *testing.T) {
	src := seedWorkloads()
	h := &harness{src: src, writer: &fakeWriter{}}
	h.s = engine.New(engine.Options{Connector: &appsConnector{workloadConnector{actionsConnector{src: src, writer: h.writer}}}, SnapshotInterval: 10 * time.Millisecond})
	if err := commands.RegisterAll(h.s.Registry()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(h.s.Close)
	if err := h.s.Connect(context.Background(), "test"); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"deployments", "statefulsets", "cronjobs"} {
		if _, err := h.s.Objects(context.Background(), k, nil); err != nil {
			t.Fatal(err)
		}
	}
	labelled(t, h, "deployments", "shop", "web", "storefront")
	labelled(t, h, "statefulsets", "shop", "db", "storefront")
	labelled(t, h, "cronjobs", "ops", "backup", "backups")

	cmd, _ := h.s.Registry().Get("application.logs")
	if cmd.Order != 300 || cmd.Danger != engine.DangerRead || cmd.Label != "Logs" || len(cmd.Kinds) != 1 || cmd.Kinds[0] != kinds.ApplicationsKey {
		t.Errorf("application.logs: %+v", cmd)
	}
	storefront := actionRef(kinds.ApplicationsKey, "shop", "storefront")
	if _, err := h.s.Objects(context.Background(), kinds.ApplicationsKey, nil); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if !testkit.WaitFor(ctx, func() bool {
		app, err := h.s.Lookup(storefront)
		return err == nil && len(kinds.ApplicationWorkloads(app)) == 2
	}) {
		t.Fatal("the storefront application never held its two workloads")
	}
	if err := cmd.Enabled(h.s, cmd.ObjectParams(storefront, "")); err != nil {
		t.Fatalf("Enabled: %v", err)
	}
	var res commands.DockResult
	h.must(t, engine.OriginExternalAgent, "application.logs", map[string]any{"ref": storefront}, &res)
	if res.Tab.Title != "Logs: Application storefront" || !res.Created || res.Tab.Ref.Kind != kinds.ApplicationsKey || !res.Tab.Logs.ResourceNames {
		t.Errorf("tab %+v", res)
	}
	var again commands.DockResult
	h.must(t, engine.OriginGUI, "application.logs", map[string]any{"ref": storefront}, &again)
	if again.Created || again.Tab.ID != res.Tab.ID || len(h.s.UIState().Dock) != 1 {
		t.Errorf("a second Logs %+v, dock %d", again, len(h.s.UIState().Dock))
	}
	var src2 engine.LogSources
	h.must(t, engine.OriginExternalAgent, "logs.sources", map[string]any{"id": res.Tab.ID}, &src2)
	if strings.Join(src2.Workloads, ",") != "web,db" {
		t.Errorf("sources %+v", src2)
	}
	var narrowed engine.DockTab
	h.must(t, engine.OriginExternalAgent, "logs.setSource", map[string]any{"id": res.Tab.ID, "workload": "db"}, &narrowed)
	if narrowed.Logs.Workload != "db" {
		t.Errorf("narrowed %+v", narrowed.Logs)
	}

	backups := actionRef(kinds.ApplicationsKey, "ops", "backups")
	if !testkit.WaitFor(ctx, func() bool { _, err := h.s.Lookup(backups); return err == nil }) {
		t.Fatal("no backups application")
	}
	if err := cmd.Enabled(h.s, cmd.ObjectParams(backups, "")); !errors.Is(err, engine.ErrNotApplicable) {
		t.Errorf("Logs of an application of a Cron Job only: %v", err)
	}
}
