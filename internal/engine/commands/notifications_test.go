package commands_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
)

// The notification commands as agents and the window dispatch them: an agent reads what the user
// was told, the window's own messages withheld; the window reads every text; marking read and
// clearing run for an agent without the user's acceptance (D-2: UI commands) and are audited.
func TestNotificationCommands(t *testing.T) {
	audit := &syncBuffer{}
	h := newHarness(t, seed(), engine.Options{Audit: audit})
	agent, gui := engine.OriginExternalAgent, engine.OriginGUI
	h.s.Emit(engine.Event{Type: engine.EventToast, Level: engine.LevelInfo, Message: "Created services/shop/web"})
	h.s.RecordWindowToast(engine.LevelError, "pods.delete: pods \"web-0\" is forbidden")
	h.s.Emit(engine.Event{Type: engine.EventToast, Level: engine.LevelWarning, Message: "Release web was not changed: quoted", WindowOnly: true})

	var forAgent, forWindow engine.NotificationList
	h.must(t, agent, "notifications.list", struct{}{}, &forAgent)
	h.must(t, gui, "notifications.list", struct{}{}, &forWindow)
	agentText := func(l engine.NotificationList) []string {
		var out []string
		for _, n := range l.Notifications {
			out = append(out, n.Level+": "+n.Message)
		}
		return out
	}
	if got, want := agentText(forAgent), []string{"warning: " + model.Withheld, "error: " + model.Withheld, "info: Created services/shop/web"}; !slices.Equal(got, want) {
		t.Errorf("the agent reads %q, want %q", got, want)
	}
	if got := agentText(forWindow); got[1] != "error: pods.delete: pods \"web-0\" is forbidden" || got[0] != "warning: Release web was not changed: quoted" {
		t.Errorf("the window reads %q", got)
	}
	if forAgent.Unread != 3 || forAgent.UnreadLevel != engine.LevelError || forAgent.Total != 3 {
		t.Errorf("the summary: %+v", forAgent.NotificationSummary)
	}

	var u engine.UIState
	h.must(t, agent, "notifications.markRead", map[string]any{"ids": []uint64{forAgent.Notifications[1].ID}}, &u)
	if u.Notifications.Unread != 2 || u.Notifications.UnreadLevel != engine.LevelWarning {
		t.Errorf("after an agent marked the error read: %+v", u.Notifications)
	}
	var all engine.UIState
	h.must(t, gui, "notifications.markRead", struct{}{}, &all)
	if all.Notifications.Unread != 0 || all.Notifications.Total != 3 {
		t.Errorf("after the window marked every one read: %+v", all.Notifications)
	}
	var cleared engine.UIState
	h.must(t, agent, "notifications.clear", struct{}{}, &cleared)
	if cleared.Notifications != (engine.NotificationSummary{}) || len(h.s.Notifications(true).Notifications) != 0 {
		t.Errorf("after an agent cleared the list: %+v", cleared.Notifications)
	}
	var done []string
	for _, e := range audit.Lines() {
		if e.Phase == "done" && e.OK {
			done = append(done, e.Command)
		}
	}
	if want := []string{"notifications.list", "notifications.markRead", "notifications.clear"}; !slices.Equal(done, want) {
		t.Errorf("audited %q, want %q", done, want)
	}
}

// A Helm action's error the toast quotes stays in the window's notification list (H-31): an agent
// whose install failed reads neither the error nor the toast's text.
func TestHelmErrorToastStaysInTheWindow(t *testing.T) {
	h, fi := installHarness(t)
	h.answer = true
	params := map[string]any{"repository": "test", "chart": "web", "version": "0.2.0", "values": "a: b\n", "namespace": "apps", "name": "web"}
	fi.err = errors.New("helm install apps/web: template: web/templates/cm.yaml: name LOOKED-UP-SECRET-DATA is not valid")
	if _, err := h.dispatch(engine.OriginExternalAgent, "helm.chart.install", params); err == nil {
		t.Fatal("the install did not fail")
	}
	res, err := h.dispatch(engine.OriginExternalAgent, "notifications.list", struct{}{})
	if err != nil {
		t.Fatal(err)
	}
	agent := res.(engine.NotificationList)
	withheld := 0
	for _, n := range agent.Notifications {
		if strings.Contains(n.Message, "LOOKED-UP-SECRET-DATA") {
			t.Errorf("an agent reads the Helm error: %+v", n)
		}
		if n.Message == model.Withheld && n.Level == engine.LevelWarning {
			withheld++
		}
	}
	window := h.s.Notifications(true)
	shown := slices.ContainsFunc(window.Notifications, func(n engine.Notification) bool {
		return n.WindowOnly && strings.Contains(n.Message, "LOOKED-UP-SECRET-DATA")
	})
	if withheld != 1 || !shown {
		t.Errorf("withheld %d for the agent; the window's list %+v", withheld, window.Notifications)
	}
}
