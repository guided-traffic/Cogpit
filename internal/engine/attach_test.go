package engine

import (
	"testing"

	"github.com/guided-traffic/cogpit/internal/terminal"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// TestAttachOptionsFromTheContainer: attach uses stdin and a TTY only when the cached pod's
// container has them, for containers and init containers; an unknown container gets neither.
func TestAttachOptionsFromTheContainer(t *testing.T) {
	pod := testkit.Object("v1", "Pod", "shop", "repl-0", map[string]any{"spec": map[string]any{
		"initContainers": []any{map[string]any{"name": "setup", "stdin": true}},
		"containers": []any{
			map[string]any{"name": "repl", "stdin": true, "tty": true},
			map[string]any{"name": "app"},
			map[string]any{"name": "odd", "stdin": "yes", "tty": 1}, // hostile shapes: neither
		},
	}})
	for container, want := range map[string]terminal.AttachOptions{
		"repl":  {Stdin: true, TTY: true},
		"app":   {},
		"setup": {Stdin: true},
		"odd":   {},
		"gone":  {},
	} {
		want.Namespace, want.Pod, want.Container = "shop", "repl-0", container
		if got := AttachOptions(pod, container); got != want {
			t.Errorf("%s: %+v, want %+v", container, got, want)
		}
	}
}
