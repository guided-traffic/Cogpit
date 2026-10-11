package helm

import (
	"regexp"
	"strconv"
	"strings"

	"sigs.k8s.io/yaml"

	"github.com/guided-traffic/cogpit/internal/model"
)

// A release's manifest renders its Secrets with their data. The drawer shows the manifest with
// every Secret's values replaced by model.Redacted, for the user and agents alike, as the Secrets
// drawer masks them (conventions: Secret data in the UI is hidden until the user reveals one
// field, and the manifest has no reveal).

// secretKindLine finds a "kind: Secret" (or SecretList) line in YAML or JSON text, also as the
// first key of a list item.
var secretKindLine = regexp.MustCompile(`(?m)^[\s-]*"?kind"?\s*:\s*["']?Secret(List)?["']?\s*,?\s*(#.*)?$|"kind"\s*:\s*"Secret(List)?"`)

// redactManifest returns a rendered manifest with the values of every Secret replaced by
// model.Redacted: Secret documents, and the Secret items of List documents. A document that names
// a Secret kind but does not parse is replaced by a note with its size. Every other document is
// kept as it is, its comments included.
func redactManifest(manifest string) string {
	if !strings.Contains(manifest, "Secret") {
		return manifest
	}
	docs := splitDocuments(manifest)
	for i, d := range docs {
		if strings.Contains(d.body, "Secret") {
			docs[i].body = redactDocument(d.body)
		}
	}
	var b strings.Builder
	b.Grow(len(manifest))
	for _, d := range docs {
		b.WriteString(d.sep)
		b.WriteString(d.body)
	}
	return b.String()
}

// document is one YAML document of a manifest: its separator line ("---\n", empty for the first)
// and its text.
type document struct{ sep, body string }

// splitDocuments splits a manifest at its document separators, lines that start with "---".
func splitDocuments(text string) []document {
	var docs []document
	cur := document{}
	for line := range strings.SplitAfterSeq(text, "\n") {
		if strings.HasPrefix(line, "---") {
			docs = append(docs, cur)
			cur = document{sep: line}
			continue
		}
		cur.body += line
	}
	return append(docs, cur)
}

// redactDocument redacts one document that mentions a Secret: it keeps the leading comment lines
// (Helm's "# Source: …") and writes the document again with the values replaced. A document that
// parses and holds no Secret is returned unchanged.
func redactDocument(body string) string {
	var comments strings.Builder
	rest := body
	for rest != "" {
		line, after, _ := strings.Cut(rest, "\n")
		if t := strings.TrimSpace(line); t != "" && !strings.HasPrefix(t, "#") {
			break
		}
		comments.WriteString(line + "\n")
		rest = after
	}
	var obj map[string]any
	if err := yaml.Unmarshal([]byte(rest), &obj); err != nil || obj == nil {
		if !secretKindLine.MatchString(rest) {
			return body
		}
		return comments.String() + "# [" + strconv.Itoa(len(rest)) + " bytes not shown: a Secret that Cogpit could not read]\n"
	}
	if !redactObject(obj) {
		return body
	}
	out, err := yaml.Marshal(obj)
	if err != nil {
		return comments.String() + "# [a Secret, not shown]\n"
	}
	return comments.String() + string(out)
}

// redactObject redacts a Secret, or the Secret items of a list, in place, and reports whether it
// held one.
func redactObject(obj map[string]any) bool {
	kind, _ := obj["kind"].(string)
	switch {
	case kind == "Secret":
		redactSecretValues(obj)
		return true
	case strings.HasSuffix(kind, "List"):
		items, _ := obj["items"].([]any)
		found := false
		for _, it := range items {
			m, ok := it.(map[string]any)
			if !ok {
				continue
			}
			if k, _ := m["kind"].(string); k == "Secret" || kind == "SecretList" {
				redactSecretValues(m)
				found = true
			}
		}
		return found
	}
	return false
}

// redactSecretValues replaces a Secret's data and stringData values (or the whole field, if it is
// not a mapping) and the last-applied annotation, which repeats them.
func redactSecretValues(obj map[string]any) {
	for _, field := range []string{"data", "stringData"} {
		v, ok := obj[field]
		if !ok {
			continue
		}
		m, isMap := v.(map[string]any)
		if !isMap {
			obj[field] = model.Redacted
			continue
		}
		for k := range m {
			m[k] = model.Redacted
		}
	}
	if md, ok := obj["metadata"].(map[string]any); ok {
		if ann, ok := md["annotations"].(map[string]any); ok {
			if _, found := ann["kubectl.kubernetes.io/last-applied-configuration"]; found {
				ann["kubectl.kubernetes.io/last-applied-configuration"] = model.Redacted
			}
		}
	}
}
