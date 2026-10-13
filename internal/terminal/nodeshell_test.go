package terminal

import (
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"testing"
	"time"

	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// nodeShellMode is the stream mode StartNodeShell uses, with its notice.
var nodeShellMode = streamMode{tty: true, stdin: true, what: "node shell", notice: "[cogpit] node shell on worker-1\r\n"}

// TestNodeShellWaitsThenStreams: the notice comes first, the stream connects only once Before
// returned, and then carries the shell as a pod shell's does.
func TestNodeShellWaitsThenStreams(t *testing.T) {
	release := make(chan struct{})
	streamed := make(chan struct{})
	before := func(ctx context.Context) error {
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	shell := fakeExecutor(func(ctx context.Context, opts remotecommand.StreamOptions) error {
		close(streamed)
		_, err := io.WriteString(opts.Stdout, "root@worker-1:/# ")
		<-ctx.Done()
		return err
	})
	sess := startStreamMode(context.Background(), beforeExecutor{exec: shell, before: before}, nodeShellMode)
	defer sess.Close()
	out := record(sess)
	out.waitFor(t, "[cogpit] node shell on worker-1", 5*time.Second)
	select {
	case <-streamed:
		t.Fatal("the stream connected before the pod ran")
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	out.waitFor(t, "root@worker-1:/# ", 5*time.Second)
}

// TestNodeShellEndsWhenThePodDoesNotStart: Before's error ends the session with its reason, and
// the stream never connects; Close during Before cancels the wait.
func TestNodeShellEndsWhenThePodDoesNotStart(t *testing.T) {
	shell := fakeExecutor(func(context.Context, remotecommand.StreamOptions) error {
		t.Error("the stream connected for a pod that did not start")
		return nil
	})
	failed := func(context.Context) error {
		return errors.New("the pod did not start: its container is waiting: ImagePullBackOff")
	}
	sess := startStreamMode(context.Background(), beforeExecutor{exec: shell, before: failed}, nodeShellMode)
	out := record(sess)
	out.waitFor(t, "[cogpit] node shell ended: the pod did not start: its container is waiting: ImagePullBackOff", 5*time.Second)
	waitClosed(t, sess.Done(), 5*time.Second, "the session")

	waiting := make(chan struct{})
	blocked := func(ctx context.Context) error {
		close(waiting)
		<-ctx.Done()
		return ctx.Err()
	}
	sess = startStreamMode(context.Background(), beforeExecutor{exec: shell, before: blocked}, nodeShellMode)
	_ = record(sess)
	waitClosed(t, waiting, 5*time.Second, "the wait")
	_ = sess.Close()
	waitClosed(t, sess.Done(), 5*time.Second, "the session after Close")
}

// TestStartNodeShellNeedsItsPod: a node shell names its pod and has a connection; its command is
// nsenter into PID 1's namespaces, then the node's own shell.
func TestStartNodeShellNeedsItsPod(t *testing.T) {
	if _, err := StartNodeShell(context.Background(), &rest.Config{Host: "https://127.0.0.1:1"}, fake.NewClientset(), NodeShellOptions{}); err == nil {
		t.Error("a node shell without a pod started")
	}
	if _, err := StartNodeShell(context.Background(), nil, nil, NodeShellOptions{ExecOptions: ExecOptions{Namespace: "kube-system", Pod: "p"}}); err == nil {
		t.Error("a node shell without a connection started")
	}
	if !slices.Equal(NodeShellCommand[:9], []string{"nsenter", "-t", "1", "-m", "-u", "-i", "-n", "-p", "--"}) ||
		!strings.Contains(strings.Join(NodeShellCommand, " "), "exec bash -l") {
		t.Errorf("command %q", NodeShellCommand)
	}
}
