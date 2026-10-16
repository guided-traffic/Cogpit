package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// The create dialogs (G7): Lens creates Namespaces, Service Accounts, Roles, Cluster Roles, Role
// Bindings, Cluster Role Bindings and Resource Quotas from small dialogs instead of YAML. Their
// commands (package commands, createdialogs.go) build one object from the dialog's fields and
// create it the way a Create resource tab creates its text (CreateObjects): discovery maps the
// kind, a server-side dry run checks the object, then it is created, after the confirmation's
// context and objects are checked; a toast reports it. The helpers here are what those commands
// need from the engine beyond that.

// CreateNamespace returns the namespace a create dialog's object lands in, in a connected context's
// cluster ("" the shown one, D-69): namespace when given, else the namespace filter's single
// namespace (the shown cluster's), else the context's default namespace, the order in which
// object.create places an object whose manifest names none.
func (s *Session) CreateNamespace(contextID, namespace string) (string, error) {
	if ns := strings.TrimSpace(namespace); ns != "" {
		return ns, nil
	}
	b, err := s.TargetBackend(contextID)
	if err != nil {
		return "", err
	}
	if ns := s.filterNamespace(b.Context); ns != "" {
		return ns, nil
	}
	if b.Namespace != "" {
		return b.Namespace, nil
	}
	return defaultNamespace, nil
}

// CreatePermitted is the access check of a create dialog's command. The registry's own check
// asks about the object a command's "ref" names, and a create names no object yet, so these
// commands ask about their verb in the namespace the object will be created in ("" for a
// cluster-scoped kind). A known denial is an error wrapping ErrNotPermitted ("not permitted: you
// may not create rolebindings.rbac.authorization.k8s.io in team-a"); an answer the access
// reviews do not hold yet, or a backend without them, passes, and the API server decides.
func (s *Session) CreatePermitted(contextID, namespace string, v Verb) error {
	b := s.BackendOf(contextID) // the context's cluster's reviews, "" the shown one (D-69)
	if b == nil || b.Access == nil {
		return nil
	}
	if allowed, known := b.Access.Allowed(namespace, v); known && !allowed {
		return fmt.Errorf("%w: you may not %s", ErrNotPermitted, describeVerb(v, namespace))
	}
	return nil
}

// CreateManifest creates one object a create dialog's command built (apiVersion, kind and
// metadata with a name; a namespaced kind's metadata names its namespace) as object.create
// creates a text: discovery maps its kind, a server-side dry run checks it, then it is created,
// after the confirmation's context and objects are checked; a toast reports it. An object the
// server refuses comes back as an error that says why, without a multi-document create's count.
func (s *Session) CreateManifest(ctx context.Context, contextID string, obj map[string]any) (CreateItem, error) {
	raw, err := json.Marshal(obj) // JSON is YAML: CreateObjects parses it as one document
	if err != nil {
		return CreateItem{}, err
	}
	res, err := s.CreateObjects(ctx, contextID, string(raw), "")
	if len(res.Items) == 1 {
		it := res.Items[0]
		if it.Created {
			return it, nil
		}
		if it.Error != "" {
			return it, fmt.Errorf("%s was not created: %s", it.Ref.String(), it.Error)
		}
	}
	if err != nil {
		return CreateItem{}, err
	}
	return CreateItem{}, errors.New("nothing was created")
}
