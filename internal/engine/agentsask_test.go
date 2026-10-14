package engine

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/guided-traffic/cogpit/internal/model"
)

type agentsAskProbeParams struct {
	Grant bool `json:"grant,omitempty"`
}

// TestNeedsConfirmationAgentsAsk is the D-2 table with the command's AgentsAsk (H-27): it makes an
// agent's mutating call ask whatever the allowance, and changes nothing for the GUI, the CLI, a
// read or a destructive command.
func TestNeedsConfirmationAgentsAsk(t *testing.T) {
	for _, c := range []struct {
		d               Danger
		o               Origin
		allowed, agents bool
		want            bool
	}{
		{DangerMutating, OriginExternalAgent, true, false, false},
		{DangerMutating, OriginExternalAgent, true, true, true},
		{DangerMutating, OriginBuiltInAgent, true, true, true},
		{DangerMutating, OriginExternalAgent, false, false, true},
		{DangerMutating, OriginGUI, true, true, false},
		{DangerMutating, OriginCLI, false, true, false},
		{DangerDestructive, OriginGUI, false, false, true},
		{DangerRead, OriginExternalAgent, false, true, false},
	} {
		if got := NeedsConfirmation(c.d, c.o, c.allowed, c.agents); got != c.want {
			t.Errorf("NeedsConfirmation(%s, %s, allowed %v, agentsAsk %v) = %v, want %v", c.d, c.o, c.allowed, c.agents, got, c.want)
		}
	}
}

// TestAgentsAskDispatch: a command's AgentsAsk, true for some parameters, makes Dispatch ask an
// agent in a context where agent writes are allowed, marks the request, and is never judged for
// the GUI; only mutating commands may declare it.
func TestAgentsAskDispatch(t *testing.T) {
	var asked []ConfirmRequest
	s := New(Options{Confirmer: ConfirmFunc(func(_ context.Context, req ConfirmRequest) (bool, error) {
		asked = append(asked, req)
		return true, nil
	})})
	s.mu.Lock()
	s.ui.Context = "c"
	s.mu.Unlock()
	s.SetAgentWrites("c", true)
	judged := 0
	err := Register(s.Registry(), Spec[agentsAskProbeParams]{
		ID: "probe.grant", Label: "Grant", Danger: DangerMutating,
		Targets: func(*Session, agentsAskProbeParams) ([]model.ObjectRef, error) {
			return []model.ObjectRef{{Kind: "rolebindings", Namespace: "ns", Name: "b"}}, nil
		},
		AgentsAsk: func(_ *Session, p agentsAskProbeParams) bool { judged++; return p.Grant },
		Run:       func(context.Context, *Session, agentsAskProbeParams) (any, error) { return nil, nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	if c, _ := s.Registry().Get("probe.grant"); !c.AgentsAsk {
		t.Error("the command does not say that agents always ask for some calls")
	}
	dispatch := func(o Origin, params string) {
		t.Helper()
		if _, err := s.Dispatch(context.Background(), Invocation{ID: "probe.grant", Params: json.RawMessage(params), Origin: o}); err != nil {
			t.Fatal(err)
		}
	}
	dispatch(OriginExternalAgent, `{}`)
	if len(asked) != 0 {
		t.Fatalf("an allowed agent write without a grant asked: %+v", asked)
	}
	dispatch(OriginExternalAgent, `{"grant":true}`)
	if len(asked) != 1 || !asked[0].AgentsAsk || asked[0].Danger != DangerMutating || asked[0].Context != "c" {
		t.Fatalf("an agent's grant with agent writes allowed: asked %+v", asked)
	}
	before := judged
	dispatch(OriginGUI, `{"grant":true}`)
	if len(asked) != 1 || judged != before {
		t.Errorf("the GUI's grant: asked %d, judged %d times", len(asked)-1, judged-before)
	}

	for _, d := range []Danger{DangerDestructive, DangerRead, DangerUI} {
		err := Register(s.Registry(), Spec[agentsAskProbeParams]{
			ID: "probe.bad" + string(d), Label: "Bad", Danger: d,
			Targets:   func(*Session, agentsAskProbeParams) ([]model.ObjectRef, error) { return nil, nil },
			AgentsAsk: func(*Session, agentsAskProbeParams) bool { return true },
			Run:       func(context.Context, *Session, agentsAskProbeParams) (any, error) { return nil, nil },
		})
		if err == nil {
			t.Errorf("a %s command declared AgentsAsk: %v", d, err)
		}
	}
}
