package model

import (
	"encoding/json"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// TitleProperties is the title of the drawer's first section. Lens merges an object's metadata
// and its kind's state into one Properties section (capture, 2026-10-11); a kind profile whose
// Detail returns a first section with this title has its items appended to the generic metadata
// items (see [MergeProperties]).
const TitleProperties = "Properties"

// DetailSection is one titled block of the detail drawer: name/value items, or a table.
//
// The JSON shape is part of MCP's object_get result; fields are only ever added, and every added
// field is optional:
//
//	{"title": "Capacity", "group": "", "items": [...], "table": {"columns": [...], "rows": [[...]]}}
type DetailSection struct {
	Title string `json:"title"`
	// Group, when set, is a heading shown once above consecutive sections of the same group:
	// "Init Containers" and "Containers" above one section per container, titled by its name.
	Group string       `json:"group,omitempty"`
	Items []DetailItem `json:"items,omitempty"`
	// Table, when set, is the section's content as a table: a node's Capacity and Allocatable.
	Table *DetailTable `json:"table,omitempty"`
}

// DetailTable is a small table in the drawer: column titles and rows of cell texts.
type DetailTable struct {
	Columns []string   `json:"columns"`
	Rows    [][]string `json:"rows"`
	// Links, when set, are the objects cells name, by row and column as in Rows: nil for a plain
	// cell. The drawer renders a cell with a link as a link that goes to the object (a binding's
	// ServiceAccount subjects). Rows still hold every cell's text.
	Links [][]*ObjectRef `json:"links,omitempty"`
}

// Link returns the object the cell at row and col names, or nil for a plain cell or one out of
// range.
func (t *DetailTable) Link(row, col int) *ObjectRef {
	if t == nil || row < 0 || row >= len(t.Links) || col < 0 || col >= len(t.Links[row]) {
		return nil
	}
	return t.Links[row][col]
}

// DetailItem is one line of a detail section: a name and a value.
//
// Value is always the plain text, lines separated by "\n" for lists, so that agents and plain
// renderers need nothing else. The other fields say how the drawer shows it, as Lens does
// (capture, 2026-10-11):
//
//   - Status colours the value (a status pill).
//   - Link is the object the value names (a node, a controller, a service account); the drawer
//     renders the value as a link that goes to it (ui.navigate; an empty Name opens the kind's
//     list view).
//   - Chips, when set, are the value's entries shown as chips, each with its own status and link
//     (labels, pod IPs, conditions, secrets).
//   - Fold, when set, folds the list behind its count: the drawer shows "<n> <Fold>" ("6 Labels",
//     "1 Label" for one entry) and expands it on click. The entries are the chips, else the lines
//     of Value.
//   - Actions are buttons beside the value that run a command for it: "Forward…" beside a
//     container port.
type DetailItem struct {
	Name    string       `json:"name,omitempty"`
	Value   string       `json:"value"`
	Status  Status       `json:"status,omitempty"`
	Link    *ObjectRef   `json:"link,omitempty"`
	Chips   []Chip       `json:"chips,omitempty"`
	Fold    string       `json:"fold,omitempty"`
	Actions []ItemAction `json:"actions,omitempty"`
}

// ItemAction is a button beside a drawer item's value (architecture §4.4): it runs the command
// Command with Params, which name everything the command needs from the item (the pod, the
// container, the port). Agents see it in object_get and can dispatch the same command. The GUI
// runs it as the row menu runs a command (a schema-generated dialog when the command needs more
// parameters, prefilled with Params), unless the GUI registers a dialog of its own for the
// command (the port-forward dialog).
type ItemAction struct {
	Label   string          `json:"label"`
	Command string          `json:"command"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Chip is one entry of a list value shown as a chip.
type Chip struct {
	Text   string     `json:"text"`
	Status Status     `json:"status,omitempty"`
	Link   *ObjectRef `json:"link,omitempty"`
}

// Entries returns how many entries a list value has: its chips, else its lines.
func (it DetailItem) Entries() int {
	if len(it.Chips) > 0 {
		return len(it.Chips)
	}
	if it.Value == "" {
		return 0
	}
	return strings.Count(it.Value, "\n") + 1
}

// ChipsItem returns an item whose value is a list of chips; Value is their texts, one per line.
// An empty list returns an item with an empty value, which detail builders drop.
func ChipsItem(name string, chips []Chip) DetailItem {
	texts := make([]string, len(chips))
	for i, c := range chips {
		texts[i] = c.Text
	}
	return DetailItem{Name: name, Value: strings.Join(texts, "\n"), Chips: chips}
}

// PropertiesSection returns the generic part of the drawer's first section, the metadata every
// kind shows: Created, Name, Namespace (a link), Labels (folded chips), Annotations (folded) and
// Controlled By (a link for built-in owner kinds). Lens shows no UID (capture, 2026-10-11).
func PropertiesSection(obj *unstructured.Unstructured, now time.Time) DetailSection {
	s := DetailSection{Title: TitleProperties}
	if ts := obj.GetCreationTimestamp(); !ts.IsZero() {
		s.Items = append(s.Items, DetailItem{
			Name:  "Created",
			Value: FormatAge(now.Sub(ts.Time)) + " ago (" + ts.UTC().Format(time.RFC3339) + ")",
		})
	}
	s.Items = append(s.Items, DetailItem{Name: "Name", Value: obj.GetName()})
	if ns := obj.GetNamespace(); ns != "" {
		s.Items = append(s.Items, DetailItem{Name: "Namespace", Value: ns, Link: &ObjectRef{Kind: "namespaces", Name: ns}})
	}
	if labels := SortedPairs(obj.GetLabels()); len(labels) > 0 {
		chips := make([]Chip, len(labels))
		for i, l := range labels {
			chips[i] = Chip{Text: l}
		}
		it := ChipsItem("Labels", chips)
		it.Fold = "Labels"
		s.Items = append(s.Items, it)
	}
	if ann := obj.GetAnnotations(); len(ann) > 0 {
		filtered := make(map[string]string, len(ann))
		for k, v := range ann {
			if k == "kubectl.kubernetes.io/last-applied-configuration" {
				continue // shown in the YAML; too long for the drawer, as in Lens
			}
			filtered[k] = v
		}
		if pairs := SortedPairs(filtered); len(pairs) > 0 {
			s.Items = append(s.Items, DetailItem{Name: "Annotations", Value: strings.Join(pairs, "\n"), Fold: "Annotations"})
		}
	}
	for _, o := range obj.GetOwnerReferences() {
		s.Items = append(s.Items, DetailItem{
			Name: "Controlled By", Value: o.Kind + " " + o.Name,
			Link: RefTo(o.APIVersion, o.Kind, obj.GetNamespace(), o.Name),
		})
	}
	return s
}

// MergeProperties returns the drawer's sections: props first, with the items of the kind's own
// Properties section (a first section titled [TitleProperties]) appended, then the kind's other
// sections in their order.
func MergeProperties(props DetailSection, kind []DetailSection) []DetailSection {
	out := make([]DetailSection, 0, 1+len(kind))
	if len(kind) > 0 && kind[0].Title == TitleProperties {
		props.Items = append(props.Items, kind[0].Items...)
		kind = kind[1:]
	}
	out = append(out, props)
	return append(out, kind...)
}
