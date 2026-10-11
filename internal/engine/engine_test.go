package engine

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/guided-traffic/cogpit/internal/model"
)

type refParams struct {
	Ref model.ObjectRef `json:"ref"`
}

func registerProbe(t *testing.T, s *Session, id string, d Danger, ran *int) {
	t.Helper()
	err := Register(s.Registry(), Spec[refParams]{
		ID: id, Label: "Probe", Danger: d,
		Targets: func(_ *Session, p refParams) ([]model.ObjectRef, error) { return []model.ObjectRef{p.Ref}, nil },
		Run: func(_ context.Context, _ *Session, p refParams) (any, error) {
			*ran++
			return p.Ref.Name, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestDispatchConfirmationPolicy(t *testing.T) {
	params := json.RawMessage(`{"ref":{"kind":"pods","namespace":"default","name":"web-0"}}`)
	cases := []struct {
		danger    Danger
		origin    Origin
		confirmer bool
		answer    bool
		wantRun   bool
		wantErr   error
	}{
		{DangerRead, OriginExternalAgent, false, false, true, nil},
		{DangerUI, OriginExternalAgent, false, false, true, nil},
		{DangerMutating, OriginGUI, false, false, true, nil},
		{DangerMutating, OriginExternalAgent, false, false, false, ErrConfirmationUnavailable},
		{DangerMutating, OriginExternalAgent, true, true, true, nil},
		{DangerDestructive, OriginGUI, true, false, false, ErrDeclined},
		{DangerDestructive, OriginGUI, true, true, true, nil},
		{DangerDestructive, OriginCLI, false, false, false, ErrConfirmationUnavailable},
	}
	for _, c := range cases {
		s := New(Options{})
		var asked int
		if c.confirmer {
			s.SetConfirmer(ConfirmFunc(func(_ context.Context, req ConfirmRequest) (bool, error) {
				asked++
				if len(req.Objects) != 1 || req.Objects[0].Name != "web-0" || req.Namespaces[0] != "default" {
					t.Errorf("confirm request does not name the object: %+v", req)
				}
				return c.answer, nil
			}))
		}
		ran := 0
		registerProbe(t, s, "probe.run", c.danger, &ran)
		_, err := s.Dispatch(context.Background(), Invocation{ID: "probe.run", Params: params, Origin: c.origin})
		if (ran == 1) != c.wantRun {
			t.Errorf("%s from %s: ran=%d, want run=%v (err %v)", c.danger, c.origin, ran, c.wantRun, err)
		}
		if c.wantErr != nil && !errors.Is(err, c.wantErr) {
			t.Errorf("%s from %s: err=%v, want %v", c.danger, c.origin, err, c.wantErr)
		}
		if c.wantErr == nil && err != nil {
			t.Errorf("%s from %s: unexpected err %v", c.danger, c.origin, err)
		}
	}
}

func TestDispatchRejectsBadParamsAndUnknownCommands(t *testing.T) {
	s := New(Options{})
	ran := 0
	registerProbe(t, s, "probe.run", DangerRead, &ran)
	for _, raw := range []string{`{"ref":{"kind":"pods"}}`, `{"ref":{"kind":"pods","name":"a"},"extra":1}`, `[1]`, `{`} {
		_, err := s.Dispatch(context.Background(), Invocation{ID: "probe.run", Params: json.RawMessage(raw), Origin: OriginGUI})
		var pe *ParamsError
		if !errors.As(err, &pe) {
			t.Errorf("params %s: err=%v, want ParamsError", raw, err)
		}
	}
	if ran != 0 {
		t.Errorf("command ran with bad params")
	}
	if _, err := s.Dispatch(context.Background(), Invocation{ID: "no.such", Origin: OriginGUI}); !errors.Is(err, ErrUnknownCommand) {
		t.Errorf("unknown command: err=%v", err)
	}
}

func TestDispatchRecoversPanics(t *testing.T) {
	s := New(Options{})
	err := Register(s.Registry(), Spec[struct{}]{
		ID: "probe.panic", Label: "Panic", Danger: DangerRead,
		Run: func(context.Context, *Session, struct{}) (any, error) { panic("boom") },
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Dispatch(context.Background(), Invocation{ID: "probe.panic", Origin: OriginGUI}); err == nil {
		t.Fatal("panic not turned into an error")
	}
}

func TestRegisterValidates(t *testing.T) {
	r := NewRegistry()
	run := func(context.Context, *Session, struct{}) (any, error) { return nil, nil }
	bad := []Spec[struct{}]{
		{ID: "", Danger: DangerRead, Run: run},
		{ID: "nodot", Danger: DangerRead, Run: run},
		{ID: "Ui.state", Danger: DangerRead, Run: run},
		{ID: "ui.state", Danger: DangerRead},
		{ID: "ui.state", Danger: "weird", Run: run},
		{ID: "pod.delete", Danger: DangerDestructive, Run: run}, // no Targets
	}
	for _, spec := range bad {
		if err := Register(r, spec); err == nil {
			t.Errorf("Register(%q) accepted a bad spec", spec.ID)
		}
	}
	if err := Register(r, Spec[struct{}]{ID: "ui.state", Danger: DangerUI, Run: run}); err != nil {
		t.Fatal(err)
	}
	if err := Register(r, Spec[struct{}]{ID: "ui.state", Danger: DangerUI, Run: run}); err == nil {
		t.Error("duplicate id accepted")
	}
	if c, _ := r.Get("ui.state"); c.ToolName() != "ui_state" {
		t.Errorf("tool name %q", c.ToolName())
	}
}

func TestUpdateUIEmitsEvents(t *testing.T) {
	s := New(Options{})
	var got []Event
	cancel := s.Subscribe(func(ev Event) { got = append(got, ev) })
	st := s.UpdateUI(func(u *UIState) { u.View = "services" })
	cancel()
	s.UpdateUI(func(u *UIState) { u.View = "pods" })
	if st.Revision != 1 || len(got) != 1 || got[0].UIState.View != "services" {
		t.Fatalf("revision %d, events %+v", st.Revision, got)
	}
}

// TestAgentWriteAllowanceIsBoundToItsContext: when the user allowed agent writes for one context,
// a mutating agent command runs without asking, but refuses to act on another context (the user
// switched clusters before it ran).
func TestAgentWriteAllowanceIsBoundToItsContext(t *testing.T) {
	s := New(Options{})
	s.UpdateUI(func(u *UIState) { u.Context = "kind-a" })
	s.SetAgentWrites("kind-a", true)
	var actOn string
	err := Register(s.Registry(), Spec[struct{}]{
		ID: "probe.write", Label: "Write", Danger: DangerMutating,
		Targets: func(*Session, struct{}) ([]model.ObjectRef, error) { return nil, nil },
		Run: func(ctx context.Context, _ *Session, _ struct{}) (any, error) {
			return nil, CheckConfirmedContext(ctx, actOn)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	inv := Invocation{ID: "probe.write", Origin: OriginExternalAgent}
	actOn = "kind-a"
	if _, err := s.Dispatch(context.Background(), inv); err != nil {
		t.Fatalf("allowed write on its own context: %v", err)
	}
	actOn = "kind-b"
	if _, err := s.Dispatch(context.Background(), inv); !errors.Is(err, ErrWrongCluster) {
		t.Fatalf("allowed write acting on another context: err=%v, want ErrWrongCluster", err)
	}
}
