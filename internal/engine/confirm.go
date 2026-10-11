package engine

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/guided-traffic/cogpit/internal/model"
)

// ConfirmRequest is what the user is asked to confirm: the command, who asked for it, and the
// objects, namespaces and context it acts on.
type ConfirmRequest struct {
	Command    string            `json:"command"`
	Label      string            `json:"label"`
	Danger     Danger            `json:"danger"`
	Origin     Origin            `json:"origin"`
	Context    string            `json:"context"`
	Namespaces []string          `json:"namespaces,omitempty"`
	Objects    []model.ObjectRef `json:"objects"`
}

// Message is the confirmation text: "Delete pod web-0 in namespace default (context kind-cogpit)?"
// with the origin named when an agent asked.
func (r ConfirmRequest) Message() string {
	var b strings.Builder
	if r.Origin.IsAgent() {
		b.WriteString("An agent asks to run: ")
	}
	b.WriteString(r.Label)
	b.WriteString(" ")
	names := make([]string, 0, len(r.Objects))
	for _, o := range r.Objects {
		names = append(names, o.Kind+" "+o.Name)
	}
	b.WriteString(strings.Join(names, ", "))
	if len(r.Namespaces) > 0 {
		b.WriteString(" in namespace ")
		b.WriteString(strings.Join(r.Namespaces, ", "))
	}
	fmt.Fprintf(&b, " (context %s)?", r.Context)
	return b.String()
}

// Confirmer asks the user to confirm a command. Confirm blocks until the user answers or ctx ends
// and must not be called on the GUI's main goroutine.
type Confirmer interface {
	Confirm(ctx context.Context, req ConfirmRequest) (bool, error)
}

// ConfirmFunc adapts a function to a Confirmer.
type ConfirmFunc func(ctx context.Context, req ConfirmRequest) (bool, error)

// Confirm calls f.
func (f ConfirmFunc) Confirm(ctx context.Context, req ConfirmRequest) (bool, error) {
	return f(ctx, req)
}

func newConfirmRequest(cmd *Command, origin Origin, uiContext string, targets []model.ObjectRef) ConfirmRequest {
	req := ConfirmRequest{
		Command: cmd.ID, Label: cmd.Label, Danger: cmd.Danger, Origin: origin,
		Context: uiContext, Objects: targets,
	}
	for _, t := range targets {
		if t.Context != "" {
			req.Context = t.Context
		}
		if t.Namespace != "" && !slices.Contains(req.Namespaces, t.Namespace) {
			req.Namespaces = append(req.Namespaces, t.Namespace)
		}
	}
	return req
}

// confirmedKey is the context key under which Dispatch hands a command the request the user
// confirmed.
type confirmedKey struct{}

// Confirmed returns the request the user confirmed for the running command, if it needed one.
// Commands that write check it against the cluster they are about to act on: if the user switched
// clusters while the dialog was open, the confirmation named another context and must not apply.
func Confirmed(ctx context.Context) (ConfirmRequest, bool) {
	req, ok := ctx.Value(confirmedKey{}).(ConfirmRequest)
	return req, ok
}

// CheckConfirmedContext fails when the running command was confirmed for another context than
// the one it is about to act on. Commands without a confirmation pass.
func CheckConfirmedContext(ctx context.Context, actOn string) error {
	req, ok := Confirmed(ctx)
	if !ok || req.Context == actOn {
		return nil
	}
	return fmt.Errorf("%w: confirmed for context %q, the connected context is now %q", ErrWrongCluster, req.Context, actOn)
}
