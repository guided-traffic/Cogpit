package commands

import (
	"context"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
)

// The notification log's commands (engine/notifications.go, ui-parity › Appearance: "Notifications
// in the title bar"): what the user was told, for the title bar's bell and for agents. Reading it
// is a read of Cogpit's own state, marking and clearing are UI operations (D-2): none reaches a
// cluster. A message the window keeps to itself is withheld from every caller but the window.

type notificationsReadParams struct {
	IDs []uint64 `json:"ids,omitempty" jsonschema:"ids of the notifications to mark read, as notifications_list returns them; empty marks every one read"`
}

func registerNotifications(g *registrar) {
	add(g, engine.Spec[struct{}]{
		ID: "notifications.list", Label: "Notifications", Danger: engine.DangerRead,
		Description: "Lists the notifications the user was shown as toasts in this session, newest first, at most " +
			"100: each with its id, level (info, warning, error), message, time, and whether the user has read it (seen " +
			"it in the title bar's notification list or closed its toast), with the totals the bell's badge shows. A " +
			"message the window keeps to itself (its own toasts, such as a command's error, and a Helm action's error) " +
			"reads \"" + model.Withheld + "\" here. Nothing is sent to a cluster.",
		Run: func(ctx context.Context, s *engine.Session, _ struct{}) (any, error) {
			return s.Notifications(engine.OriginOf(ctx) == engine.OriginGUI), nil
		},
	})

	add(g, engine.Spec[notificationsReadParams]{
		ID: "notifications.markRead", Label: "Mark Notifications Read", Danger: engine.DangerUI,
		Description: "Marks notifications read, as opening the title bar's notification list does: the ones named by " +
			"ids, or every one. The bell's badge counts the unread ones. Returns the new UI state, whose notifications " +
			"field sums the log up.",
		Run: func(_ context.Context, s *engine.Session, p notificationsReadParams) (any, error) {
			return s.MarkNotificationsRead(p.IDs), nil
		},
	})

	add(g, engine.Spec[struct{}]{
		ID: "notifications.clear", Label: "Clear Notifications", Danger: engine.DangerUI,
		Description: "Empties the notification list, as its \"Clear all\" does in Cogpit's window. The audit log keeps " +
			"what agents did either way. Returns the new UI state.",
		Run: func(_ context.Context, s *engine.Session, _ struct{}) (any, error) {
			return s.ClearNotifications(), nil
		},
	})
}

// windowToast emits a toast whose text the notification log keeps for the window: one that quotes
// what the cluster returned to a Helm action, which a chart's templates can fill from a Secret
// (H-31).
func windowToast(s *engine.Session, level, msg string) {
	s.Emit(engine.Event{Type: engine.EventToast, Level: level, Message: msg, WindowOnly: true})
}
