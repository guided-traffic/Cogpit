package commands

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
)

// StateResult is ui.state's result: the UI state and what the open view shows.
type StateResult struct {
	engine.UIState
	// ViewTitle is the open view's title, "Pods".
	ViewTitle string `json:"viewTitle,omitempty"`
	// Rows is the number of rows the view shows after the namespace and search filters.
	Rows int `json:"rows"`
	// Total is the number of objects of the view's kind before filtering.
	Total int `json:"total"`
	// Synced is false while the view's kind is still being listed.
	Synced bool `json:"synced"`
	// ViewError is what keeps the view from showing objects: a list the cluster refuses, or a
	// custom resource kind the cluster no longer serves.
	ViewError string `json:"viewError,omitempty"`
}

// NavigateResult is ui.navigate's result.
type NavigateResult struct {
	View  string          `json:"view"`
	Ref   model.ObjectRef `json:"ref"`
	Found bool            `json:"found"` // false: the object is not in the cache; the view and drawer show it once it appears
}

type viewParams struct {
	View string `json:"view" jsonschema:"kind key of the view to open: pods, services, or <plural>.<group> for a custom resource kind (gateways.gateway.networking.k8s.io); kinds_list lists them"`
}

type namespaceParams struct {
	Namespaces []string `json:"namespaces,omitempty" jsonschema:"namespaces to show; empty or omitted shows all namespaces"`
}

type filterParams struct {
	Filter string `json:"filter,omitempty" jsonschema:"search text; every whitespace-separated term must occur in a cell, case-insensitively; empty clears the search"`
}

type selectParams struct {
	Refs     []model.ObjectRef `json:"refs,omitempty" jsonschema:"objects of the open view to select; empty clears the selection"`
	NoDrawer bool              `json:"noDrawer,omitempty" jsonschema:"select without opening the detail drawer on the first object"`
}

type refParams struct {
	Ref model.ObjectRef `json:"ref" jsonschema:"the object"`
}

func registerUI(g *registrar) {
	add(g, engine.Spec[struct{}]{
		ID: "ui.state", Label: "Show UI State", Danger: engine.DangerRead,
		Description: "Returns what the user sees: the connected context and its connection state, the open view (kind key), " +
			"the namespace filter, the search text, the sort order, the selected objects, the object in the detail drawer, " +
			"the dock tabs, and how many rows the view shows of how many objects. Use it to learn which object the user means.",
		Run: func(_ context.Context, s *engine.Session, _ struct{}) (any, error) {
			u := s.UIState()
			res := StateResult{UIState: u}
			if t := s.Table(u.View); t != nil {
				res.ViewTitle, res.Rows, res.Total, res.Synced, res.ViewError = t.Title, len(t.Rows), t.Total, t.Synced, t.Error
			} else if b := s.Backend(); b != nil {
				if p, ok := b.Profile(u.View); ok {
					res.ViewTitle = p.Title
				}
			}
			return res, nil
		},
	})

	add(g, engine.Spec[viewParams]{
		ID: "ui.openView", Label: "Open View", MenuPath: "View/Open View", Danger: engine.DangerUI, Verbs: cacheReads,
		Description: "Opens the list view of a kind (pods, services, ...), as clicking it in the sidebar does. Clears the " +
			"selection, the detail drawer and the search text; keeps the namespace filter. Returns the new UI state.",
		Enabled: func(s *engine.Session, _ viewParams) error { _, err := connected(s); return err },
		Run: func(_ context.Context, s *engine.Session, p viewParams) (any, error) {
			_, prof, err := s.Kind(p.View)
			if err != nil {
				return nil, err
			}
			sort := s.ViewSort(prof.Key)
			return s.UpdateUI(func(u *engine.UIState) {
				u.View, u.Sort = prof.Key, sort
				u.Selected, u.Drawer, u.Filter = nil, nil, ""
			}), nil
		},
	})

	add(g, engine.Spec[namespaceParams]{
		ID: "ui.setNamespace", Label: "Select Namespaces", MenuPath: "View/Namespaces", Danger: engine.DangerUI,
		Description: "Sets the namespace filter of the namespaced views; an empty list shows all namespaces. Returns the new UI state.",
		Run: func(_ context.Context, s *engine.Session, p namespaceParams) (any, error) {
			var ns []string
			for _, n := range p.Namespaces {
				n = strings.TrimSpace(n)
				if n != "" && !slices.Contains(ns, n) {
					ns = append(ns, n)
				}
			}
			return s.UpdateUI(func(u *engine.UIState) { u.Namespaces = ns }), nil
		},
	})

	add(g, engine.Spec[filterParams]{
		ID: "ui.setFilter", Label: "Search", MenuPath: "View/Search", Danger: engine.DangerUI,
		Description: "Sets the search text of the open view: a row is shown when every whitespace-separated term occurs, " +
			"case-insensitively, in one of its cells. An empty text clears the search. Returns the new UI state.",
		Run: func(_ context.Context, s *engine.Session, p filterParams) (any, error) {
			return s.UpdateUI(func(u *engine.UIState) { u.Filter = p.Filter }), nil
		},
	})

	add(g, engine.Spec[engine.SortSpec]{
		ID: "ui.sort", Label: "Sort", Danger: engine.DangerUI,
		Description: "Sorts the open view by a column id (name, namespace, age, ... as kinds_list reports them), " +
			"ascending unless desc is true. Empty cells sort last. The view remembers its sort order. Returns the new UI state.",
		Enabled: func(s *engine.Session, _ engine.SortSpec) error { _, err := openView(s); return err },
		Run: func(_ context.Context, s *engine.Session, p engine.SortSpec) (any, error) {
			prof, err := openView(s)
			if err != nil {
				return nil, err
			}
			if prof.ColumnIndex(p.Column) < 0 {
				return nil, fmt.Errorf("the %s view has no column %q; columns: %s", prof.Key, p.Column, columnIDs(prof))
			}
			applied := false
			st := s.UpdateUI(func(u *engine.UIState) {
				if u.View == prof.Key {
					u.Sort, applied = p, true
				}
			})
			if !applied {
				return nil, errors.New("the open view changed; sort again")
			}
			return st, nil
		},
	})

	add(g, engine.Spec[selectParams]{
		ID: "ui.select", Label: "Select", Danger: engine.DangerUI, Verbs: cacheReads,
		Description: "Selects objects of the open view and, unless noDrawer is true, opens the detail drawer on the first " +
			"one. An empty list clears the selection (and closes the drawer unless noDrawer). To select an object of " +
			"another kind or one hidden by the filters, use ui_navigate. Returns the new UI state.",
		Enabled: func(s *engine.Session, _ selectParams) error { _, err := openView(s); return err },
		Run: func(_ context.Context, s *engine.Session, p selectParams) (any, error) {
			prof, err := openView(s)
			if err != nil {
				return nil, err
			}
			refs := make([]model.ObjectRef, 0, len(p.Refs))
			for _, r := range p.Refs {
				if r.Kind != prof.Key {
					return nil, fmt.Errorf("%s is not of the open view's kind %s; use ui_navigate", r.String(), prof.Key)
				}
				rr, _, err := s.ResolveRef(r)
				if err != nil {
					return nil, err
				}
				if !slices.ContainsFunc(refs, rr.Same) {
					refs = append(refs, rr)
				}
			}
			return s.UpdateUI(func(u *engine.UIState) {
				u.Selected = refs
				if p.NoDrawer {
					return
				}
				if len(refs) == 0 {
					u.Drawer = nil
					return
				}
				d := refs[0]
				u.Drawer = &d
			}), nil
		},
	})

	add(g, engine.Spec[struct{}]{
		ID: "ui.closeDrawer", Label: "Close Details", MenuPath: "View/Close Details", Danger: engine.DangerUI,
		Description: "Closes the detail drawer. Returns the new UI state.",
		Run: func(_ context.Context, s *engine.Session, _ struct{}) (any, error) {
			return s.UpdateUI(func(u *engine.UIState) { u.Drawer = nil }), nil
		},
	})

	add(g, engine.Spec[refParams]{
		ID: "ui.navigate", Label: "Go to Object", MenuPath: "View/Go to Object", Danger: engine.DangerUI, Verbs: cacheReads,
		Description: "Shows one object to the user: opens its kind's view, makes sure the namespace filter and the search " +
			"text do not hide it (resetting them if they do), selects it and opens its detail drawer. Use it after " +
			"resources_find. Reads only the cache. Returns {view, ref, found}; found is false when the object is not in the cache.",
		Enabled: func(s *engine.Session, _ refParams) error { _, err := connected(s); return err },
		Run:     navigate,
	})
}

// navigate implements ui.navigate. It decides from the cache, not from the view's snapshot, so it
// is correct before the view's first snapshot exists.
func navigate(ctx context.Context, s *engine.Session, p refParams) (any, error) {
	ref, prof, err := s.ResolveRef(p.Ref)
	if err != nil {
		return nil, err
	}
	var row *model.Row
	obj, err := s.Object(ctx, ref)
	switch {
	case err == nil:
		r := model.BuildRow(prof, ref.Context, obj)
		row = &r
	case !errors.Is(err, engine.ErrNotFound):
		return nil, err
	}
	sort := s.ViewSort(prof.Key)
	st := s.UpdateUI(func(u *engine.UIState) {
		if u.View != prof.Key {
			u.View, u.Sort, u.Filter = prof.Key, sort, ""
		}
		if prof.Namespaced && len(u.Namespaces) > 0 && !slices.Contains(u.Namespaces, ref.Namespace) {
			u.Namespaces = nil
		}
		if u.Filter != "" && (row == nil || !model.MatchesFilter(*row, u.Filter)) {
			u.Filter = ""
		}
		u.Selected = []model.ObjectRef{ref}
		d := ref
		u.Drawer = &d
	})
	return NavigateResult{View: st.View, Ref: ref, Found: row != nil}, nil
}

// openView returns the profile of the open view, or why there is none.
func openView(s *engine.Session) (*model.KindProfile, error) {
	b, err := connected(s)
	if err != nil {
		return nil, err
	}
	view := s.UIState().View
	p, ok := b.Profile(view)
	if !ok {
		return nil, errors.New("no list view is open")
	}
	return p, nil
}

func columnIDs(p *model.KindProfile) string {
	ids := make([]string, 0, len(p.Columns))
	for _, c := range p.Columns {
		ids = append(ids, c.ID)
	}
	return strings.Join(ids, ", ")
}
