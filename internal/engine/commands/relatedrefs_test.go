package commands_test

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// relatedConnector serves the built-in profiles of the kinds a Deployment's drawer relates to.
type relatedConnector struct{ src *testkit.MemorySource }

func (c *relatedConnector) Contexts() ([]engine.ContextInfo, error) {
	return []engine.ContextInfo{{Name: "test", Current: true}}, nil
}

func (c *relatedConnector) Connect(_ context.Context, name string) (*engine.Backend, error) {
	var profiles []*model.KindProfile
	for _, k := range []string{"pods", "nodes", "deployments", "replicasets", "namespaces"} {
		p, ok := kinds.ByKey(k)
		if !ok {
			return nil, fmt.Errorf("no profile %s", k)
		}
		profiles = append(profiles, p)
	}
	ch := cache.New(c.src, cache.Options{WarmPeriod: time.Minute})
	return &engine.Backend{Context: name, Namespace: "default", Cache: ch, Profiles: profiles, Close: ch.Close}, nil
}

// ownedBy makes obj controlled by the object of kind and name in its namespace (testkit's UIDs are
// "uid-<namespace>-<name>").
func ownedBy(obj *unstructured.Unstructured, apiVersion, kind, name string) *unstructured.Unstructured {
	return controlledBy(obj, apiVersion, kind, name, "uid-"+obj.GetNamespace()+"-"+name)
}

func onNode(pod *unstructured.Unstructured, node string) *unstructured.Unstructured {
	_ = unstructured.SetNestedField(pod.Object, node, "spec", "nodeName")
	return pod
}

func relatedHarness(t *testing.T, src *testkit.MemorySource) *harness {
	t.Helper()
	s := engine.New(engine.Options{Connector: &relatedConnector{src: src}, SnapshotInterval: 10 * time.Millisecond})
	t.Cleanup(s.Close)
	if err := commands.RegisterAll(s.Registry()); err != nil {
		t.Fatal(err)
	}
	if err := s.Connect(context.Background(), "test"); err != nil {
		t.Fatal(err)
	}
	return &harness{s: s, src: src}
}

type objectResult struct {
	Ref              model.ObjectRef     `json:"ref"`
	Title            string              `json:"title"`
	YAML             string              `json:"yaml"`
	Related          []engine.RelatedRef `json:"related"`
	RelatedTotal     int                 `json:"relatedTotal"`
	RelatedTruncated bool                `json:"relatedTruncated"`
}

func relatedNames(rs []engine.RelatedRef) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.Relation + ": " + r.Ref.Kind + " " + r.Ref.Name
	}
	return out
}

// object.get with related lists what the drawer links to and what a workload controls, as refs from
// the cache: a Deployment its pods (its drawer's Pods table, D-40: not another Deployment's, not a
// pod without an owner), their node and its replica set, a pod its controller and its node, a node
// its pods; each object once, never the object itself; without related the result is the drawer
// model alone.
func TestObjectGetRelated(t *testing.T) {
	src := testkit.NewMemorySource()
	src.Set(testkit.NodesGVR, testkit.Object("v1", "Node", "", "node-1", nil))
	src.Set(testkit.DeploymentsGVR, testkit.Object("apps/v1", "Deployment", "shop", "web", nil))
	src.Set(testkit.DeploymentsGVR, testkit.Object("apps/v1", "Deployment", "shop", "api", nil))
	src.Set(replicaSetsGVR, ownedBy(testkit.Object("apps/v1", "ReplicaSet", "shop", "web-abc", nil), "apps/v1", "Deployment", "web"))
	src.Set(replicaSetsGVR, ownedBy(testkit.Object("apps/v1", "ReplicaSet", "shop", "api-xyz", nil), "apps/v1", "Deployment", "api"))
	src.Set(testkit.PodsGVR, ownedBy(testkit.Pod("shop", "web-abc-2"), "apps/v1", "ReplicaSet", "web-abc"))
	src.Set(testkit.PodsGVR, ownedBy(testkit.Pod("shop", "web-abc-1"), "apps/v1", "ReplicaSet", "web-abc"))
	src.Set(testkit.PodsGVR, ownedBy(testkit.Pod("shop", "api-xyz-1"), "apps/v1", "ReplicaSet", "api-xyz"))
	src.Set(testkit.PodsGVR, testkit.Pod("shop", "lonely"))
	h := relatedHarness(t, src)
	agent := engine.OriginExternalAgent

	deploy := model.ObjectRef{Kind: "deployments", Namespace: "shop", Name: "web"}
	var got objectResult
	h.must(t, agent, "object.get", map[string]any{"ref": deploy, "related": true, "noYaml": true}, &got)
	// The drawer's links first: its Namespace, then its Pods table (D-40: each pod and its node, in
	// the table's order), then the replica sets it controls; the pods again are not repeated.
	want := []string{"Namespace: namespaces shop", "Pods: pods web-abc-1", "Pods: nodes node-1", "Pods: pods web-abc-2", "Replica Sets: replicasets web-abc"}
	if names := relatedNames(got.Related); !slices.Equal(names, want) || got.RelatedTotal != 5 || got.RelatedTruncated {
		t.Errorf("the Deployment's related objects: %v (total %d, truncated %v), want %v", names, got.RelatedTotal, got.RelatedTruncated, want)
	}
	for _, r := range got.Related {
		if r.Ref.Context != "test" || (r.Ref.Kind != "namespaces" && r.Ref.Kind != "nodes" && r.Ref.Namespace != "shop") {
			t.Errorf("related ref %+v: want context test and namespace shop", r.Ref)
		}
	}
	if got.Title == "" || got.Ref.Name != "web" {
		t.Errorf("the drawer model is missing: %+v", got)
	}

	pod := model.ObjectRef{Kind: "pods", Namespace: "shop", Name: "web-abc-1"}
	h.must(t, agent, "object.get", map[string]any{"ref": pod, "related": true}, &got)
	names := relatedNames(got.Related)
	for _, w := range []string{"Controlled By: replicasets web-abc", "Node: nodes node-1"} {
		if !slices.Contains(names, w) {
			t.Errorf("the pod's related objects %v lack %q", names, w)
		}
	}
	if slices.ContainsFunc(got.Related, func(r engine.RelatedRef) bool { return r.Ref.Kind == "pods" && r.Ref.Name == "web-abc-1" }) {
		t.Errorf("the pod relates to itself: %v", names)
	}

	node := model.ObjectRef{Kind: "nodes", Name: "node-1"}
	h.must(t, agent, "object.get", map[string]any{"ref": node, "related": true, "noYaml": true}, &got)
	var pods []string
	for _, r := range got.Related {
		if r.Ref.Kind == "pods" {
			pods = append(pods, r.Ref.Name)
		}
	}
	slices.Sort(pods)
	if !slices.Equal(pods, []string{"api-xyz-1", "lonely", "web-abc-1", "web-abc-2"}) {
		t.Errorf("the node's pods: %v (related %v)", pods, relatedNames(got.Related))
	}

	// Without related: the drawer model only.
	res, err := h.s.Dispatch(context.Background(), engine.Invocation{ID: "object.get", Params: json.RawMessage(`{"ref":{"kind":"deployments","namespace":"shop","name":"web"}}`), Origin: agent})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(res)
	if strings.Contains(string(b), `"related`) {
		t.Errorf("object.get without related lists them: %s", b)
	}
}

// The related objects are bounded: a node with 250 pods gives 200 of them, relatedTotal 250.
func TestObjectGetRelatedIsBounded(t *testing.T) {
	src := testkit.NewMemorySource()
	src.Set(testkit.NodesGVR, testkit.Object("v1", "Node", "", "big", nil))
	for i := range 250 {
		src.Set(testkit.PodsGVR, onNode(testkit.Pod("load", fmt.Sprintf("p-%03d", i)), "big"))
	}
	h := relatedHarness(t, src)
	var got objectResult
	h.must(t, engine.OriginExternalAgent, "object.get", map[string]any{"ref": model.ObjectRef{Kind: "nodes", Name: "big"}, "related": true, "noYaml": true}, &got)
	pods := 0
	for _, r := range got.Related {
		if r.Ref.Kind == "pods" {
			pods++
		}
	}
	if len(got.Related) != engine.MaxRelated || !got.RelatedTruncated || got.RelatedTotal < 250 || pods < engine.MaxRelated-5 {
		t.Errorf("%d related (%d pods), total %d, truncated %v; want %d of at least 250", len(got.Related), pods, got.RelatedTotal, got.RelatedTruncated, engine.MaxRelated)
	}
}
