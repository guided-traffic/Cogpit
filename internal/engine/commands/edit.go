package commands

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
)

// Editing (G9): Edit opens an object's YAML from the cache in a dock tab; editor_diff shows the
// edited text against the object as the cache holds it now; object_dryRun asks the server to
// validate it; object_apply writes it with the resourceVersion the edit is based on as
// precondition, so a concurrent change is never overwritten. Create resource tabs and
// object_create make new objects from YAML. Every step is a command, so agents can drive the
// whole flow; apply and create are mutating and follow D-2.

// EditResult is what object.edit returns: the Edit tab and its text.
type EditResult struct {
	Tab                 engine.DockTab `json:"tab"`
	Created             bool           `json:"created"` // false: the tab already open on the object was activated
	Text                string         `json:"yaml"`
	Revision            uint64         `json:"revision"`
	BaseResourceVersion string         `json:"baseResourceVersion"`
}

// TextResult is what editor.setText returns.
type TextResult struct {
	Revision uint64 `json:"revision"`
}

type editorParams struct {
	ID string `json:"id" jsonschema:"editor tab id as object_edit or resource_createTab returned it, e.g. edit-1"`
}

type editorTextParams struct {
	ID   string `json:"id" jsonschema:"editor tab id, e.g. edit-1 or create-1"`
	YAML string `json:"yaml" jsonschema:"the tab's new text"`
}

type editParams struct {
	ID   string `json:"id" jsonschema:"Edit tab id as object_edit returned it, e.g. edit-1"`
	YAML string `json:"yaml,omitempty" jsonschema:"the edited YAML of the object; empty uses the tab's text"`
}

type diffParams struct {
	ID      string `json:"id" jsonschema:"Edit tab id as object_edit returned it, e.g. edit-1"`
	YAML    string `json:"yaml,omitempty" jsonschema:"the edited YAML of the object; empty uses the tab's text"`
	Context *int   `json:"context,omitempty" jsonschema:"unchanged lines to keep around each change; default 3; -1 keeps every line"`
}

type createParams struct {
	YAML      string `json:"yaml" jsonschema:"one or more manifests separated by ---, or a v1 List"`
	Namespace string `json:"namespace,omitempty" jsonschema:"namespace for objects whose manifest names none; empty uses the namespace filter if it names exactly one namespace, else the context's default namespace"`
}

type createTabParams struct {
	YAML     string `json:"yaml,omitempty" jsonschema:"text to start the tab with, e.g. a manifest for the user to review and create"`
	Template string `json:"template,omitempty" jsonschema:"start from a template instead: its kind, e.g. Deployment (resource_templates lists them); default Pod"`
}

// edits are the verbs of apply and its dry run: an update of whatever kind the tab edits.
var edits = []engine.Verb{{Verb: "update", Resource: "*"}}

// lookupTimeout bounds the discovery lookups of Targets, which has no context of its own.
const lookupTimeout = 15 * time.Second

func registerEdit(g *registrar) {
	add(g, engine.Spec[refParams]{
		ID: "object.edit", Label: "Edit", MenuPath: "Object/Edit", Order: engine.OrderEdit, Danger: engine.DangerUI,
		Verbs: append(append([]engine.Verb(nil), cacheReads...), edits...), // offered when the user may update it (access reviews)
		Description: "Opens a dock tab with an object's YAML from Cogpit's cache (managed fields dropped) for editing, or " +
			"activates the tab already open on it. Returns the tab id, the text and the resourceVersion the edit is based " +
			"on. Continue with editor_diff, object_dryRun and object_apply. Secrets cannot be edited (their values are redacted).",
		Enabled: func(s *engine.Session, p refParams) error { return editable(s, p.Ref) },
		Run: func(ctx context.Context, s *engine.Session, p refParams) (any, error) {
			st, created, err := s.OpenEditTab(ctx, p.Ref)
			if err != nil {
				return nil, err
			}
			return EditResult{Tab: st.Tab, Created: created, Text: st.Text, Revision: st.Revision, BaseResourceVersion: st.BaseResourceVersion}, nil
		},
	})

	add(g, engine.Spec[editorParams]{
		ID: "editor.get", Label: "Read Editor", Danger: engine.DangerRead, Verbs: cacheReads,
		Description: "Returns an editor tab's text (what the user has typed so far) and its revision; for an Edit tab also " +
			"the resourceVersion the edit is based on, the one Cogpit's cache holds now, and whether they differ (stale: an " +
			"apply would be refused with a conflict) or the object is gone (missing); for a Helm Upgrade tab the release " +
			"revision its values are based on. For agents, the values of any Secret in the text are replaced by " +
			"<redacted>, and a Helm Upgrade tab's values are withheld whole.",
		Run: func(ctx context.Context, s *engine.Session, p editorParams) (any, error) {
			st, err := s.EditorState(ctx, p.ID)
			switch {
			case err != nil || !engine.OriginOf(ctx).IsAgent():
			case st.Tab.Type == engine.DockHelmUpgrade:
				st.Text = model.Withheld // a release's values can hold credentials (helmupgrade.go)
			default:
				// A Secret the user pasted into a Create tab is theirs to see, not an LLM's.
				st.Text, _ = engine.RedactManifests(st.Text, false)
			}
			return st, err
		},
	})

	add(g, engine.Spec[editorTextParams]{
		ID: "editor.setText", Label: "Set Editor Text", Danger: engine.DangerUI,
		Description: "Replaces an editor tab's text, as typing does; the tab shows it unless the user has unsaved typing " +
			"of their own. Writes nothing to the cluster. Returns the text's new revision.",
		Run: func(_ context.Context, s *engine.Session, p editorTextParams) (any, error) {
			rev, err := s.SetEditorText(p.ID, p.YAML)
			if err != nil {
				return nil, err
			}
			return TextResult{Revision: rev}, nil
		},
	})

	add(g, engine.Spec[diffParams]{
		ID: "editor.diff", Label: "Show Changes", Danger: engine.DangerRead, Verbs: cacheReads,
		Description: "Returns the line diff of an Edit tab's text (or of yaml) against the object as Cogpit's cache holds it " +
			"now: lines with op \"-\" (removed), \"+\" (added) or \"=\" (context), with line numbers. Also checks that the text " +
			"still names the same object. stale means the object changed since the edit's base version: its changes show " +
			"as reverted, and an apply would be refused with a conflict.",
		Enabled: func(s *engine.Session, _ diffParams) error { _, err := writable(s); return err },
		Run: func(ctx context.Context, s *engine.Session, p diffParams) (any, error) {
			n := engine.DefaultDiffContext
			if p.Context != nil {
				n = *p.Context
			}
			return s.EditDiff(ctx, p.ID, p.YAML, n)
		},
	})

	add(g, engine.Spec[editParams]{
		ID: "object.dryRun", Label: "Check with Server", Danger: engine.DangerRead, Verbs: edits,
		Description: "Sends an Edit tab's text (or yaml) to the API server as an update with dryRun=All: the server " +
			"validates it and runs admission, and stores nothing. Carries the edit's base resourceVersion, so a concurrent " +
			"change is reported as a conflict (then the tab compares with the newer version). Needs the update permission.",
		Enabled: func(s *engine.Session, _ editParams) error { _, err := writable(s); return err },
		Run: func(ctx context.Context, s *engine.Session, p editParams) (any, error) {
			return s.DryRunEdit(ctx, p.ID, p.YAML)
		},
	})

	add(g, engine.Spec[editParams]{
		ID: "object.apply", Label: "Apply", MenuPath: "Object/Apply", Danger: engine.DangerMutating, Verbs: edits,
		Description: "Writes an Edit tab's text (or yaml) to the cluster: a server-side dry run, then an update with the " +
			"resourceVersion the edit is based on as precondition. If the object changed meanwhile nothing is written and " +
			"the result is a conflict; the tab then compares with the newer version, and applying again overwrites it. On " +
			"success returns the stored version, which becomes the tab's text. An agent's request waits for the user's acceptance. " +
			"An edit that scales the object to zero replicas is destructive and is always confirmed, as Scale to zero is.",
		Escalate: func(s *engine.Session, p editParams) bool { return s.EditScalesToZero(p.ID, p.YAML) },
		Enabled:  func(s *engine.Session, _ editParams) error { _, err := writable(s); return err },
		Targets: func(s *engine.Session, p editParams) ([]model.ObjectRef, error) {
			st, err := editTabRef(s, p.ID)
			if err != nil {
				return nil, err
			}
			return []model.ObjectRef{st}, nil
		},
		Run: func(ctx context.Context, s *engine.Session, p editParams) (any, error) {
			return s.ApplyEdit(ctx, p.ID, p.YAML)
		},
	})

	add(g, engine.Spec[createParams]{
		ID: "object.create", Label: "Create", MenuPath: "Object/Create", Danger: engine.DangerMutating,
		Verbs: []engine.Verb{{Verb: "create", Resource: "*"}},
		Description: "Creates the objects of a YAML text (several documents separated by ---) in order, each after a " +
			"server-side dry run; an object that fails is reported and the next one is tried. Objects without a namespace " +
			"land in namespace, else in the namespace filter's single namespace, else in the context's default. Secrets are " +
			"refused. Returns one item per object. An agent's request waits for the user's acceptance.",
		Enabled: func(s *engine.Session, _ createParams) error { _, err := writable(s); return err },
		Targets: func(s *engine.Session, p createParams) ([]model.ObjectRef, error) {
			ctx, cancel := context.WithTimeout(context.Background(), lookupTimeout)
			defer cancel()
			return s.CreateTargets(ctx, p.YAML, p.Namespace)
		},
		Run: func(ctx context.Context, s *engine.Session, p createParams) (any, error) {
			return s.CreateObjects(ctx, p.YAML, p.Namespace)
		},
	})

	add(g, engine.Spec[createTabParams]{
		ID: "resource.createTab", Label: "Create Resource", MenuPath: "View/Create Resource", Danger: engine.DangerUI,
		Description: "Opens a Create resource dock tab with yaml, with a template (resource_templates lists them), or with " +
			"an example Pod. Writes nothing: the user reviews the text and creates it, or object_create does. Returns the tab.",
		Enabled: func(s *engine.Session, _ createTabParams) error { _, err := writable(s); return err },
		Run: func(_ context.Context, s *engine.Session, p createTabParams) (any, error) {
			text := p.YAML
			if text == "" {
				name := p.Template
				if name == "" {
					name = engine.DefaultTemplate
				}
				t, err := engine.TemplateByName(name)
				if err != nil {
					return nil, err
				}
				text = t.Text
			} else if p.Template != "" {
				return nil, errors.New("give yaml or template, not both")
			}
			st, err := s.OpenCreateTab(text)
			if err != nil {
				return nil, err
			}
			return EditResult{Tab: st.Tab, Created: true, Text: st.Text, Revision: st.Revision}, nil
		},
	})

	add(g, engine.Spec[struct{}]{
		ID: "resource.templates", Label: "List Templates", Danger: engine.DangerRead,
		Description: "Lists Cogpit's templates for new objects, one per kind in alphabetical order, each with its " +
			"apiVersion, whether the kind is namespaced, and its YAML (without a namespace).",
		Run: func(context.Context, *engine.Session, struct{}) (any, error) { return engine.Templates() },
	})
}

// editable is object.edit's Enabled check. It must not block: a kind with a profile is checked
// against the cache without waiting; a kind without one is resolved when the command runs.
func editable(s *engine.Session, ref model.ObjectRef) error {
	b, err := writable(s)
	if err != nil {
		return err
	}
	if _, err := b.ObjectWriter(); err != nil {
		return err
	}
	if p, ok := b.Profile(ref.Kind); ok {
		if !p.APIObject() {
			return fmt.Errorf("%s are not objects of the cluster's API: %w", p.Title, engine.ErrNotApplicable)
		}
		gv := p.Group + "/" + p.Version
		if p.Group == "" {
			gv = p.Version
		}
		if engine.RedactedKind(gv, p.Kind) {
			return fmt.Errorf("%s cannot be edited here: their values are redacted (H-8)", p.Title)
		}
		_, err := s.Lookup(ref)
		return err
	}
	if ref.Kind == "secrets" {
		return errors.New("editing Secrets is off: their values are redacted (H-8)")
	}
	return nil
}

// editTabRef is the object an Edit tab edits, for the confirmation of object.apply.
func editTabRef(s *engine.Session, id string) (model.ObjectRef, error) {
	for _, t := range s.UIState().Dock {
		if t.ID != id {
			continue
		}
		if t.Type != engine.DockEdit || t.Ref == nil {
			return model.ObjectRef{}, fmt.Errorf("dock tab %q is not an Edit tab", id)
		}
		return *t.Ref, nil
	}
	return model.ObjectRef{}, fmt.Errorf("%w: %q", engine.ErrNotEditorTab, id)
}

// objectTarget is the Targets function of the generic commands on one object (Delete): the
// resolved reference of any kind, so that the confirmation names its namespace and context.
func objectTarget(s *engine.Session, ref model.ObjectRef) ([]model.ObjectRef, error) {
	ctx, cancel := context.WithTimeout(context.Background(), lookupTimeout)
	defer cancel()
	t, _, err := s.ResolveTarget(ctx, ref)
	if err != nil {
		return nil, err
	}
	return []model.ObjectRef{t.Ref}, nil
}

// cachedAny is the Enabled check of the generic commands on one object: the object must be in the
// cache when its kind has a profile (checked without waiting); other kinds are resolved when the
// command runs.
func cachedAny(s *engine.Session, ref model.ObjectRef) error {
	b, err := writable(s)
	if err != nil {
		return err
	}
	if _, err := b.ObjectWriter(); err != nil {
		return err
	}
	if p, ok := b.Profile(ref.Kind); ok {
		if !p.APIObject() {
			return fmt.Errorf("%s are not objects of the cluster's API: %w", p.Title, engine.ErrNotApplicable)
		}
		_, err := s.Lookup(ref)
		return err
	}
	if ref.Name == "" {
		return errors.New("the reference has no name")
	}
	return nil
}
