//go:build darwin || linux

package cluster

import (
	"context"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/testkit"
)

// TestCredentialPluginIsNoted: a context whose user runs a credential plugin says so; its login in
// the file is the plugin's configuration, unchanged while the plugin hands out new tokens.
func TestCredentialPluginIsNoted(t *testing.T) {
	srv := testkit.NewFakeAPIServer(testkit.FakeAPIOptions{Token: "tok"})
	defer srv.Close()
	p := testkit.NewLoginPlugin(t, srv, "oidc")
	p.Give("tok", time.Time{})
	c, err := Connect(context.Background(), Options{KubeconfigPaths: []string{p.Kubeconfig}, Context: "oidc"})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if !c.CredentialPlugin() {
		t.Error("the plugin's context does not say it runs one")
	}
	p.Give("another", time.Time{})
	if _, changed, err := c.LoginInFile(); err != nil || changed {
		t.Errorf("the plugin's file after a new token: %v %v", changed, err)
	}
}
