//go:build nodemo

// The release build's connector (D-67), as connector_nodemo.go: go test -tags nodemo runs these
// (make check-nodemo, in CI's lint job).

package engine

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

// TestNoDemoInARelease: a release build offers no demo cluster whatever IncludeDemo says, and a
// kubeconfig context called "demo" is one like any other: listed with its server rather than
// shadowed, resolved and connected through the kubeconfig, and without the demo's link reason.
func TestNoDemoInARelease(t *testing.T) {
	if DemoBuilt || IsDemoContext(DemoContext) {
		t.Fatal("a nodemo build says it has the demo cluster")
	}

	// The merged kubeconfig (a run without a configuration directory): its context "demo" is listed.
	path := filepath.Join(t.TempDir(), "config")
	writeUserFile(t, path, sourceKubeconfig(DemoContext))
	merged := &ClusterConnector{KubeconfigPaths: []string{path}, IncludeDemo: true}
	list, err := merged.Contexts()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Name != DemoContext || list[0].Demo || list[0].Server != "https://demo.example:6443" {
		t.Errorf("contexts %+v, want the kubeconfig's context demo alone", list)
	}

	// No kubeconfig at all: nothing is offered, and connecting "demo" asks the kubeconfig.
	none := &ClusterConnector{KubeconfigPaths: []string{filepath.Join(t.TempDir(), "none")}, IncludeDemo: true}
	if list, _ := none.Contexts(); len(list) != 0 {
		t.Errorf("contexts without a kubeconfig: %+v", list)
	}
	if b, err := none.Connect(context.Background(), DemoContext); err == nil || errors.Is(err, errDemoUnavailable) {
		if b != nil {
			b.Close()
		}
		t.Errorf("connecting demo without a kubeconfig context of that name: %v", err)
	}

	// Kubeconfig sources: the bare name resolves to the source's context, as any other name does.
	kube := filepath.Join(t.TempDir(), ".kube")
	writeUserFile(t, filepath.Join(kube, "config"), sourceKubeconfig(DemoContext))
	sources := &ClusterConnector{Sources: NewKubeconfigSources(filepath.Join(t.TempDir(), "Cogpit"), WithDefaultFolder(kube)), IncludeDemo: true}
	list, err = sources.Contexts()
	if err != nil || len(list) != 1 || list[0].Demo {
		t.Fatalf("source contexts %+v (%v)", list, err)
	}
	if id, err := sources.ResolveContext(DemoContext); err != nil || id != list[0].Name {
		t.Errorf("demo resolves to %q (%v), want %q", id, err, list[0].Name)
	}
	s := New(Options{Connector: sources})
	t.Cleanup(s.Close)
	if link, err := s.ContextLink(DemoContext); err == nil {
		t.Errorf("the bare name demo has a link of its own: %+v", link)
	}
}
