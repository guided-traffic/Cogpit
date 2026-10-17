package commands_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// rejectProber is a connection whose login the API server rejects once the test says so.
type rejectProber struct{ reject atomic.Bool }

func (p *rejectProber) Heartbeat(context.Context) error {
	if p.reject.Load() {
		return cluster.ErrLoginRejected
	}
	return nil
}
func (p *rejectProber) Reachable(context.Context) error { return nil }
func (p *rejectProber) SetHealth(cluster.Health)        {}
func (p *rejectProber) ResetConnections()               {}
func (p *rejectProber) ArmLogin()                       {}

// pluginConnector is the fake connector whose contexts log in through a credential plugin.
type pluginConnector struct{ *fakeConnector }

func (c pluginConnector) Connect(ctx context.Context, name string) (*engine.Backend, error) {
	b, err := c.fakeConnector.Connect(ctx, name)
	if b != nil {
		b.CredentialPlugin = true
	}
	return b, err
}

func reconnectHarness(t *testing.T, plugin bool) (*harness, *rejectProber) {
	t.Helper()
	src := seed()
	p := &rejectProber{}
	h := &harness{src: src, writer: &fakeWriter{}}
	var conn engine.Connector = &fakeConnector{src: src, writer: h.writer, health: p}
	if plugin {
		conn = pluginConnector{&fakeConnector{src: src, writer: h.writer, health: p}}
	}
	s := engine.New(engine.Options{
		Connector: conn, SnapshotInterval: 10 * time.Millisecond,
		ConnWatch: engine.ConnWatchOptions{Heartbeat: 20 * time.Millisecond, ProbeTimeout: 50 * time.Millisecond, Confirm: 10 * time.Millisecond,
			LoginWait: 300 * time.Millisecond, BackoffFirst: 20 * time.Millisecond, BackoffMax: 100 * time.Millisecond},
	})
	if err := commands.RegisterAll(s.Registry()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	if err := s.Connect(context.Background(), "test"); err != nil {
		t.Fatal(err)
	}
	h.s = s
	return h, p
}

// TestReconnectCommand is F29's command: the window and agents reconnect a connected cluster, its
// tabs and view kept, and get the new connection; the demo cluster and a cluster not connected are
// refused with the reason; an agent may not reconnect a cluster whose plugin login expired.
func TestReconnectCommand(t *testing.T) {
	h, _ := reconnectHarness(t, false)
	gui, agent := engine.OriginGUI, engine.OriginExternalAgent
	h.s.OpenTab("pods", true)
	before := h.s.UIState()
	old := h.s.Backend()
	var res commands.ConnectionResult
	h.must(t, gui, "cluster.reconnect", nil, &res)
	if res.Context != "test" || res.Connection.State != engine.Connected || res.Connection.Reconnecting {
		t.Errorf("the window's reconnect: %+v", res)
	}
	if h.s.Backend() == old {
		t.Error("the backend stayed")
	}
	if u := h.s.UIState(); u.View != before.View || len(u.Tabs) != len(before.Tabs) || u.ActiveTab != before.ActiveTab {
		t.Errorf("the tabs after the reconnect: %d/%d, view %q", len(u.Tabs), len(before.Tabs), u.View)
	}
	h.must(t, agent, "cluster.reconnect", map[string]any{"context": "test"}, &res) // a UI command: no confirmation
	var de *engine.DisabledError
	if err := h.run(t, gui, "cluster.reconnect", map[string]any{"context": "other"}, nil); !errors.As(err, &de) || !errors.Is(err, engine.ErrNotConnected) {
		t.Errorf("a cluster not connected: %v", err)
	}

	hp, p := reconnectHarness(t, true)
	p.reject.Store(true)
	waitConn(t, hp.s, engine.LoginExpired)
	if c := hp.s.UIState().Connection; !c.CredentialPlugin {
		t.Errorf("the plugin's connection: %+v", c)
	}
	for _, o := range []engine.Origin{agent, engine.OriginBuiltInAgent, engine.OriginCLI} {
		if err := hp.run(t, o, "cluster.reconnect", nil, nil); !errors.Is(err, engine.ErrReconnectLoginGUIOnly) {
			t.Errorf("%s's reconnect of an expired plugin login: %v", o, err)
		}
	}
	p.reject.Store(false)
	hp.must(t, gui, "cluster.reconnect", nil, &res)
	if res.Connection.State != engine.Connected {
		t.Errorf("the window's reconnect of an expired plugin login: %+v", res)
	}

	demo := newDemoHarness(t, engine.Options{})
	if err := demo.run(t, gui, "cluster.reconnect", nil, nil); !errors.As(err, &de) {
		t.Errorf("the demo cluster's reconnect: %v", err)
	}
}

// TestPortForwardsCarriedAcrossAReconnect: the forwards that listened before a reconnect listen on
// the same local ports after it, now through the new connection.
func TestPortForwardsCarriedAcrossAReconnect(t *testing.T) {
	h, fw := newForwardHarness(t, engine.Options{})
	var a, b engine.Forward
	h.must(t, engine.OriginGUI, "pod.portForward", map[string]any{"ref": web0, "port": 8080}, &a)
	h.must(t, engine.OriginGUI, "service.portForward", map[string]any{"ref": webService, "port": 80}, &b)
	dials := len(fw.dialed())
	h.must(t, engine.OriginGUI, "cluster.reconnect", nil, nil)
	for _, f := range []engine.Forward{a, b} {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		ok := testkit.WaitFor(ctx, func() bool {
			got, err := h.s.ForwardOf(f.Ref)
			return err == nil && got.Listening() && got.LocalPort == f.LocalPort && got.Phase == engine.ForwardActive
		})
		cancel()
		if !ok {
			got, err := h.s.ForwardOf(f.Ref)
			t.Fatalf("%s after the reconnect: %+v %v", f.Ref.Name, got, err)
		}
		if line, err := ask(t, f.Address); err != nil || line == "" {
			t.Errorf("%s does not answer on %s after the reconnect: %q %v", f.Ref.Name, f.Address, line, err)
		}
	}
	if len(fw.dialed()) <= dials {
		t.Error("the forwards did not dial through the new connection")
	}
}
