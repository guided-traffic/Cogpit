package engine

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/terminal"
)

// openAttach starts the session of an attach dock tab (pod.attach): the stream to the tab's
// container's main process, with stdin and a TTY as the container has them in the cached pod.
func (s *Session) openAttach(ctx context.Context, tab DockTab, b *Backend) (terminal.Session, error) {
	pod, err := s.Object(ctx, *tab.Ref)
	if err != nil {
		return nil, err
	}
	return terminal.StartPodAttach(ctx, b.Conn.RESTConfig(), b.Conn.Kube(), AttachOptions(pod, tab.Container))
}

// AttachOptions are the attach options of a pod's container: its stdin and TTY flags from the
// pod's spec (containers, then init containers). An unknown container gets neither: its output is
// shown and nothing typed is sent.
func AttachOptions(pod *unstructured.Unstructured, container string) terminal.AttachOptions {
	opts := terminal.AttachOptions{Namespace: pod.GetNamespace(), Pod: pod.GetName(), Container: container}
	for _, field := range []string{"containers", "initContainers"} {
		for _, c := range model.NestedSlice(pod, "spec", field) {
			if model.NestedString(c, "name") != container {
				continue
			}
			opts.Stdin, _ = model.NestedBool(c, "stdin")
			opts.TTY, _ = model.NestedBool(c, "tty")
			return opts
		}
	}
	return opts
}
