package commands

import (
	"context"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// The notification log's commands (engine/notifications.go, ui-parity › Appearance: "Notifications
// in the title bar"): what the user was told, for the title bar's bell and for agents. Reading it
// is a read of Cogpit's own state, marking and clearing are the window's (A18): none reaches a
// cluster. A message the window raised itself, or one quoting a Helm action's error, reaches every
// other caller filtered: the cached Secrets' values replaced, the action's inline values removed
// (A19).

type notificationsReadParams struct {
	IDs []uint64 `json:"ids,omitempty" jsonschema:"ids of the notifications to mark read, as notifications_list returns them; empty marks every one read"`
}

func registerNotifications(g *registrar) {
	add(g, engine.Spec[struct{}]{
		ID: "notifications.list", Label: "Notifications", Danger: engine.DangerRead,
		Description: "Lists the notifications the user was shown as toasts in this session, newest first, at most " +
			"100: each with its id, level (info, warning, error), message, time, and whether the user has read it (seen " +
			"it in the title bar's notification list or closed its toast), with the totals the bell's badge shows. A " +
			"message the window raised itself (such as a command's error) or one quoting a Helm action's error reads " +
			"here with the values of the Secrets Cogpit's cache holds replaced by «secret <namespace>/<name>/<key>» and " +
			"the action's inline values by «inline value». Only the user marks them read or clears them. Nothing is sent " +
			"to a cluster." + privateReads,
		Run: func(ctx context.Context, s *engine.Session, _ struct{}) (any, error) {
			return s.Notifications(engine.ReadsLikeUser(ctx)), nil // the window's, in full, to a private turn too (H-94)
		},
	})

	add(g, engine.Spec[notificationsReadParams]{
		ID: "notifications.markRead", Label: "Mark Notifications Read", Danger: engine.DangerUI,
		GUIOnly: engine.ErrNotificationsGUIOnly, // A18
		Description: "Marks notifications read, as opening the title bar's notification list does: the ones named by " +
			"ids, or every one. The bell's badge counts the unread ones. Cogpit's window only: agents and the CLI are " +
			"refused. Returns the new UI state, whose notifications field sums the log up.",
		Run: func(_ context.Context, s *engine.Session, p notificationsReadParams) (any, error) {
			return s.MarkNotificationsRead(p.IDs), nil
		},
	})

	add(g, engine.Spec[struct{}]{
		ID: "notifications.clear", Label: "Clear Notifications", Danger: engine.DangerUI,
		GUIOnly: engine.ErrNotificationsGUIOnly, // A18
		Description: "Empties the notification list, as its \"Clear all\" does in Cogpit's window. Cogpit's window " +
			"only: agents and the CLI are refused. Returns the new UI state.",
		Run: func(_ context.Context, s *engine.Session, _ struct{}) (any, error) {
			return s.ClearNotifications(), nil
		},
	})
}

// windowToast emits a toast that quotes what the cluster returned to a Helm action, which a chart's
// templates can fill from a Secret (H-31): the window's in full, agents' filtered and without the
// action's inline values (A19, engine.EmitWindowToast).
func windowToast(s *engine.Session, level, msg, inline string) {
	s.EmitWindowToast(level, msg, inline)
}
