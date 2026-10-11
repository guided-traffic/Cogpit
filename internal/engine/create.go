package engine

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
)

// CreateItem is the outcome for one object of a Create: its reference (with the name the server
// gave it, for a generateName), and whether it was created or why not.
type CreateItem struct {
	Ref     model.ObjectRef `json:"ref"`
	Created bool            `json:"created"`
	Error   string          `json:"error,omitempty"`
}

// CreateResult is the outcome of a Create, one item per object in the text's order.
type CreateResult struct {
	Items   []CreateItem `json:"items"`
	Created int          `json:"created"`
	Failed  int          `json:"failed"`
}

// createPlan is one object of a Create, mapped to its resource and namespace.
type createPlan struct {
	obj *unstructured.Unstructured
	t   ObjectTarget
	err error // the object cannot be created (unknown kind, redacted kind, no name)
}

// planCreate parses text and maps each object to its resource through discovery. An object lands
// in its own metadata.namespace, else in namespace, else in the namespace filter's single
// namespace, else in the context's default namespace; a cluster-scoped kind's namespace is
// dropped. Objects that cannot be created carry their error; a text that does not parse fails.
func (s *Session) planCreate(ctx context.Context, text, namespace string) (*Backend, ObjectWriter, []createPlan, error) {
	b := s.Backend()
	if b == nil {
		return nil, nil, nil, ErrNotConnected
	}
	w, err := b.ObjectWriter()
	if err != nil {
		return nil, nil, nil, err
	}
	objs, err := ParseManifests(text)
	if err != nil {
		return nil, nil, nil, err
	}
	fallback := strings.TrimSpace(namespace)
	if fallback == "" {
		if ns := s.UIState().Namespaces; len(ns) == 1 {
			fallback = ns[0]
		}
	}
	if fallback == "" {
		fallback = b.Namespace
	}
	plans := make([]createPlan, 0, len(objs))
	for _, obj := range objs {
		p := createPlan{obj: obj.DeepCopy()}
		p.t.APIVersion, p.t.Kind = obj.GetAPIVersion(), obj.GetKind()
		p.t.Ref = model.ObjectRef{Context: b.Context, Kind: obj.GetKind(), Name: obj.GetName()}
		if p.t.Ref.Name == "" && obj.GetGenerateName() != "" {
			p.t.Ref.Name = obj.GetGenerateName() + "*"
		}
		switch {
		case RedactedKind(p.t.APIVersion, p.t.Kind):
			p.err = errRedactedKind(p.t.Kind)
		case obj.GetName() == "" && obj.GetGenerateName() == "":
			p.err = fmt.Errorf("%w: a %s needs metadata.name (or metadata.generateName)", ErrManifest, p.t.Kind)
		}
		if p.err == nil {
			mctx, cancel := context.WithTimeout(ctx, mappingTimeout)
			m, merr := w.MapKind(mctx, p.t.APIVersion, p.t.Kind)
			cancel()
			if merr != nil {
				p.err = merr
			} else {
				p.t.GVR, p.t.Namespaced = m.GVR, m.Namespaced
				p.t.Ref.Kind = KindKeyFor(b, m.GVR)
			}
		}
		if p.err == nil {
			if p.t.Namespaced {
				ns := obj.GetNamespace()
				if ns == "" {
					ns = fallback
				}
				p.obj.SetNamespace(ns)
				p.t.Ref.Namespace = ns
			} else {
				unstructured.RemoveNestedField(p.obj.Object, "metadata", "namespace")
			}
			// Server-owned fields that a copied manifest may carry would make the create fail or
			// mislead: a create never names a version or an identity.
			for _, f := range []string{"resourceVersion", "uid", "managedFields", "creationTimestamp"} {
				unstructured.RemoveNestedField(p.obj.Object, "metadata", f)
			}
		}
		plans = append(plans, p)
	}
	return b, w, plans, nil
}

// CreateTargets returns the objects a Create of text would make, for the confirmation that names
// them.
func (s *Session) CreateTargets(ctx context.Context, text, namespace string) ([]model.ObjectRef, error) {
	_, _, plans, err := s.planCreate(ctx, text, namespace)
	if err != nil {
		return nil, err
	}
	refs := make([]model.ObjectRef, 0, len(plans))
	for _, p := range plans {
		refs = append(refs, p.t.Ref)
	}
	return refs, nil
}

// CreateObjects creates the objects of a YAML text in order, each after a server-side dry run of
// its own, so that a Namespace earlier in the text exists when an object in it is checked. An
// object that fails is reported and the next one is tried, as kubectl create does. The result
// lists every object; the error, if any object failed, names each failure.
func (s *Session) CreateObjects(ctx context.Context, text, namespace string) (CreateResult, error) {
	b, w, plans, err := s.planCreate(ctx, text, namespace)
	if err != nil {
		return CreateResult{}, err
	}
	if err := CheckConfirmedContext(ctx, b.Context); err != nil {
		return CreateResult{}, err
	}
	res := CreateResult{Items: make([]CreateItem, 0, len(plans))}
	var failures []string
	for _, p := range plans {
		item := CreateItem{Ref: p.t.Ref}
		err := p.err
		if err == nil {
			_, err = w.CreateObject(ctx, p.t.GVR, p.t.Ref.Namespace, p.obj, true)
		}
		if err == nil {
			var out *unstructured.Unstructured
			if out, err = w.CreateObject(ctx, p.t.GVR, p.t.Ref.Namespace, p.obj, false); err == nil && out != nil && out.GetName() != "" {
				item.Ref.Name = out.GetName()
			}
		}
		if err != nil {
			item.Error = err.Error()
			res.Failed++
			failures = append(failures, item.Ref.String()+": "+item.Error)
		} else {
			item.Created = true
			res.Created++
		}
		res.Items = append(res.Items, item)
		if ctx.Err() != nil {
			break
		}
	}
	if res.Created > 0 {
		s.Emit(Event{Type: EventToast, Level: LevelInfo, Message: createdMessage(res)})
	}
	if res.Failed > 0 {
		return res, fmt.Errorf("%d of %d objects not created: %s", res.Failed, len(plans), strings.Join(failures, "; "))
	}
	if len(res.Items) < len(plans) {
		return res, errors.Join(ctx.Err(), errors.New("the create was cancelled before every object was created"))
	}
	return res, nil
}

func createdMessage(res CreateResult) string {
	if res.Created == 1 {
		for _, it := range res.Items {
			if it.Created {
				return "Created " + it.Ref.String()
			}
		}
	}
	return fmt.Sprintf("Created %d objects", res.Created)
}
