package commands_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
)

// proxyPassword is a proxy credential planted in the tests: only the settings file may hold it.
const proxyPassword = "pw-PROXY-secret-31"

// proxySession is settingsSession with an audit log.
func proxySession(t *testing.T) (*engine.Session, *syncBuffer, string) {
	t.Helper()
	cfg := filepath.Join(t.TempDir(), "Cogpit")
	kubeconfig := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(kubeconfig, []byte(settingsKubeconfig), 0o600); err != nil {
		t.Fatal(err)
	}
	audit := &syncBuffer{}
	s := engine.New(engine.Options{
		Connector:       &engine.ClusterConnector{KubeconfigPaths: []string{kubeconfig}, IncludeDemo: true},
		Preferences:     engine.NewPreferencesStore(cfg),
		ClusterSettings: engine.NewClusterSettingsStore(cfg),
		Audit:           audit,
	})
	if err := commands.RegisterAll(s.Registry()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	return s, audit, cfg
}

// Only the user sets a cluster's proxy, in the window; agents and the CLI are refused before
// anything is kept, may clear it, and read it back with the password as •••. The audit log
// carries the refused attempt with the password redacted.
func TestClusterProxyCommands(t *testing.T) {
	s, audit, cfg := proxySession(t)
	withPassword := "http://alice:" + proxyPassword + "@proxy.example.com:3128"
	for _, origin := range []engine.Origin{engine.OriginExternalAgent, engine.OriginBuiltInAgent, engine.OriginCLI} {
		err := dispatch(t, s, origin, "cluster.settings.set", map[string]any{"context": "kind-dev", "proxy": withPassword}, nil)
		if !errors.Is(err, engine.ErrProxyGUIOnly) {
			t.Errorf("%s set a proxy: %v", origin, err)
		}
		if err != nil && strings.Contains(err.Error(), proxyPassword) {
			t.Errorf("the refusal quotes the password: %v", err)
		}
	}
	if cs, _ := s.ClusterSettingsOf("kind-dev"); cs.Proxy != "" {
		t.Fatalf("a refused proxy was kept: %q", cs.Proxy)
	}

	var v engine.ClusterSettingsView
	if err := dispatch(t, s, engine.OriginGUI, "cluster.settings.set", map[string]any{"context": "kind-dev", "proxy": withPassword}, &v); err != nil {
		t.Fatal(err)
	}
	if v.Proxy != "http://alice:•••@proxy.example.com:3128" {
		t.Errorf("the user's set returns %q", v.Proxy)
	}
	var got engine.ClusterSettingsView
	if err := dispatch(t, s, engine.OriginExternalAgent, "cluster.settings.get", map[string]any{"context": "kind-dev"}, &got); err != nil {
		t.Fatal(err)
	}
	if got.Proxy != v.Proxy {
		t.Errorf("an agent reads %q", got.Proxy)
	}
	// The field's redacted form with another host keeps the password (the user edits the host).
	if err := dispatch(t, s, engine.OriginGUI, "cluster.settings.set", map[string]any{"context": "kind-dev", "proxy": "http://alice:•••@other:8080"}, nil); err != nil {
		t.Fatal(err)
	}
	if cs, _ := s.ClusterSettingsOf("kind-dev"); cs.Proxy != "http://alice:"+proxyPassword+"@other:8080" {
		t.Errorf("after editing the host: %q", cs.Proxy)
	}
	if err := dispatch(t, s, engine.OriginGUI, "cluster.settings.set", map[string]any{"context": "kind-dev", "proxy": "ftp://alice:" + proxyPassword + "@x"}, nil); err == nil ||
		strings.Contains(err.Error(), proxyPassword) {
		t.Errorf("an ftp proxy: %v", err)
	}

	// An agent may clear it.
	if err := dispatch(t, s, engine.OriginExternalAgent, "cluster.settings.set", map[string]any{"context": "kind-dev", "proxy": ""}, nil); err != nil {
		t.Fatalf("an agent's clear: %v", err)
	}
	if cs, _ := s.ClusterSettingsOf("kind-dev"); cs.Proxy != "" {
		t.Errorf("not cleared: %q", cs.Proxy)
	}

	for _, line := range audit.Lines() {
		raw, _ := json.Marshal(line)
		if strings.Contains(string(raw), proxyPassword) {
			t.Errorf("the audit log carries the password: %s", raw)
		}
	}
	if lines := audit.Lines(); len(lines) == 0 || !strings.Contains(string(lines[0].Params), `alice:•••@proxy.example.com`) {
		t.Errorf("the refused attempt's audit line: %+v", lines)
	}
	data, _ := os.ReadFile(filepath.Join(cfg, engine.ClusterSettingsFile))
	if strings.Contains(string(data), "•••") {
		t.Errorf("the file keeps the redaction mark: %s", data)
	}
}
