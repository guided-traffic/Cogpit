package engine

import (
	"context"
	"errors"
	"regexp"
	"slices"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

// What an agent reads of the cluster's text (agents-and-mcp H-82 and H-63, owner 2026-10-14).
// Dispatch passes every result of an agent's command through forAgent: the values of the Secrets
// Cogpit's cache holds are replaced by «secret <namespace>/<name>/<key>» (secretscrub.go), and
// Flux's credential fields are withheld from YAML (kinds.WithholdFluxCredentials), in a logs or
// terminal tab's lines (dock.read), an object's YAML, drawer values and events (object.get), and an
// editor's text and diff (object.edit, editor.get, editor.diff, object.apply, editor.revert). The
// GUI and the CLI get the text as it is. A result type of another package that carries cluster
// text implements AgentResult.
//
// Writing a placeholder or a withheld marker back would replace the value in the cluster, so an
// agent's text that holds one is refused (ErrAgentMarkerWrite) wherever a Secret's text is
// (GuardSecretEditor, GuardSecretText).

// AgentResult is a command result that carries cluster text: Dispatch hands an agent's result to
// ForAgent and returns what it returns.
type AgentResult interface {
	ForAgent(v *AgentView) any
}

// AgentView filters text for an agent: the cached Secrets' values and Flux's credential fields.
type AgentView struct {
	m *secretMatcher // nil: no Secret cached, nothing to replace
}

// ErrAgentMarkerWrite refuses an agent's text that holds a placeholder or a withheld marker Cogpit
// put in place of a value: writing it would replace the value in the cluster.
var ErrAgentMarkerWrite = errors.New("the text holds «secret …» or <withheld…>, which Cogpit shows agents in place of a " +
	"Secret's value or a credential; writing it back would replace that value in the cluster, so the text was not " +
	"taken: leave those parts out of the text you send, or ask the user to make the change in Cogpit's window")

// agentMarkers start the texts Cogpit puts in place of a value for agents.
var agentMarkers = []string{secretPlaceholderPrefix, "<withheld"}

// GuardAgentMarkers refuses an agent a text (a command's yaml parameter) that holds a placeholder
// or a withheld marker.
func GuardAgentMarkers(ctx context.Context, text string) error {
	if !OriginOf(ctx).IsAgent() {
		return nil
	}
	for _, m := range agentMarkers {
		if strings.Contains(text, m) {
			return ErrAgentMarkerWrite
		}
	}
	return nil
}

// agentView returns the view of the connected backend's cache: its matcher is built again only
// when the cached Secrets changed.
func (s *Session) agentView() *AgentView {
	return &AgentView{m: s.Backend().secretMatcher()}
}

// forAgent filters an agent's command result.
func (s *Session) forAgent(res any) any {
	switch r := res.(type) {
	case nil:
		return nil
	case *DetailModel:
		s.agentView().Detail(r)
	case DockOutput:
		r.Lines = s.agentView().Lines(r.Lines)
		return r
	case EditorState:
		r.Text = s.agentView().TabText(r.Tab.Type, r.Text)
		return r
	case EditDiff: // Flux's fields are withheld from both sides before the diff (EditDiff)
		r.Lines = s.agentView().DiffLines(r.Lines)
		return r
	case ApplyResult:
		r.Text = s.agentView().YAML(r.Text)
		return r
	case RevertResult:
		r.Text = s.agentView().YAML(r.Text)
		return r
	case AgentResult:
		return r.ForAgent(s.agentView())
	}
	return res
}

// Text returns s with the cached Secrets' values replaced.
func (v *AgentView) Text(s string) string {
	if v == nil || v.m == nil {
		return s
	}
	out, _ := v.m.scrub(s)
	return out
}

// Lines returns lines with the cached Secrets' values replaced; lines itself when none held one.
func (v *AgentView) Lines(lines []string) []string {
	if v == nil || v.m == nil {
		return lines
	}
	var out []string
	for i, l := range lines {
		if t, changed := v.m.scrub(l); changed {
			if out == nil {
				out = slices.Clone(lines)
			}
			out[i] = t
		}
	}
	if out == nil {
		return lines
	}
	return out
}

// DiffLines returns diff lines with the cached Secrets' values replaced.
func (v *AgentView) DiffLines(lines []DiffLine) []DiffLine {
	if v == nil || v.m == nil {
		return lines
	}
	out := slices.Clone(lines)
	for i := range out {
		out[i].Text = v.Text(out[i].Text)
	}
	return out
}

// YAML returns manifest text with Flux's credential fields withheld and the cached Secrets'
// values replaced. A text that does not parse is withheld whole when it names a Flux kind that
// carries such fields, since what it holds cannot be told; else only its values are replaced.
func (v *AgentView) YAML(text string) string {
	return v.Text(withholdFluxText(text))
}

// TabText is YAML for an editor tab's text, Text for a Helm Install or Upgrade tab's, whose text
// is a chart's values, not manifests.
func (v *AgentView) TabText(t DockType, text string) string {
	if HelmValuesTab(t) {
		return v.Text(text)
	}
	return v.YAML(text)
}

// fluxCredentialKindLine finds a kind line of a Flux kind whose YAML can hold a credential.
var fluxCredentialKindLine = regexp.MustCompile(`(?m)^\s*kind:\s*["']?(HelmRelease|Provider|Receiver|GitRepository|HelmRepository|OCIRepository|Bucket)\b`)

// withholdFluxText withholds Flux's credential fields in manifest text (one object or several),
// written again as YAML when any was withheld; the text as it is otherwise.
func withholdFluxText(text string) string {
	if !strings.Contains(text, "toolkit.fluxcd.io") {
		return text
	}
	objs, err := ParseManifests(text)
	if err != nil {
		if fluxCredentialKindLine.MatchString(text) {
			return model.Withheld // what it holds cannot be told
		}
		return text
	}
	changed := false
	docs := make([]string, 0, len(objs))
	for _, o := range objs {
		w, c := kinds.WithholdFluxCredentials(o)
		changed = changed || c
		y, err := yaml.Marshal(w.Object)
		if err != nil {
			return model.Withheld
		}
		docs = append(docs, string(y))
	}
	if !changed {
		return text
	}
	return strings.Join(docs, "---\n")
}

// withholdFluxObject is kinds.WithholdFluxCredentials for an agent's origin, obj itself otherwise.
func withholdFluxObject(ctx context.Context, obj *unstructured.Unstructured) *unstructured.Unstructured {
	if !OriginOf(ctx).IsAgent() {
		return obj
	}
	out, _ := kinds.WithholdFluxCredentials(obj)
	return out
}

// Detail filters a drawer's model for an agent, in place: its YAML (Flux's fields, the cached
// Secrets' values), its sections' values, chips and table cells, and its events' messages.
// Sections and events are copied before they change (the model may share them). The drawer never
// shows kubectl's last-applied annotation, which repeats Flux's fields, and Flux's drawers never
// show the fields themselves (TestFluxKeepsSecretsOut).
func (v *AgentView) Detail(m *DetailModel) {
	if m == nil {
		return
	}
	m.YAML = v.YAML(m.YAML)
	if v == nil || v.m == nil {
		return
	}
	m.Sections = v.sections(m.Sections)
	cloned := false
	for i := range m.Events {
		if t, changed := v.m.scrub(m.Events[i].Message); changed {
			if !cloned {
				m.Events, cloned = slices.Clone(m.Events), true
			}
			m.Events[i].Message = t
		}
	}
}

// sections returns sections with the cached Secrets' values replaced in their items' values and
// chips and their tables' cells, copied where anything changed.
func (v *AgentView) sections(in []model.DetailSection) []model.DetailSection {
	out := in
	copied := false
	for i, sec := range in {
		next, changed := v.section(sec)
		if !changed {
			continue
		}
		if !copied {
			out, copied = slices.Clone(in), true
		}
		out[i] = next
	}
	return out
}

func (v *AgentView) section(sec model.DetailSection) (model.DetailSection, bool) {
	changed := false
	text := func(s string) string {
		t, c := v.m.scrub(s)
		changed = changed || c
		return t
	}
	items := make([]model.DetailItem, len(sec.Items))
	for j, it := range sec.Items {
		it.Value = text(it.Value)
		if len(it.Chips) > 0 {
			chips := slices.Clone(it.Chips)
			for k := range chips {
				chips[k].Text = text(chips[k].Text)
			}
			it.Chips = chips
		}
		items[j] = it
	}
	var table *model.DetailTable
	if sec.Table != nil {
		t := *sec.Table
		t.Rows = make([][]string, len(sec.Table.Rows))
		for r, row := range sec.Table.Rows {
			t.Rows[r] = make([]string, len(row))
			for c, cell := range row {
				t.Rows[r][c] = text(cell)
			}
		}
		if sec.Table.Cells != nil {
			t.Cells = make([][]model.Cell, len(sec.Table.Cells))
			for r, row := range sec.Table.Cells {
				t.Cells[r] = slices.Clone(row)
				for c := range t.Cells[r] {
					t.Cells[r][c].Text = text(t.Cells[r][c].Text)
				}
			}
		}
		table = &t
	}
	if !changed {
		return sec, false
	}
	sec.Items, sec.Table = items, table
	return sec, true
}
