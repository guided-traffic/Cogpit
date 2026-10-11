package engine

import (
	"context"
	"errors"
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
	// UIDs are the objects' UIDs as Cogpit's cache held them when the user was asked, in the order
	// of Objects ("" for an object not in the cache): destructive commands act on exactly those
	// objects, never on one created again under the same name while the dialog was open. Only
	// destructive commands record them.
	UIDs []string `json:"uids,omitempty"`
	// Detail adds what the objects alone don't say: the options that make a drain lose data.
	Detail string `json:"detail,omitempty"`
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
	if r.Detail != "" {
		b.WriteString(" ")
		b.WriteString(r.Detail)
	}
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

func newConfirmRequest(cmd *Command, danger Danger, origin Origin, uiContext string, targets []model.ObjectRef) ConfirmRequest {
	req := ConfirmRequest{
		Command: cmd.ID, Label: cmd.Label, Danger: danger, Origin: origin,
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

// originKey is the context key under which Dispatch hands a command its origin.
type originKey struct{}

// OriginOf returns the origin that dispatched the running command; empty outside Dispatch.
// Commands use it where the answer depends on who asks: agents get Secret values redacted.
func OriginOf(ctx context.Context) Origin {
	o, _ := ctx.Value(originKey{}).(Origin)
	return o
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

// ErrTargetsChanged is returned when the objects a command would act on are not the ones the user
// confirmed. Match it with errors.Is.
var ErrTargetsChanged = errors.New("the objects changed since they were confirmed")

// ConfirmedUID returns the UID the confirmation recorded for ref and whether the running command
// was confirmed at all. A confirmed command whose object was not in the cache when the user was
// asked gets "" and true: it must refuse rather than act on whatever has the name now.
func ConfirmedUID(ctx context.Context, ref model.ObjectRef) (uid string, confirmed bool) {
	req, ok := Confirmed(ctx)
	if !ok || len(req.UIDs) == 0 {
		return "", ok && req.Danger == DangerDestructive
	}
	for i, o := range req.Objects {
		if i < len(req.UIDs) && o.Kind == ref.Kind && o.Namespace == ref.Namespace && o.Name == ref.Name {
			return req.UIDs[i], true
		}
	}
	return "", true
}

// CheckConfirmedTargets fails when the objects a command is about to act on are not the ones the
// user confirmed: a namespace that came from the UI's filter and changed while the dialog was
// open, a manifest whose objects moved. Commands without a confirmation, or confirmed without
// objects (an agent allowance), pass.
func CheckConfirmedTargets(ctx context.Context, refs []model.ObjectRef) error {
	req, ok := Confirmed(ctx)
	if !ok || len(req.Objects) == 0 {
		return nil
	}
	same := len(refs) == len(req.Objects)
	for i := 0; same && i < len(refs); i++ {
		a, b := refs[i], req.Objects[i]
		same = a.Kind == b.Kind && a.Namespace == b.Namespace && a.Name == b.Name
	}
	if !same {
		return fmt.Errorf("%w (the namespace filter or the text moved); nothing was done", ErrTargetsChanged)
	}
	return nil
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
