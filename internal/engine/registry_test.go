package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/guided-traffic/cogpit/internal/model"
)

type scaleProbeParams struct {
	Ref      model.ObjectRef `json:"ref"`
	Replicas *int            `json:"replicas,omitempty"`
}

// TestEscalateConfirmsAsDestructive: a mutating command whose Escalate says yes (Scale to zero)
// needs a confirmation from every origin, with the request naming it destructive; without
// escalation it follows the mutating rules (the GUI is not asked).
func TestEscalateConfirmsAsDestructive(t *testing.T) {
	var asked []ConfirmRequest
	s := New(Options{Confirmer: ConfirmFunc(func(_ context.Context, req ConfirmRequest) (bool, error) {
		asked = append(asked, req)
		return true, nil
	})})
	ran := 0
	err := Register(s.Registry(), Spec[scaleProbeParams]{
		ID: "probe.scale", Label: "Scale", Danger: DangerMutating, Order: 200,
		Targets:  func(_ *Session, p scaleProbeParams) ([]model.ObjectRef, error) { return []model.ObjectRef{p.Ref}, nil },
		Escalate: func(_ *Session, p scaleProbeParams) bool { return p.Replicas != nil && *p.Replicas == 0 },
		Prefill: func(_ *Session, p scaleProbeParams) (scaleProbeParams, error) {
			if p.Replicas == nil {
				n := 3
				p.Replicas = &n
			}
			return p, nil
		},
		Run: func(context.Context, *Session, scaleProbeParams) (any, error) { ran++; return nil, nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	cmd, _ := s.Registry().Get("probe.scale")
	if !cmd.Escalates || cmd.Order != 200 {
		t.Errorf("command: escalates %v, order %d", cmd.Escalates, cmd.Order)
	}

	dispatch := func(params string) error {
		_, err := s.Dispatch(context.Background(), Invocation{ID: "probe.scale", Params: json.RawMessage(params), Origin: OriginGUI})
		return err
	}
	if err := dispatch(`{"ref":{"kind":"deployments","namespace":"shop","name":"web"},"replicas":2}`); err != nil {
		t.Fatal(err)
	}
	if len(asked) != 0 || ran != 1 {
		t.Fatalf("scale to 2 from the GUI: asked %d times, ran %d; want no confirmation", len(asked), ran)
	}
	if err := dispatch(`{"ref":{"kind":"deployments","namespace":"shop","name":"web"},"replicas":0}`); err != nil {
		t.Fatal(err)
	}
	if len(asked) != 1 || asked[0].Danger != DangerDestructive || ran != 2 {
		t.Fatalf("scale to 0 from the GUI: asked %+v, ran %d; want one destructive confirmation", asked, ran)
	}

	got, err := cmd.Prefill(s, json.RawMessage(`{"ref":{"kind":"deployments","namespace":"shop","name":"web"}}`))
	if err != nil {
		t.Fatal(err)
	}
	var p scaleProbeParams
	if err := json.Unmarshal(got, &p); err != nil || p.Replicas == nil || *p.Replicas != 3 {
		t.Errorf("Prefill = %s, %v; want replicas 3", got, err)
	}
	if _, err := cmd.Prefill(s, json.RawMessage(`{"ref":1}`)); err == nil {
		t.Error("Prefill accepted bad params")
	}
}

// TestNotApplicableHidesTheCommand: Enabled wrapping ErrNotApplicable makes Applicable false and
// Dispatch refuse; any other reason leaves the command applicable but disabled.
func TestNotApplicableHidesTheCommand(t *testing.T) {
	s := New(Options{})
	err := Register(s.Registry(), Spec[refParams]{
		ID: "probe.cordon", Label: "Cordon", Danger: DangerRead, Order: 200,
		Enabled: func(_ *Session, p refParams) error {
			switch p.Ref.Name {
			case "cordoned":
				return fmt.Errorf("%w: the node is already cordoned", ErrNotApplicable)
			case "busy":
				return errors.New("busy")
			}
			return nil
		},
		Run: func(context.Context, *Session, refParams) (any, error) { return nil, nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	cmd, _ := s.Registry().Get("probe.cordon")
	params := func(name string) json.RawMessage {
		return json.RawMessage(`{"ref":{"kind":"nodes","name":"` + name + `"}}`)
	}
	if cmd.Applicable(s, params("cordoned")) {
		t.Error("a not-applicable command reported applicable")
	}
	if !cmd.Applicable(s, params("busy")) || !cmd.Applicable(s, params("ready")) {
		t.Error("a disabled or enabled command reported not applicable")
	}
	_, err = s.Dispatch(context.Background(), Invocation{ID: "probe.cordon", Params: params("cordoned"), Origin: OriginGUI})
	var dis *DisabledError
	if !errors.As(err, &dis) {
		t.Errorf("Dispatch of a not-applicable command: %v, want a DisabledError", err)
	}
}

func TestRegisterValidatesEscalateAndOrder(t *testing.T) {
	r := NewRegistry()
	run := func(context.Context, *Session, refParams) (any, error) { return nil, nil }
	targets := func(*Session, refParams) ([]model.ObjectRef, error) { return nil, nil }
	bad := []Spec[refParams]{
		{ID: "a.read", Danger: DangerRead, Run: run, Escalate: func(*Session, refParams) bool { return true }},
		{ID: "a.destroy", Danger: DangerDestructive, Run: run, Targets: targets, Escalate: func(*Session, refParams) bool { return true }},
		{ID: "a.order", Danger: DangerRead, Run: run, Order: -1},
	}
	for _, spec := range bad {
		if err := Register(r, spec); err == nil {
			t.Errorf("Register(%q) accepted a bad spec", spec.ID)
		}
	}
	if OrderEdit >= OrderDelete {
		t.Error("Edit must come before Delete")
	}
}
