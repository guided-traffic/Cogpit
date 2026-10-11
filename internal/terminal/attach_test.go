package terminal

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// TestAttachRequestParams: the request is kubectl attach's: POST on the pod's attach subresource
// for the container, stdout always, stdin and a TTY only as the container has them, stderr apart
// only without a TTY.
func TestAttachRequestParams(t *testing.T) {
	client, err := kubernetes.NewForConfig(&rest.Config{Host: "https://cluster.invalid"})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		opts AttachOptions
		want map[string]string // query parameters; "" means absent
	}{
		{AttachOptions{Namespace: "shop", Pod: "web-0", Container: "repl", Stdin: true, TTY: true},
			map[string]string{"container": "repl", "stdin": "true", "stdout": "true", "stderr": "", "tty": "true"}},
		{AttachOptions{Namespace: "shop", Pod: "web-0", Container: "app"},
			map[string]string{"container": "app", "stdin": "", "stdout": "true", "stderr": "true", "tty": ""}},
		{AttachOptions{Namespace: "shop", Pod: "web-0", Container: "worker", Stdin: true},
			map[string]string{"container": "worker", "stdin": "true", "stdout": "true", "stderr": "true", "tty": ""}},
	} {
		req := attachRequest(client, tc.opts)
		u := req.URL()
		if u.Path != "/api/v1/namespaces/shop/pods/web-0/attach" {
			t.Errorf("%+v: path %s", tc.opts, u.Path)
		}
		q := u.Query()
		for k, v := range tc.want {
			if got := q.Get(k); got != v {
				t.Errorf("%+v: %s=%q, want %q (%s)", tc.opts, k, got, v, u.RawQuery)
			}
		}
		if q.Has("command") {
			t.Errorf("%+v: attach runs no command: %s", tc.opts, u.RawQuery)
		}
	}
}

// printingContainer is a container's main process without a TTY: it reports what the stream
// gives it, prints to stdout and stderr, and echoes stdin lines back.
func printingContainer(seen chan<- remotecommand.StreamOptions) fakeExecutor {
	return func(ctx context.Context, opts remotecommand.StreamOptions) error {
		seen <- opts
		if opts.Stdout == nil || opts.Stderr == nil {
			return errors.New("fake: want stdout and stderr")
		}
		_, _ = io.WriteString(opts.Stdout, "line one\nline two\n")
		_, _ = io.WriteString(opts.Stderr, "warning on stderr\n")
		if opts.Stdin == nil {
			<-ctx.Done()
			return nil
		}
		buf := make([]byte, 64)
		n, err := opts.Stdin.Read(buf)
		if err != nil {
			return err
		}
		_, _ = io.WriteString(opts.Stdout, "got "+strings.ReplaceAll(string(buf[:n]), "\n", "<LF>")+"\n")
		<-ctx.Done()
		return nil
	}
}

// TestAttachWithoutTTY: without a TTY the session shows stdout and stderr with the carriage
// returns a terminal needs, sends no size, turns Enter into a line feed for stdin, and drops
// keystrokes when the container has no stdin; its first line says which.
func TestAttachWithoutTTY(t *testing.T) {
	seen := make(chan remotecommand.StreamOptions, 1)
	sess := startStreamMode(context.Background(), printingContainer(seen), streamMode{
		stdin: true, what: "attach", notice: attachNotice(AttachOptions{Pod: "web-0", Container: "worker", Stdin: true}),
	})
	defer sess.Close()
	out := record(sess)
	opts := <-seen
	if opts.Tty || opts.TerminalSizeQueue != nil || opts.Stdin == nil {
		t.Errorf("stream options %+v", opts)
	}
	out.waitFor(t, "warning on stderr\r\n", 5*time.Second)
	if err := sess.Resize(80, 24); err != nil {
		t.Errorf("Resize without a TTY: %v", err)
	}
	if _, err := sess.Write([]byte("hello\r")); err != nil {
		t.Fatal(err)
	}
	out.waitFor(t, "got hello<LF>\r\n", 5*time.Second)
	got := out.String()
	if !strings.HasPrefix(got, "[cogpit] attached to web-0/worker without a terminal") ||
		!strings.Contains(got, "line one\r\nline two\r\n") {
		t.Errorf("output %q", got)
	}

	seen = make(chan remotecommand.StreamOptions, 1)
	noStdin := startStreamMode(context.Background(), printingContainer(seen), streamMode{
		what: "attach", notice: attachNotice(AttachOptions{Pod: "web-0", Container: "app"}),
	})
	out = record(noStdin)
	if opts := <-seen; opts.Stdin != nil {
		t.Error("stdin was sent for a container without one")
	}
	if n, err := noStdin.Write([]byte("ignored\r")); n != len("ignored\r") || err != nil {
		t.Errorf("Write without stdin: %d, %v", n, err)
	}
	out.waitFor(t, "warning on stderr", 5*time.Second)
	if !strings.HasPrefix(out.String(), "[cogpit] attached to web-0/app, output only") {
		t.Errorf("output %q", out.String())
	}
	_ = noStdin.Close()
	if _, err := noStdin.Write([]byte("x")); err == nil {
		t.Error("Write after Close succeeded")
	}
	if err := out.waitEnd(t, 5*time.Second); err != io.EOF {
		t.Errorf("Read after Close: %v", err)
	}
}

// TestAttachWithTTY: with a TTY the stream is the shell's: stdin, the size queue, one output.
func TestAttachWithTTY(t *testing.T) {
	sizes := make(chan remotecommand.TerminalSize, 8)
	connected := make(chan struct{})
	close(connected)
	sess := startStreamMode(context.Background(), echoContainer(sizes, connected), streamMode{
		tty: true, stdin: true, what: "attach", notice: attachNotice(AttachOptions{Pod: "repl-0", Container: "repl", Stdin: true, TTY: true}),
	})
	defer sess.Close()
	out := record(sess)
	if err := sess.Resize(100, 30); err != nil {
		t.Fatal(err)
	}
	if _, err := sess.Write([]byte("hi\n")); err != nil {
		t.Fatal(err)
	}
	out.waitFor(t, "you said: hi", 5*time.Second)
	if got := <-sizes; got.Width != 100 || got.Height != 30 {
		t.Errorf("size %+v", got)
	}
	if !strings.HasPrefix(out.String(), "[cogpit] attached to repl-0/repl; if no prompt shows, press Enter\r\n") {
		t.Errorf("output %q", out.String())
	}
	if _, err := sess.Write([]byte("exit\n")); err != nil {
		t.Fatal(err)
	}
	if err := out.waitEnd(t, 5*time.Second); err != io.EOF {
		t.Errorf("end: %v", err)
	}
}

// TestAttachShowsStreamErrors: a refused attach ends with the error, named as an attach.
func TestAttachShowsStreamErrors(t *testing.T) {
	sess := startStreamMode(context.Background(), fakeExecutor(func(context.Context, remotecommand.StreamOptions) error {
		return errors.New("pods \"web-0\" is forbidden: cannot create resource \"pods/attach\"\x1b]52;c;QUFB\a")
	}), streamMode{what: "attach"})
	defer sess.Close()
	out := record(sess)
	_ = out.waitEnd(t, 5*time.Second)
	got := out.String()
	if !strings.HasPrefix(got, "\r\n[cogpit] attach ended: pods \"web-0\" is forbidden") || strings.Contains(got, "\x1b") {
		t.Errorf("output %q", got)
	}
}

func TestStartPodAttachValidates(t *testing.T) {
	client := fake.NewClientset()
	cfg := &rest.Config{Host: "https://127.0.0.1:1"}
	for name, opts := range map[string]AttachOptions{"no namespace": {Pod: "p"}, "no pod": {Namespace: "ns"}} {
		if _, err := StartPodAttach(context.Background(), cfg, client, opts); err == nil {
			t.Errorf("%s: StartPodAttach succeeded", name)
		}
	}
	if _, err := StartPodAttach(context.Background(), nil, client, AttachOptions{Namespace: "ns", Pod: "p"}); err == nil {
		t.Error("StartPodAttach without a config succeeded")
	}
	if got := attachNotice(AttachOptions{Pod: "p", Container: "evil\x1b]0;x\a"}); strings.ContainsAny(got[:len(got)-2], "\x1b\a\r\n") {
		t.Errorf("notice keeps control characters: %q", got)
	}
}
