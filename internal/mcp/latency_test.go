package mcp

import (
	"context"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestToolCallLatency shows that an agent's cache-only read is cheap: the MCP layer adds little
// to a command that answers from memory. The bound is loose enough for the race detector.
func TestToolCallLatency(t *testing.T) {
	f := newFixture(t, nil, Options{})
	for name, cs := range map[string]*sdk.ClientSession{
		"in-memory": f.connect(t),
		"http":      func() *sdk.ClientSession { u, tok := f.start(t); return connectHTTP(t, u, tok, "") }(),
	} {
		const n = 200
		call(t, cs, "ui_state", nil) // warm up
		start := time.Now()
		for range n {
			if res := call(t, cs, "ui_state", nil); res.IsError {
				t.Fatal(text(res))
			}
		}
		mean := time.Since(start) / n
		t.Logf("%s: ui_state takes %v per call (mean of %d)", name, mean, n)
		if mean > 50*time.Millisecond {
			t.Errorf("%s: %v per cache-only call", name, mean)
		}
	}
}

func BenchmarkToolCallInMemory(b *testing.B) {
	f := newFixture(b, nil, Options{Logger: discardLogger()})
	benchmarkCalls(b, f.connect(b))
}

func BenchmarkToolCallHTTP(b *testing.B) {
	f := newFixture(b, nil, Options{Logger: discardLogger()})
	u, tok := f.start(b)
	benchmarkCalls(b, connectHTTP(b, u, tok, ""))
}

func benchmarkCalls(b *testing.B, cs *sdk.ClientSession) {
	params := &sdk.CallToolParams{Name: "ui_state"}
	b.ReportAllocs()
	for b.Loop() {
		res, err := cs.CallTool(context.Background(), params)
		if err != nil || res.IsError {
			b.Fatal(err, res)
		}
	}
}
