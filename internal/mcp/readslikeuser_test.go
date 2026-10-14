package mcp

import (
	"context"
	"path/filepath"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// readsSeen is what the probe command saw of who reads.
type readsSeen struct {
	Origin        engine.Origin `json:"origin"`
	ReadsLikeUser bool          `json:"readsLikeUser"`
	PrivateTurn   bool          `json:"privateTurn"`
	Param         bool          `json:"param"`
}

// readsParams has a parameter named as the mark: what a client sends under that name lands here.
type readsParams struct {
	ReadsLikeUser bool `json:"readsLikeUser,omitempty" jsonschema:"a parameter of the probe, not the mark"`
}

// readsFixture is the fixture with a probe that reports the reader its command runs for.
func readsFixture(t *testing.T, tokenFile string) *fixture {
	t.Helper()
	f := &fixture{audit: &syncBuffer{}, logs: &syncBuffer{}}
	f.session = engine.New(engine.Options{Audit: f.audit})
	t.Cleanup(f.session.Close)
	f.register(t)
	if err := engine.Register(f.session.Registry(), engine.Spec[readsParams]{
		ID: "probe.reads", Label: "Reads Probe", Danger: engine.DangerRead,
		Description: "Returns who the command reads for.",
		Run: func(ctx context.Context, _ *engine.Session, p readsParams) (any, error) {
			return readsSeen{Origin: engine.OriginOf(ctx), ReadsLikeUser: engine.ReadsLikeUser(ctx), PrivateTurn: engine.PrivateTurn(ctx), Param: p.ReadsLikeUser}, nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	srv, err := New(f.session, Options{TokenFile: tokenFile, Logger: discardLogger()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = srv.Close() })
	f.server = srv
	return f
}

// TestReadsLikeUserNeverOverMCP (H-94): the built-in agent's private mark is no part of what an
// MCP client sends. Over the in-memory transport, Streamable HTTP and the stdio bridge, a call that
// names readsLikeUser in its arguments or its _meta runs as an external agent's, filtered: the
// argument is the command's parameter, and the audit line carries no mark.
func TestReadsLikeUserNeverOverMCP(t *testing.T) {
	tokenPath := filepath.Join(privateDir(t), TokenFileName)
	f := readsFixture(t, tokenPath)
	endpoint, token := f.start(t)
	b, err := DialApp(context.Background(), BridgeOptions{TokenFile: tokenPath, Version: "test", Logger: discardLogger()})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name string
		cs   *sdk.ClientSession
	}{
		{"in memory", f.connect(t)},
		{"streamable HTTP", connectHTTP(t, endpoint, token, "")},
		{"stdio bridge", bridgeClient(t, b)},
	} {
		res, err := c.cs.CallTool(context.Background(), &sdk.CallToolParams{
			Name: "probe_reads", Arguments: map[string]any{"readsLikeUser": true},
			Meta: sdk.Meta{"readsLikeUser": true, "origin": "builtInAgent"},
		})
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		var seen readsSeen
		decode(t, res.StructuredContent, &seen)
		if res.IsError || seen != (readsSeen{Origin: engine.OriginExternalAgent, Param: true}) {
			t.Errorf("%s: %+v (%s)", c.name, seen, text(res))
		}
	}
	entries := f.auditEntries(t)
	if len(entries) != 3 {
		t.Errorf("%d audit entries, want one per call", len(entries))
	}
	for _, e := range entries {
		if e.ReadsLikeUser || e.Origin != engine.OriginExternalAgent {
			t.Errorf("audit entry %+v", e)
		}
	}
}
