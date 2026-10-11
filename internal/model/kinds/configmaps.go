package kinds

import (
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
)

// Caps on what the drawer shows of a Config Map's data, so that a large one (up to 1 MiB, with
// thousands of keys) cannot blow up the drawer, which renders one text line per line of a value.
// The YAML view has everything.
const (
	configMapValueBytes = 4 << 10  // bytes shown of one value
	configMapValueLines = 40       // lines shown of one value
	configMapTotalBytes = 32 << 10 // bytes shown of all values together
	configMapTotalLines = 400      // lines shown of all values together
	configMapItems      = 100      // keys listed per section
)

// configMaps has the columns of Lens's documentation (not captured). A Config Map has no state,
// so no status token (inferred).
var configMaps = &model.KindProfile{
	Key:        "configmaps",
	Version:    "v1",
	Resource:   "configmaps",
	Kind:       "ConfigMap",
	Title:      "Config Maps",
	Namespaced: true,
	Columns: []model.Column{
		nameColumn,
		namespaceColumn,
		{ID: "keys", Title: "Keys", Type: model.ColumnText},
		ageColumn,
	},
	Cells: configMapCells,
	Fields: map[string]fieldFunc{
		"key": func(obj *unstructured.Unstructured) []string {
			return configMapKeys(obj)
		},
	},
	Detail: configMapDetail,
}

func configMapCells(obj *unstructured.Unstructured) ([]model.Cell, model.Status) {
	return []model.Cell{
		nameCell(obj),
		namespaceCell(obj),
		{Text: strings.Join(configMapKeys(obj), ", ")},
		model.AgeCell(created(obj)),
	}, model.StatusNone
}

// configMapKeys returns the keys of data and binaryData, sorted. The API server keeps the two
// disjoint; a key in both is listed once.
func configMapKeys(obj any) []string {
	data, binary := model.NestedMap(obj, "data"), model.NestedMap(obj, "binaryData")
	if len(data)+len(binary) == 0 {
		return nil
	}
	keys := make([]string, 0, len(data)+len(binary))
	for k := range data {
		keys = append(keys, k)
	}
	for k := range binary {
		if _, dup := data[k]; !dup {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return keys
}

func configMapDetail(obj *unstructured.Unstructured) []model.DetailSection {
	var sections []model.DetailSection
	if immutable, _ := model.NestedBool(obj, "immutable"); immutable {
		sections = withSection(sections, "Properties", detailItems{{Name: "Immutable", Value: "true"}})
	}

	data := model.NestedMap(obj, "data")
	budget := valueBudget{bytes: configMapTotalBytes, lines: configMapTotalLines}
	var items detailItems
	for i, k := range sortedKeys(data) {
		if i == configMapItems {
			items = append(items, moreKeys(len(data)-i))
			break
		}
		// Appended directly: an empty value is a value, and the drawer shows its key.
		items = append(items, model.DetailItem{Name: k, Value: budget.take(model.NestedString(data, k))})
	}
	sections = withSection(sections, "Data", items)

	// Binary values are shown by their size only.
	binary := model.NestedMap(obj, "binaryData")
	var bin detailItems
	for i, k := range sortedKeys(binary) {
		if i == configMapItems {
			bin = append(bin, moreKeys(len(binary)-i))
			break
		}
		bin = append(bin, model.DetailItem{Name: k, Value: formatBytes(base64Size(model.NestedString(binary, k)))})
	}
	return withSection(sections, "Binary Data", bin)
}

// moreKeys is the last item of a section that lists only the first configMapItems keys.
func moreKeys(n int) model.DetailItem {
	noun := " more keys"
	if n == 1 {
		noun = " more key"
	}
	return model.DetailItem{Value: strconv.Itoa(n) + noun + ": see the YAML"}
}

// valueBudget is what the drawer may still show of a Config Map's values.
type valueBudget struct{ bytes, lines int }

// take returns as much of a value as the caps allow, cut at a line or character boundary, with a
// note saying how much was cut. Once the total budget is down to a quarter of one value's cap, a
// value that does not fit is not shown at all rather than as a scrap. A file's final newline is
// dropped: the drawer would show it as an empty line.
func (b *valueBudget) take(v string) string {
	v = strings.TrimSuffix(v, "\n")
	lines := strings.Count(v, "\n") + 1
	if (len(v) > b.bytes && b.bytes < configMapValueBytes/4) || (lines > b.lines && b.lines < configMapValueLines/4) {
		return "[not shown: " + formatBytes(int64(len(v))) + "; the full value is in the YAML]"
	}
	maxBytes, maxLines := min(configMapValueBytes, b.bytes), min(configMapValueLines, b.lines)
	shown := v
	if len(shown) > maxBytes {
		cut := maxBytes
		for cut > 0 && !utf8.RuneStart(shown[cut]) {
			cut--
		}
		shown = shown[:cut]
	}
	if i := nthIndex(shown, '\n', maxLines); i >= 0 {
		shown = shown[:i]
	}
	b.bytes -= len(shown)
	b.lines -= strings.Count(shown, "\n") + 1
	if len(shown) == len(v) {
		return v
	}
	return shown + "\n[cut: " + formatBytes(int64(len(shown))) + " of " + formatBytes(int64(len(v))) +
		" shown; the full value is in the YAML]"
}

// nthIndex returns the index of the n-th occurrence of c in s, or -1.
func nthIndex(s string, c byte, n int) int {
	off := 0
	for range n {
		i := strings.IndexByte(s[off:], c)
		if i < 0 {
			return -1
		}
		off += i + 1
	}
	return off - 1
}

// base64Size returns the decoded length of a base64 value (binaryData) without decoding it.
func base64Size(s string) int64 {
	n := len(s)
	pad := 0
	for pad < 2 && n-pad > 0 && s[n-1-pad] == '=' {
		pad++
	}
	return int64((n - pad) * 3 / 4)
}

// formatBytes renders a size: "512 bytes", "1 byte", "4.0 KiB", "1.5 MiB".
func formatBytes(n int64) string {
	switch {
	case n == 1:
		return "1 byte"
	case n < 1<<10:
		return strconv.FormatInt(n, 10) + " bytes"
	case n < 1<<20:
		return strconv.FormatFloat(float64(n)/(1<<10), 'f', 1, 64) + " KiB"
	default:
		return strconv.FormatFloat(float64(n)/(1<<20), 'f', 1, 64) + " MiB"
	}
}
