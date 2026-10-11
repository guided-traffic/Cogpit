package terminal

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"unicode"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/streaming/pkg/httpstream"
)

// DefaultExecCommand is the command a pod shell runs when ExecOptions.Command is empty: the
// first shell the image has, after clearing the screen.
var DefaultExecCommand = []string{"sh", "-c", "clear; (bash || ash || sh)"}

// maxExecMessage bounds the error text written into a pod shell's output when it ends.
const maxExecMessage = 512

// ExecOptions select a container to run a shell in.
type ExecOptions struct {
	Namespace string
	Pod       string
	// Container is the container to run in. Empty leaves the choice to the API server, which
	// accepts it only for a pod with one container; callers pick the default container from the
	// cache (the kubectl.kubernetes.io/default-container annotation, else the first container).
	Container string
	Command   []string // empty: DefaultExecCommand
}

// StartPodExec opens an interactive shell in a container through the API server's exec
// subresource, with a TTY and resize support.
//
// It returns at once; the stream connects in the background. It uses the executor kubectl 0.37
// uses: WebSocket (protocol v5) first, SPDY when the upgrade fails or an HTTPS proxy is in the
// way. When the stream ends with an error (connection refused, forbidden, no such container, a
// non-zero exit), the error is written into the output as "[cogpit] shell ended: …" so the user
// sees it, and Read then returns io.EOF. The context bounds the whole session.
func StartPodExec(ctx context.Context, config *rest.Config, client kubernetes.Interface, opts ExecOptions) (Session, error) {
	if opts.Namespace == "" || opts.Pod == "" {
		return nil, errors.New("pod shell: namespace and pod are required")
	}
	if config == nil || client == nil {
		return nil, errors.New("pod shell: no cluster connection")
	}
	command := opts.Command
	if len(command) == 0 {
		command = DefaultExecCommand
	}
	req := client.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(opts.Namespace).
		Name(opts.Pod).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: opts.Container,
			Command:   command,
			Stdin:     true,
			Stdout:    true,
			Stderr:    false, // with a TTY, stderr arrives on stdout
			TTY:       true,
		}, scheme.ParameterCodec)

	exec, err := newExecutor(config, req)
	if err != nil {
		return nil, fmt.Errorf("pod shell %s/%s: %w", opts.Namespace, opts.Pod, err)
	}
	return startStream(ctx, exec), nil
}

// newExecutor builds the executor kubectl exec builds (k8s.io/kubectl/pkg/cmd/exec,
// createExecutor): WebSocket with a fallback to SPDY.
func newExecutor(config *rest.Config, req *rest.Request) (remotecommand.Executor, error) {
	spdy, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return nil, err
	}
	// RFC 6455 §4.1: the WebSocket handshake is a GET.
	ws, err := remotecommand.NewWebSocketExecutor(config, "GET", req.URL().String())
	if err != nil {
		return nil, err
	}
	return remotecommand.NewFallbackExecutor(ws, spdy, func(err error) bool {
		return httpstream.IsUpgradeFailure(err) || httpstream.IsHTTPSProxyError(err)
	})
}

// execSession is a TTY stream to a process in a container.
type execSession struct {
	input   *inputQueue
	outR    *io.PipeReader
	outW    *io.PipeWriter
	sizes   *sizeQueue
	cancel  context.CancelFunc
	done    chan struct{}
	closing atomic.Bool

	closeOnce sync.Once
}

// startStream runs exec in the background and returns the session around it.
func startStream(ctx context.Context, exec remotecommand.Executor) *execSession {
	ctx, cancel := context.WithCancel(ctx)
	outR, outW := io.Pipe()
	s := &execSession{
		input:  newInputQueue(),
		outR:   outR,
		outW:   outW,
		sizes:  newSizeQueue(),
		cancel: cancel,
		done:   make(chan struct{}),
	}
	go s.run(ctx, exec)
	return s
}

func (s *execSession) run(ctx context.Context, exec remotecommand.Executor) {
	defer close(s.done)
	defer func() {
		// Whatever ended the stream, release everything the executor's goroutines wait on.
		_ = s.input.Close()
		s.sizes.stop()
		s.cancel()
	}()
	var err error
	defer func() {
		if err != nil && !s.closing.Load() {
			_, _ = io.WriteString(s.outW, "\r\n[cogpit] shell ended: "+execMessage(err)+"\r\n")
		}
		_ = s.outW.Close()
	}()
	defer recoverTo("pod shell stream", func(e error) { err = e })

	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:             s.input,
		Stdout:            s.outW,
		Tty:               true,
		TerminalSizeQueue: s.sizes,
	})
}

// execMessage makes a stream error fit for the terminal: one line, no control characters (the
// text can come from the cluster), bounded.
func execMessage(err error) string {
	msg := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, err.Error())
	if len(msg) > maxExecMessage {
		msg = strings.ToValidUTF8(msg[:maxExecMessage], "") + "…"
	}
	return msg
}

// Read returns the container's output; io.EOF once the stream has ended and its output is read.
func (s *execSession) Read(p []byte) (int, error) {
	n, err := s.outR.Read(p)
	if errors.Is(err, io.ErrClosedPipe) {
		err = io.EOF
	}
	return n, err
}

// Write queues keystrokes; they reach the container once the stream is connected.
func (s *execSession) Write(p []byte) (int, error) {
	return s.input.Write(p)
}

// Resize sends the terminal size to the container's TTY. The latest size wins; one set before
// the stream connects is sent when it does.
func (s *execSession) Resize(cols, rows uint16) error {
	if cols == 0 || rows == 0 {
		return fmt.Errorf("pod shell: invalid size %dx%d", cols, rows)
	}
	if !s.sizes.set(remotecommand.TerminalSize{Width: cols, Height: rows}) {
		return errors.New("pod shell: session ended")
	}
	return nil
}

// Done is closed when the stream has ended and its last output was taken by the reader, or
// after Close.
func (s *execSession) Done() <-chan struct{} { return s.done }

// Close ends the stream; the container sees its TTY close. It does not wait for the stream's
// goroutines.
func (s *execSession) Close() error {
	s.closeOnce.Do(func() {
		s.closing.Store(true)
		s.cancel()
		_ = s.input.Close()
		s.sizes.stop()
		// Unblocks the executor's stdout copy and our reader.
		_ = s.outR.Close()
	})
	return nil
}

// sizeQueue is the remotecommand.TerminalSizeQueue behind Resize. It holds only the latest size:
// intermediate sizes of a window drag do not matter to the container.
type sizeQueue struct {
	mu      sync.Mutex
	latest  *remotecommand.TerminalSize
	stopped bool
	notify  chan struct{} // capacity 1: a pending wake-up
	done    chan struct{}
}

func newSizeQueue() *sizeQueue {
	return &sizeQueue{notify: make(chan struct{}, 1), done: make(chan struct{})}
}

// set records size; false once the queue has stopped.
func (q *sizeQueue) set(size remotecommand.TerminalSize) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.stopped {
		return false
	}
	q.latest = &size
	select {
	case q.notify <- struct{}{}:
	default:
	}
	return true
}

// Next implements remotecommand.TerminalSizeQueue: it blocks until a new size is set and returns
// nil once the queue has stopped.
func (q *sizeQueue) Next() *remotecommand.TerminalSize {
	for {
		q.mu.Lock()
		if q.stopped {
			q.mu.Unlock()
			return nil
		}
		if size := q.latest; size != nil {
			q.latest = nil
			q.mu.Unlock()
			return size
		}
		q.mu.Unlock()
		select {
		case <-q.notify:
		case <-q.done:
		}
	}
}

func (q *sizeQueue) stop() {
	q.mu.Lock()
	defer q.mu.Unlock()
	if !q.stopped {
		q.stopped = true
		close(q.done)
	}
}
