package engine

import (
	"strings"
	"testing"
)

// TestNodeShellImageRules: what counts as pinned, what cannot be an image or a pull secret, and the
// default being pinned (privilege-footprint.md › Node shell: "a pinned default, by digest").
func TestNodeShellImageRules(t *testing.T) {
	if !ImagePinned(DefaultNodeShellImage) || !strings.HasPrefix(DefaultNodeShellImage, "docker.io/library/alpine:3.19@sha256:") {
		t.Errorf("the default %s is not alpine 3.19 pinned by digest", DefaultNodeShellImage)
	}
	for image, pinned := range map[string]bool{
		"alpine:3.19": false, "docker.io/alpine": false, "alpine@sha256:abc": false,
		"registry.example.com:5000/a/b@sha256:" + strings.Repeat("0", 64): true,
	} {
		if ImagePinned(image) != pinned {
			t.Errorf("ImagePinned(%q) = %v", image, !pinned)
		}
	}
	for _, bad := range []string{"alpine 3.19", "https://example.com/alpine", "a\nb", strings.Repeat("a", 513)} {
		if CheckNodeShellImage(bad) == nil {
			t.Errorf("image %q accepted", bad)
		}
	}
	for _, good := range []string{"", "alpine:3.19", "registry.example.com:5000/tools/shell:1.2"} {
		if err := CheckNodeShellImage(good); err != nil {
			t.Errorf("image %q: %v", good, err)
		}
	}
	if CheckNodeShellPullSecret("Not_A_Name") == nil || CheckNodeShellPullSecret("") != nil || CheckNodeShellPullSecret("registry-pull") != nil {
		t.Error("pull secret names")
	}
	if !strings.Contains(NodeShellDetail("n1", "alpine:3.19", ""), "a floating tag, not pinned by digest") {
		t.Error("a tag's detail does not say it floats")
	}
	pod := NodeShellPod("cogpit-node-shell-x", "n1", "alpine:3.19", "pull")
	if len(pod.Spec.ImagePullSecrets) != 1 || pod.Spec.ImagePullSecrets[0].Name != "pull" || pod.Labels["app.kubernetes.io/name"] != "cogpit-node-shell" {
		t.Errorf("pod %+v", pod)
	}
	pod.Labels["changed"] = "x"
	if _, shared := NodeShellLabels["changed"]; shared {
		t.Error("a pod's labels are NodeShellLabels themselves")
	}
	if name := nodeShellName(); len(name) != len(nodeShellNamePrefix)+8 || strings.ToLower(name) != name {
		t.Errorf("pod name %q", name)
	}
}
