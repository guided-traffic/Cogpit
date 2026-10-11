package commands_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/model"
)

// FuzzEditYAML feeds the YAML that editor.diff, object.dryRun, object.apply and object.create
// parse with adversarial text, through Dispatch: every command returns a result or an error,
// never a panic; an update never carries another resourceVersion than the edit's base; nothing
// with a redacted placeholder or of a redacted kind is ever written.
func FuzzEditYAML(f *testing.F) {
	for _, s := range []string{
		"apiVersion: v1\nkind: Service\nmetadata:\n  name: checkout\n  namespace: shop\nspec:\n  sessionAffinity: ClientIP\n",
		"apiVersion: v1\nkind: Service\nmetadata:\n  name: checkout\n  namespace: shop\n  resourceVersion: \"1\"\n",
		"apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: x\n---\napiVersion: v1\nkind: Secret\nmetadata:\n  name: s\ndata:\n  k: <redacted>\n",
		"apiVersion: v1\nkind: List\nitems:\n- apiVersion: v1\n  kind: ConfigMap\n  metadata: {name: y, namespace: infra}\n",
		"kind: [", "", "null", "a: &a [*a]", "apiVersion: v1\nkind: Service\nmetadata: []\n",
	} {
		for cmd := uint8(0); cmd < 4; cmd++ {
			f.Add(cmd, s)
		}
	}
	catcher := &panicCatcher{}
	h := newHarness(f, seed(), engine.Options{
		Logger: slog.New(catcher), Audit: io.Discard,
		Confirmer: engine.ConfirmFunc(func(context.Context, engine.ConfirmRequest) (bool, error) { return true, nil }),
	})
	srv := serveObjects(h)
	if _, err := h.s.Objects(context.Background(), "services", nil); err != nil {
		f.Fatal(err)
	}
	var opened commands.EditResult
	h.must(f, engine.OriginGUI, "object.edit", map[string]any{"ref": model.ObjectRef{Kind: "services", Namespace: "shop", Name: "checkout"}}, &opened)
	ids := []string{"editor.diff", "object.dryRun", "object.apply", "object.create"}

	f.Fuzz(func(t *testing.T, cmd uint8, text string) {
		id := ids[int(cmd)%len(ids)]
		params := map[string]any{"id": opened.Tab.ID, "yaml": text}
		if id == "object.create" {
			params = map[string]any{"yaml": text}
		}
		raw, err := json.Marshal(params)
		if err != nil {
			t.Skip()
		}
		var st engine.EditorState
		h.must(t, engine.OriginGUI, "editor.get", map[string]any{"id": opened.Tab.ID}, &st)
		srv.mu.Lock()
		before := len(srv.updates)
		srv.mu.Unlock()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		res, err := h.s.Dispatch(ctx, engine.Invocation{ID: id, Params: raw, Origin: engine.OriginExternalAgent})
		if msgs := catcher.take(); len(msgs) > 0 {
			t.Fatalf("%s %q: internal error: %v", id, text, msgs)
		}
		if err == nil && res == nil {
			t.Fatalf("%s %q: no result and no error", id, text)
		}
		srv.mu.Lock()
		defer srv.mu.Unlock()
		for _, u := range srv.updates[before:] {
			if u.GetResourceVersion() != st.BaseResourceVersion {
				t.Fatalf("%s sent resourceVersion %q, the base was %q", id, u.GetResourceVersion(), st.BaseResourceVersion)
			}
		}
		for _, c := range srv.creates {
			if engine.RedactedKind(c.GetAPIVersion(), c.GetKind()) {
				t.Fatalf("a %s was created from %q", c.GetKind(), text)
			}
		}
	})
}
