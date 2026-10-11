package cluster

import (
	"context"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/testkit"
)

// GetSecret reads one Secret with its data through the connection's counted client: one "get
// secrets" request, and nothing else; a missing Secret is an error that names it.
func TestGetSecretIsOneCountedGet(t *testing.T) {
	srv := testkit.NewFakeAPIServer(testkit.FakeAPIOptions{
		Resources: append(append([]testkit.FakeResource(nil), testkit.DefaultFakeResources...),
			testkit.FakeResource{GVR: testkit.SecretsGVR, Kind: "Secret", Namespaced: true}),
	})
	defer srv.Close()
	srv.Set(testkit.SecretsGVR, testkit.Secret("apps", "db", nil, map[string]string{"password": "hunter2"}))
	c, err := Connect(context.Background(), Options{KubeconfigPaths: []string{srv.WriteKubeconfig(t, "fake", "")}})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	before := c.Counter().Snapshot()
	obj, err := c.GetSecret(context.Background(), "apps", "db")
	if err != nil {
		t.Fatal(err)
	}
	if v, _, _ := unstructured.NestedString(obj.Object, "data", "password"); v != "aHVudGVyMg==" {
		t.Errorf("data.password = %q", v)
	}
	after := c.Counter().Snapshot()
	if got := after.ByVerbResource["get secrets"] - before.ByVerbResource["get secrets"]; got != 1 || after.Total-before.Total != 1 {
		t.Errorf("get secrets counted %d, %d requests in all; want one GET", got, after.Total-before.Total)
	}
	if srv.Requests()["get secrets"] != 1 {
		t.Errorf("server saw %v", srv.Requests())
	}

	_, err = c.GetSecret(context.Background(), "apps", "gone")
	if err == nil || !strings.Contains(err.Error(), "secrets apps/gone") {
		t.Errorf("missing Secret: %v", err)
	}
	if _, err := c.GetSecret(context.Background(), "", "db"); err == nil {
		t.Error("a Secret without a namespace was asked for")
	}
}
