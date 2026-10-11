package engine

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestEventInvolvesNodeByName(t *testing.T) {
	// The kubelet reports node events with the node's name as the involved object's UID.
	ev := &unstructured.Unstructured{Object: map[string]any{
		"involvedObject": map[string]any{"kind": "Node", "name": "worker-1", "uid": "worker-1"},
	}}
	if !eventInvolves(ev, "Node", "", "worker-1", "6c0f3c1e-real-node-uid") {
		t.Error("a kubelet node event does not match its Node")
	}
	if eventInvolves(ev, "Node", "", "worker-2", "other-uid") {
		t.Error("a node event matches another Node")
	}
	pod := &unstructured.Unstructured{Object: map[string]any{
		"involvedObject": map[string]any{"kind": "Pod", "namespace": "a", "name": "web-0", "uid": "old-uid"},
	}}
	if eventInvolves(pod, "Pod", "a", "web-0", "new-uid") {
		t.Error("an event of an earlier pod with the same name matches by name although the UIDs differ")
	}
}
