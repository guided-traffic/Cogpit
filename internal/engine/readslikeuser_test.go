package engine

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
)

// readsProbe is what a command sees of who reads: the probe command's result.
type readsProbe struct {
	Origin        Origin `json:"origin"`
	ReadsLikeUser bool   `json:"readsLikeUser"`
	ReadsAsAgent  bool   `json:"readsAsAgent"`
	PrivateTurn   bool   `json:"privateTurn"`
	// Nested is what a command dispatched with the outer command's context and another origin
	// sees (nil when the probe dispatched none).
	Nested *readsProbe `json:"nested,omitempty"`
}

type readsProbeParams struct {
	// ReadsLikeUser is a parameter of the same name: a caller's JSON lands here, never in the
	// Invocation.
	ReadsLikeUser bool `json:"readsLikeUser,omitempty"`
	// Nest dispatches the probe again with the running command's context and this origin.
	Nest Origin `json:"nest,omitempty"`
	// Fail makes the probe fail with a UserOnlyError whose text quotes a value.
	Fail bool `json:"fail,omitempty"`
}

const probeLookedUp = "LOOKED-UP-SECRET-DATA-4711"

func probeSession(t *testing.T) (*Session, *lockedBuffer) {
	t.Helper()
	audit := &lockedBuffer{}
	s := New(Options{Audit: audit})
	err := Register(s.Registry(), Spec[readsProbeParams]{
		ID: "probe.reads", Label: "Reads Probe", Danger: DangerRead,
		Run: func(ctx context.Context, s *Session, p readsProbeParams) (any, error) {
			if p.Fail {
				return nil, &UserOnlyError{Err: errors.New("template: name " + probeLookedUp + " is not valid"), Withheld: "the probe failed"}
			}
			out := readsProbe{Origin: OriginOf(ctx), ReadsLikeUser: ReadsLikeUser(ctx), ReadsAsAgent: ReadsAsAgent(ctx), PrivateTurn: PrivateTurn(ctx)}
			if p.Nest != "" {
				res, err := s.Dispatch(ctx, Invocation{ID: "probe.reads", Params: json.RawMessage(`{}`), Origin: p.Nest})
				if err != nil {
					return nil, err
				}
				n := res.(readsProbe)
				out.Nested = &n
			}
			return out, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return s, audit
}

// TestReadsLikeUserOnlyForTheBuiltInAgent (D-10, H-94): Dispatch carries Invocation.ReadsLikeUser
// into the command's context for the built-in agent only; the same mark from an external agent,
// the CLI or a link is ignored, and the window reads like the user by being the window. A nested
// Dispatch with another origin does not inherit the mark. The audit log marks the private turn's
// lines only.
func TestReadsLikeUserOnlyForTheBuiltInAgent(t *testing.T) {
	s, audit := probeSession(t)
	cases := []struct {
		origin Origin
		mark   bool
		want   readsProbe
	}{
		{OriginBuiltInAgent, true, readsProbe{Origin: OriginBuiltInAgent, ReadsLikeUser: true, PrivateTurn: true}},
		{OriginBuiltInAgent, false, readsProbe{Origin: OriginBuiltInAgent, ReadsAsAgent: true}},
		{OriginExternalAgent, true, readsProbe{Origin: OriginExternalAgent, ReadsAsAgent: true}},
		{OriginExternalAgent, false, readsProbe{Origin: OriginExternalAgent, ReadsAsAgent: true}},
		{OriginCLI, true, readsProbe{Origin: OriginCLI}},
		{OriginGUI, true, readsProbe{Origin: OriginGUI, ReadsLikeUser: true}},
		{OriginGUI, false, readsProbe{Origin: OriginGUI, ReadsLikeUser: true}},
	}
	for _, c := range cases {
		res, err := s.Dispatch(context.Background(), Invocation{ID: "probe.reads", Params: json.RawMessage(`{}`), Origin: c.origin, ReadsLikeUser: c.mark})
		if err != nil {
			t.Fatalf("%s (mark %v): %v", c.origin, c.mark, err)
		}
		if got := res.(readsProbe); got != c.want {
			t.Errorf("%s (mark %v): %+v, want %+v", c.origin, c.mark, got, c.want)
		}
	}

	// A command of the private turn that dispatches another origin's command passes its context on:
	// the inner command does not read like the user.
	res, err := s.Dispatch(context.Background(), Invocation{ID: "probe.reads", Params: json.RawMessage(`{"nest":"externalAgent"}`),
		Origin: OriginBuiltInAgent, ReadsLikeUser: true})
	if err != nil {
		t.Fatal(err)
	}
	if n := res.(readsProbe).Nested; n == nil || n.ReadsLikeUser || n.PrivateTurn || !n.ReadsAsAgent {
		t.Errorf("the nested external agent's command: %+v", n)
	}

	var private, agents int
	for _, line := range strings.Split(strings.TrimSpace(audit.String()), "\n") {
		var e AuditEntry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatalf("audit line %q: %v", line, err)
		}
		agents++
		if e.ReadsLikeUser {
			private++
			if e.Origin != OriginBuiltInAgent {
				t.Errorf("a %s line marked readsLikeUser: %s", e.Origin, line)
			}
		}
	}
	// Five agent calls of the table and the private turn's nesting call with its nested external
	// one; the private turn's two lines are marked.
	if agents != 6 || private != 2 {
		t.Errorf("%d agent lines, %d marked; want 6 and 2:\n%s", agents, private, audit.String())
	}
}

// TestReadsLikeUserNeverFromJSON (H-94): the mark is not part of an Invocation's JSON, so no MCP
// call, bridge or CLI request that is decoded into one can set it, under any spelling; a parameter
// of that name is the command's parameter, not the mark.
func TestReadsLikeUserNeverFromJSON(t *testing.T) {
	for _, body := range []string{
		`{"id":"probe.reads","origin":"builtInAgent","readsLikeUser":true}`,
		`{"id":"probe.reads","origin":"builtInAgent","ReadsLikeUser":true}`,
		`{"id":"probe.reads","origin":"builtInAgent","readslikeuser":true,"-":true}`,
	} {
		var inv Invocation
		if err := json.Unmarshal([]byte(body), &inv); err != nil {
			t.Fatal(err)
		}
		if inv.ReadsLikeUser || inv.Origin != OriginBuiltInAgent {
			t.Errorf("%s decoded to %+v", body, inv)
		}
	}
	out, err := json.Marshal(Invocation{ID: "probe.reads", Origin: OriginBuiltInAgent, ReadsLikeUser: true})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(string(out)), "readslikeuser") {
		t.Errorf("the mark is serialised: %s", out)
	}

	s, _ := probeSession(t)
	res, err := s.Dispatch(context.Background(), Invocation{ID: "probe.reads", Params: json.RawMessage(`{"readsLikeUser":true}`), Origin: OriginBuiltInAgent})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.(readsProbe); got.ReadsLikeUser || got.PrivateTurn {
		t.Errorf("a parameter named readsLikeUser set the mark: %+v", got)
	}
}

// TestUserOnlyErrorStaysOutOfTheAuditLog (H-94): a private turn reads a failed Helm action's error
// in full, as the window does; its audit line records what failed, never the text, which can quote
// what a chart's lookup read.
func TestUserOnlyErrorStaysOutOfTheAuditLog(t *testing.T) {
	s, audit := probeSession(t)
	_, err := s.Dispatch(context.Background(), Invocation{ID: "probe.reads", Params: json.RawMessage(`{"fail":true}`), Origin: OriginBuiltInAgent, ReadsLikeUser: true})
	var uo *UserOnlyError
	if err == nil || !strings.Contains(err.Error(), probeLookedUp) || !errors.As(err, &uo) {
		t.Fatalf("the private turn's error: %v", err)
	}
	line := audit.String()
	if strings.Contains(line, probeLookedUp) || !strings.Contains(line, `"error":"the probe failed"`) || !strings.Contains(line, `"readsLikeUser":true`) {
		t.Errorf("audit line %s", line)
	}
}

// TestReadsLikeUserConcurrentTurns: private and other turns dispatched at once each see their own
// mark (run with -race).
func TestReadsLikeUserConcurrentTurns(t *testing.T) {
	s, _ := probeSession(t)
	var wg sync.WaitGroup
	errs := make(chan string, 64)
	for i := range 64 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mark := i%2 == 0
			origin := OriginBuiltInAgent
			if i%4 == 1 {
				origin = OriginExternalAgent
			}
			res, err := s.Dispatch(context.Background(), Invocation{ID: "probe.reads", Params: json.RawMessage(`{}`), Origin: origin, ReadsLikeUser: mark})
			if err != nil {
				errs <- err.Error()
				return
			}
			if got, want := res.(readsProbe).PrivateTurn, mark && origin == OriginBuiltInAgent; got != want {
				errs <- "a call saw another call's mark"
			}
		}()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Error(e)
	}
}
