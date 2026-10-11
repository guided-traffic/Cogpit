package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/guided-traffic/cogpit/internal/model"
)

type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

type manifestProbeParams struct {
	YAML string `json:"yaml"`
}

// TestAuditKeepsSecretValuesOut: an agent's manifest parameter is logged, but a Secret's values in
// it are not, whatever shape or apiVersion it claims; text that is not a manifest is logged by
// size only; other manifests stay as sent.
func TestAuditKeepsSecretValuesOut(t *testing.T) {
	audit := &lockedBuffer{}
	s := New(Options{Audit: audit})
	err := Register(s.Registry(), Spec[manifestProbeParams]{
		ID: "probe.create", Label: "Create", Danger: DangerRead,
		Run: func(context.Context, *Session, manifestProbeParams) (any, error) { return nil, nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, yaml string
		want       []string
		never      []string
	}{
		{"secret", "apiVersion: v1\nkind: Secret\nmetadata:\n  name: db\n  annotations:\n    kubectl.kubernetes.io/last-applied-configuration: '{\"data\":{\"password\":\"aHVudGVyMg==\"}}'\ndata:\n  password: aHVudGVyMg==\nstringData:\n  token: s3cr3t-token\n",
			[]string{"kind: Secret", "name: db", model.Redacted}, []string{"aHVudGVyMg==", "s3cr3t-token"}},
		{"secret among others", "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: cfg\ndata:\n  mode: fast\n---\napiVersion: v9\nkind: Secret\nmetadata:\n  name: odd\ndata: c2VjcmV0LWJsb2I=\n",
			[]string{"mode: fast", "name: odd"}, []string{"c2VjcmV0LWJsb2I="}},
		{"not a manifest", "password: hunter2\n: [unbalanced",
			[]string{"not manifests; not logged"}, []string{"hunter2"}},
		{"configmap", "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: cfg\ndata:\n  mode: fast\n",
			[]string{"mode: fast"}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := len(audit.String())
			params, _ := json.Marshal(manifestProbeParams{YAML: tc.yaml})
			if _, err := s.Dispatch(context.Background(), Invocation{ID: "probe.create", Params: params, Origin: OriginExternalAgent}); err != nil {
				t.Fatal(err)
			}
			line := audit.String()[before:]
			var e AuditEntry
			if err := json.Unmarshal([]byte(strings.TrimSpace(line)), &e); err != nil {
				t.Fatalf("audit line %q: %v", line, err)
			}
			var p manifestProbeParams
			if err := json.Unmarshal(e.Params, &p); err != nil {
				t.Fatalf("params %s: %v", e.Params, err)
			}
			for _, w := range tc.want {
				if !strings.Contains(p.YAML, w) {
					t.Errorf("logged yaml lacks %q:\n%s", w, p.YAML)
				}
			}
			for _, n := range tc.never {
				if strings.Contains(line, n) {
					t.Errorf("audit line holds %q: %s", n, line)
				}
			}
		})
	}
}
