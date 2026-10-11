package commands_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
)

// panicCatcher is a slog handler that remembers every error record: the engine logs a recovered
// panic at error level, and nothing else at that level.
type panicCatcher struct {
	mu   sync.Mutex
	msgs []string
}

func (p *panicCatcher) Enabled(_ context.Context, l slog.Level) bool { return l >= slog.LevelError }

func (p *panicCatcher) Handle(_ context.Context, r slog.Record) error {
	var b strings.Builder
	b.WriteString(r.Message)
	r.Attrs(func(a slog.Attr) bool {
		if a.Key != "stack" {
			fmt.Fprintf(&b, " %s=%v", a.Key, a.Value)
		}
		return true
	})
	p.mu.Lock()
	p.msgs = append(p.msgs, b.String())
	p.mu.Unlock()
	return nil
}

func (p *panicCatcher) WithAttrs([]slog.Attr) slog.Handler { return p }
func (p *panicCatcher) WithGroup(string) slog.Handler      { return p }

func (p *panicCatcher) take() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := p.msgs
	p.msgs = nil
	return out
}

// FuzzDispatch feeds every command adversarial parameters from every origin: a command must
// return an error, never panic (testing.md, command fuzzing).
func FuzzDispatch(f *testing.F) {
	r := engine.NewRegistry()
	if err := commands.RegisterAll(r); err != nil {
		f.Fatal(err)
	}
	var ids []string
	for _, c := range r.Commands() {
		ids = append(ids, c.ID)
	}
	seeds := []string{
		`{}`, `null`, ``, `[]`, `{"ref":{"kind":"pods","namespace":"shop","name":"web-0"}}`,
		`{"ref":{"kind":"deployments","namespace":"infra","name":"web"}}`,
		`{"ref":{"kind":"namespaces","name":"shop"}}`, `{"ref":{"kind":"pods","name":""}}`,
		`{"view":"services"}`, `{"view":""}`, `{"namespaces":["shop","",null]}`, `{"filter":"\u0000web"}`,
		`{"column":"name","desc":true}`, `{"column":""}`, `{"refs":[{"kind":"pods","namespace":"shop","name":"web-1"}]}`,
		`{"kind":"services","field":"clusterIP","value":"10.96.100.100"}`, `{"field":"label","value":"","match":"prefix","limit":1000}`,
		`{"kind":"pods","limit":-5}`, `{"context":"test"}`, `{"context":"broken"}`, `{"id":"logs-1"}`,
		`{"ref":{"kind":"pods","namespace":"shop","name":"web-0"},"container":"app"}`,
		`{"ref":{"context":"prod","kind":"pods","namespace":"shop","name":"web-0"}}`,
		`{"yaml":"apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: x\n"}`, `{"id":"edit-1","yaml":"kind: ["}`,
		`{"template":"Pod"}`, `{"ref":{"kind":"configmaps","namespace":"shop","name":"x"}}`,
	}
	for i := range ids {
		for j, s := range seeds {
			if (i+j)%4 == 0 {
				f.Add(uint8(i), []byte(s), uint8(j))
			}
		}
	}

	catcher := &panicCatcher{}
	h := newHarness(f, seed(), engine.Options{
		Logger: slog.New(catcher), Audit: io.Discard,
		Confirmer: engine.ConfirmFunc(func(context.Context, engine.ConfirmRequest) (bool, error) { return true, nil }),
	})
	origins := []engine.Origin{engine.OriginGUI, engine.OriginCLI, engine.OriginExternalAgent, engine.OriginBuiltInAgent}

	f.Fuzz(func(t *testing.T, idx uint8, params []byte, origin uint8) {
		if h.s.Backend() == nil { // a fuzzed cluster.connect failed: keep exercising a connected session
			if err := h.s.Connect(context.Background(), "test"); err != nil {
				t.Fatal(err)
			}
		}
		id := ids[int(idx)%len(ids)]
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		res, err := h.s.Dispatch(ctx, engine.Invocation{ID: id, Params: params, Origin: origins[int(origin)%len(origins)]})
		if msgs := catcher.take(); len(msgs) > 0 {
			t.Fatalf("%s %q: internal error: %v", id, params, msgs)
		}
		if err == nil && res == nil {
			t.Fatalf("%s %q: no result and no error", id, params)
		}
	})
}
