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

// The table commands (ui-parity › Tables): which columns a view shows and the Definitions view's
// group filter. Both are UI state, so agents see them in ui_state and set them as the user does.

type columnsParams struct {
	View   string   `json:"view,omitempty" jsonschema:"kind key of the view; empty or omitted: the open view"`
	Hidden []string `json:"hidden,omitempty" jsonschema:"ids of the columns to hide (kinds_list lists a kind's columns, hidden: true for those hidden by default); every other column is shown, and at least one must stay visible"`
	Reset  bool     `json:"reset,omitempty" jsonschema:"true: go back to the view's default columns; hidden is then ignored"`
}

type groupsParams struct {
	Groups []string `json:"groups,omitempty" jsonschema:"API groups the Definitions view shows (cert-manager.io); empty or omitted shows all groups"`
}

func registerTables(g *registrar) {
	add(g, engine.Spec[columnsParams]{
		ID: "ui.setColumns", Label: "Columns", MenuPath: "View/Columns", Danger: engine.DangerUI,
		Description: "Shows and hides the columns of a view, as the ⋮ at the end of the table's header does: hidden " +
			"lists the column ids to hide, every other column is shown. Each view keeps its own columns; reset returns " +
			"a view to its default columns (some kinds hide columns by default, such as a custom resource's printer " +
			"columns of a higher priority). ui_state shows the views whose columns differ from their defaults in " +
			"hiddenColumns. Returns the new UI state.",
		Enabled: func(s *engine.Session, _ columnsParams) error { _, err := connected(s); return err },
		Run: func(_ context.Context, s *engine.Session, p columnsParams) (any, error) {
			view := p.View
			if view == "" {
				view = s.UIState().View
			}
			if view == "" {
				return nil, errors.New("no view is open; name one in view")
			}
			_, prof, err := s.Kind(view)
			if err != nil {
				return nil, err
			}
			hidden, err := hiddenColumns(prof, p.Hidden)
			if err != nil {
				return nil, err
			}
			defaults := engine.HiddenByDefault(prof.Columns)
			return s.UpdateUI(func(u *engine.UIState) {
				if p.Reset || slices.Equal(hidden, defaults) {
					delete(u.HiddenColumns, view)
					return
				}
				if u.HiddenColumns == nil {
					u.HiddenColumns = map[string][]string{}
				}
				u.HiddenColumns[view] = hidden
			}), nil
		},
	})

	add(g, engine.Spec[groupsParams]{
		ID: "ui.setGroups", Label: "Select API Groups", MenuPath: "View/API Groups", Danger: engine.DangerUI,
		Description: "Sets the API group filter of the Definitions view (customresourcedefinitions), which Lens shows " +
			"in place of the namespace filter: only the definitions of these groups are listed. An empty list shows " +
			"all groups. Returns the new UI state.",
		Run: func(_ context.Context, s *engine.Session, p groupsParams) (any, error) {
			var gs []string
			for _, g := range p.Groups {
				g = strings.TrimSpace(g)
				if g != "" && !slices.Contains(gs, g) {
					gs = append(gs, g)
				}
			}
			return s.UpdateUI(func(u *engine.UIState) { u.Groups = gs }), nil
		},
	})
}

// hiddenColumns checks the ids against a kind's columns and returns them in the columns' order.
// Hiding every column is refused: a table without columns shows nothing to click.
func hiddenColumns(prof *model.KindProfile, ids []string) ([]string, error) {
	var out []string
	for _, c := range prof.Columns {
		if slices.Contains(ids, c.ID) {
			out = append(out, c.ID)
		}
	}
	for _, id := range ids {
		if prof.ColumnIndex(id) < 0 {
			return nil, fmt.Errorf("the %s view has no column %q; columns: %s", prof.Key, id, columnIDs(prof))
		}
	}
	if len(out) == len(prof.Columns) {
		return nil, fmt.Errorf("at least one column of the %s view must stay visible", prof.Key)
	}
	return out, nil
}
