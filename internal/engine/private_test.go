package engine

import (
	"context"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// A private drawer item (Helm values) reaches the user's window and never an agent: the GUI's
// Detail and the CLI show it, object_get's DetailWait for an agent withholds it.
func TestDetailWithholdsPrivateItemsFromAgents(t *testing.T) {
	cm := &model.KindProfile{
		Key: "configmaps", Version: "v1", Resource: "configmaps", Kind: "ConfigMap", Title: "Config Maps", Namespaced: true,
		Columns: []model.Column{{ID: "name", Title: "Name"}},
		Detail: func(*unstructured.Unstructured) []model.DetailSection {
			return []model.DetailSection{{Title: "Values", Items: []model.DetailItem{{Value: "token: s3cr3t", Private: true}}}}
		},
	}
	src := testkit.NewMemorySource()
	src.Set(cm.GVR(), testkit.Object("v1", "ConfigMap", "apps", "web", nil))
	s, fc := newTestSession(t, src, 5*time.Millisecond)
	fc.profiles = append(testProfiles(), cm)
	connectTest(t, s)
	ref := model.ObjectRef{Kind: "configmaps", Namespace: "apps", Name: "web"}

	for _, tt := range []struct {
		origin Origin
		want   string
	}{{OriginGUI, "token: s3cr3t"}, {OriginCLI, "token: s3cr3t"}, {OriginExternalAgent, model.Withheld}, {OriginBuiltInAgent, model.Withheld}} {
		ctx := context.WithValue(context.Background(), originKey{}, tt.origin)
		m, err := s.DetailWait(ctx, ref)
		if err != nil {
			t.Fatal(err)
		}
		if got := sectionValue(m, "Values"); got != tt.want {
			t.Errorf("%s: %q, want %q", tt.origin, got, tt.want)
		}
	}
	m, err := s.Detail(ref)
	if err != nil {
		t.Fatal(err)
	}
	if got := sectionValue(m, "Values"); got != "token: s3cr3t" {
		t.Errorf("the GUI's drawer: %q", got)
	}
}
