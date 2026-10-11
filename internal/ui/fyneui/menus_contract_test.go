package fyneui

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
)

// TestRegistryMenusAsLens checks the row menus and drawer toolbars the real registry yields
// against Lens's captured menus (2026-10-11), Show Details aside: Node: Shell, Cordon or Uncordon,
// Drain, Edit, Delete; Cron Job: Trigger, Suspend or Resume, Edit, Delete; Pod: Attach Pod, Shell,
// Evict, Share, Logs, Edit, Delete; Deployment: Scale, Restart, Logs. A pair such as Cordon and
// Uncordon are both listed here; the menu shows the one that applies (TestRowMenuNodes). Commands
// Cogpit has not built are named in missing, so that the test shows the gap instead of hiding it.
func TestRegistryMenusAsLens(t *testing.T) {
	s := engine.New(engine.Options{})
	defer s.Close()
	if err := commands.RegisterAll(s.Registry()); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		kind    string
		want    []string
		missing string
	}{
		{"nodes", []string{"node.cordon", "node.uncordon", "node.drain", "object.edit", "object.delete"}, "node.shell (100)"},
		{"cronjobs", []string{"cronjob.trigger", "cronjob.resume", "cronjob.suspend", "object.edit", "object.delete"}, ""},
		{"pods", []string{"pod.attach", "pod.shell", "pod.evict", "pod.logs", "object.edit", "object.delete"}, "pod.share is not a target"},
		{"deployments", []string{"deployment.scale", "deployment.restart", "object.edit", "object.delete"}, "workload.logs (300)"},
		{"statefulsets", []string{"statefulset.restart", "statefulset.scale", "object.edit", "object.delete"}, "workload.logs"},
		{"daemonsets", []string{"daemonset.restart", "object.edit", "object.delete"}, "workload.logs"},
		{"replicasets", []string{"replicaset.scale", "object.edit", "object.delete"}, "workload.logs"},
	} {
		var got []string
		for _, c := range s.Registry().Actions(tt.kind) {
			got = append(got, c.ID)
		}
		if strings.Join(got, ",") != strings.Join(tt.want, ",") {
			t.Errorf("%s: actions %v, want %v (not built: %s)", tt.kind, got, tt.want, tt.missing)
		}
	}
	// What each real command gets from the menus: Scale a dialog for its replicas, Drain one for its
	// options (both have a Prefill), Logs and Shell a submenu of the containers, the others nothing.
	for id, want := range map[string]string{
		"deployment.scale": "dialog:replicas", "statefulset.scale": "dialog:replicas", "replicaset.scale": "dialog:replicas",
		"daemonset.restart": "", "statefulset.restart": "",
		"pod.logs": "containers", "pod.shell": "containers", "pod.attach": "containers",
		"node.drain": "dialog:ignoreDaemonSets,deleteEmptyDirData,force,gracePeriodSeconds,timeoutSeconds",
		"pod.evict":  "", "node.cordon": "", "cronjob.trigger": "", "object.edit": "", "object.delete": "",
	} {
		c, ok := s.Registry().Get(id)
		if !ok {
			t.Errorf("no command %s", id)
			continue
		}
		got := ""
		switch {
		case len(c.DialogFields()) > 0:
			got = "dialog:" + strings.Join(c.DialogFields(), ",")
		case c.TakesContainer():
			got = "containers"
		}
		if got != want {
			t.Errorf("%s: %q, want %q", id, got, want)
		}
		// Every dialog field is one the generated form can show.
		test.NewTempApp(t)
		for _, name := range c.DialogFields() {
			if _, _, err := formField(name, c.Schema.Properties[name], nil, c.Requires(name)); err != nil {
				t.Errorf("%s: %v", id, err)
			}
		}
	}
}
