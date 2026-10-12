package commands_test

import (
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

var podsResource = schema.GroupResource{Resource: "pods"}

// portFree reports whether nothing listens on the forward's local address any more.
func portFree(t *testing.T, addr string) bool {
	t.Helper()
	ln, err := net.Listen("tcp4", addr)
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}

// TestPortForwardFailsAtOncePermanentErrors (D-20): a connection that cannot be made, and that
// waiting does not cure, ends a forward that was active (or had waited for its pod) as Failed at
// once, names the cause and frees the local port; no dial follows.
func TestPortForwardFailsAtOncePermanentErrors(t *testing.T) {
	for name, cause := range map[string]error{
		"forbidden":    apierrors.NewForbidden(podsResource, "web-0", errors.New(`User "dev" cannot create resource "pods/portforward"`)),
		"unauthorized": apierrors.NewUnauthorized("token expired"),
		"not found":    apierrors.NewNotFound(podsResource, "web-0"),
	} {
		t.Run(name, func(t *testing.T) {
			h, fw := newForwardHarness(t, engine.Options{})
			var f engine.Forward
			h.must(t, engine.OriginGUI, "pod.portForward", map[string]any{"ref": web0, "port": 8080}, &f)
			fw.fail(cause)
			fw.cut()
			failed := waitForward(t, h.s, f.Ref, "failed", func(f engine.Forward) bool { return f.Phase == engine.ForwardFailed })
			if failed.Listening() || !strings.Contains(failed.Message, "not retried") || !strings.Contains(failed.Message, cause.Error()) {
				t.Errorf("failed forward %+v", failed)
			}
			if !portFree(t, f.Address) {
				t.Errorf("the local port %s is not freed", f.Address)
			}
			n := len(fw.dialed())
			time.Sleep(2*time.Second + 200*time.Millisecond) // longer than the first backoff
			if got := len(fw.dialed()); got != n {
				t.Errorf("%d dials after the failure, was %d", got, n)
			}
		})
	}
}

// TestPortForwardRetriesTransientErrors: a network error, a server error and a pod that is not
// running yet (400) keep the forward Reconnecting, and it connects again once they pass.
func TestPortForwardRetriesTransientErrors(t *testing.T) {
	for name, cause := range map[string]error{
		"network":     errors.New("dial tcp 10.0.0.1:443: connect: connection refused"),
		"unavailable": apierrors.NewServiceUnavailable("the apiserver is restarting"),
		"timeout":     apierrors.NewTimeoutError("too slow", 1),
		"not running": apierrors.NewBadRequest("pod does not have a host assigned"),
	} {
		t.Run(name, func(t *testing.T) {
			h, fw := newForwardHarness(t, engine.Options{})
			var f engine.Forward
			h.must(t, engine.OriginGUI, "pod.portForward", map[string]any{"ref": web0, "port": 8080}, &f)
			fw.fail(cause)
			fw.cut()
			waitForward(t, h.s, f.Ref, "reconnecting", func(f engine.Forward) bool { return f.Phase == engine.ForwardReconnecting })
			time.Sleep(100 * time.Millisecond)
			if cur, err := h.s.ForwardOf(f.Ref); err != nil || cur.Phase == engine.ForwardFailed {
				t.Fatalf("a transient error failed the forward: %+v %v", cur, err)
			}
			fw.fail(nil)
			waitForward(t, h.s, f.Ref, "active again", func(f engine.Forward) bool { return f.Phase == engine.ForwardActive })
		})
	}
}

// TestPortForwardThatWaitedFailsOnAPermanentError is the case D-20 names: a forward that first
// waited for its pod, and then could not connect, used to stay Reconnecting for ever.
func TestPortForwardThatWaitedFailsOnAPermanentError(t *testing.T) {
	h, fw := newForwardHarness(t, engine.Options{})
	pending := forwardPod("web-0", false)
	_ = unstructured.SetNestedField(pending.Object, "Pending", "status", "phase")
	h.src.Set(testkit.PodsGVR, pending)
	var f engine.Forward
	h.must(t, engine.OriginGUI, "pod.portForward", map[string]any{"ref": web0, "port": 8080}, &f)
	if f.Phase != engine.ForwardWaiting {
		t.Fatalf("started %+v", f)
	}
	fw.fail(apierrors.NewForbidden(podsResource, "web-0", errors.New("no pods/portforward")))
	h.src.Set(testkit.PodsGVR, forwardPod("web-0", true))
	failed := waitForward(t, h.s, f.Ref, "failed", func(f engine.Forward) bool { return f.Phase == engine.ForwardFailed })
	if !strings.Contains(failed.Message, "forbidden") {
		t.Errorf("failed forward %+v", failed)
	}
	if !portFree(t, f.Address) {
		t.Errorf("the local port %s is not freed", f.Address)
	}
}
