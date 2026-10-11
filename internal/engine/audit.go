package engine

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"

	"github.com/guided-traffic/cogpit/internal/model"
)

// AuditEntry is one line of the audit log: an agent-initiated command and its outcome.
type AuditEntry struct {
	Time    time.Time       `json:"time"`
	Origin  Origin          `json:"origin"`
	Command string          `json:"command"`
	Params  json.RawMessage `json:"params,omitempty"`
	// Context is the context the command was dispatched for, or the one the user confirmed.
	Context string `json:"context,omitempty"`
	// Phase is "requested" for the line a write gets before it runs, "done" for the outcome.
	Phase string `json:"phase"`
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

// Audit phases.
const (
	auditRequested = "requested"
	auditDone      = "done"
)

// audit writes one JSON line for an agent-initiated command. Results are not logged: they can be
// large and may carry cluster data; the parameters and the outcome are what the user reviews.
func (s *Session) audit(inv Invocation, context, phase string, err error) {
	if s.opts.Audit == nil {
		return
	}
	e := AuditEntry{
		Time: s.opts.Now().UTC(), Origin: inv.Origin, Command: inv.ID,
		Params: normalizeParams(inv.Params), Context: context, Phase: phase, OK: err == nil,
	}
	if !json.Valid(e.Params) {
		e.Params = nil
	}
	e.Params = auditParams(e.Params)
	if err != nil {
		e.Error = err.Error()
	}
	line, mErr := json.Marshal(e)
	if mErr != nil {
		return
	}
	s.auditMu.Lock()
	defer s.auditMu.Unlock()
	if _, wErr := s.opts.Audit.Write(append(line, '\n')); wErr != nil {
		s.log.Warn("audit log write failed", "err", wErr)
	}
}

// auditParams keeps Secret values out of the audit log. Commands that take manifests (object.apply,
// object.create, editor.setText, resource.createTab) carry them in a "yaml" parameter; an agent may
// send a Secret's manifest even though Cogpit refuses to write Secrets. Its documents are logged
// with every Secret's data, stringData and last-applied annotation replaced by model.Redacted; text
// that does not parse as manifests is logged by its size only, since it may hold anything.
func auditParams(raw json.RawMessage) json.RawMessage {
	var params map[string]any
	if json.Unmarshal(raw, &params) != nil {
		return raw
	}
	text, ok := params["yaml"].(string)
	if !ok || strings.TrimSpace(text) == "" {
		return raw
	}
	objs, err := ParseManifests(text)
	if err != nil {
		params["yaml"] = fmt.Sprintf("<%d bytes that are not manifests; not logged>", len(text))
		return marshalParams(params, raw)
	}
	secret := false
	for _, o := range objs {
		secret = secret || o.GetKind() == "Secret"
	}
	if !secret {
		return raw
	}
	docs := make([]string, 0, len(objs))
	for _, o := range objs {
		if o.GetKind() == "Secret" {
			o = redactSecret(o)
		}
		y, err := yaml.Marshal(o.Object)
		if err != nil {
			params["yaml"] = fmt.Sprintf("<%d bytes holding a Secret; not logged>", len(text))
			return marshalParams(params, raw)
		}
		docs = append(docs, string(y))
	}
	params["yaml"] = strings.Join(docs, "---\n")
	return marshalParams(params, raw)
}

// redactSecret replaces a Secret manifest's values, whatever its apiVersion says and whatever
// shape its data has: data and stringData entries (or the whole field, if it is not a mapping)
// and the last-applied annotation, which repeats them.
func redactSecret(o *unstructured.Unstructured) *unstructured.Unstructured {
	cp := o.DeepCopy()
	for _, field := range []string{"data", "stringData"} {
		v, ok := cp.Object[field]
		if !ok {
			continue
		}
		m, isMap := v.(map[string]any)
		if !isMap {
			cp.Object[field] = model.Redacted
			continue
		}
		for k := range m {
			m[k] = model.Redacted
		}
	}
	if md, ok := cp.Object["metadata"].(map[string]any); ok {
		if ann, ok := md["annotations"].(map[string]any); ok {
			if _, found := ann["kubectl.kubernetes.io/last-applied-configuration"]; found {
				ann["kubectl.kubernetes.io/last-applied-configuration"] = model.Redacted
			}
		}
	}
	return cp
}

func marshalParams(params map[string]any, fallback json.RawMessage) json.RawMessage {
	out, err := json.Marshal(params)
	if err != nil {
		return fallback
	}
	return out
}
