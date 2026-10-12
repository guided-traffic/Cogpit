package cluster

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"unicode"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"k8s.io/streaming/pkg/httpstream"
)

// LoopbackAddress is the only address a port forward listens on: IPv4 loopback. Never all
// interfaces, never "::", never a host name that could resolve elsewhere
// (docs/security/trust-boundaries.md, H-2).
const LoopbackAddress = "127.0.0.1"

// maxForwardMessage bounds the error text a pod's port-forward error stream may return: it comes
// from the cluster and ends up in the UI.
const maxForwardMessage = 512

// ListenLoopback opens a port forward's local listener: TCP on 127.0.0.1 and port, or a free
// port the system picks when port is 0. It refuses a listener that did not end up on IPv4
// loopback, so that no future change of the address can widen who may connect.
func ListenLoopback(port uint16) (net.Listener, error) {
	ln, err := net.Listen("tcp4", net.JoinHostPort(LoopbackAddress, strconv.Itoa(int(port))))
	if err != nil {
		return nil, fmt.Errorf("listen on %s:%d: %w", LoopbackAddress, port, err)
	}
	if a, ok := ln.Addr().(*net.TCPAddr); !ok || a.IP.To4() == nil || !a.IP.IsLoopback() {
		_ = ln.Close()
		return nil, fmt.Errorf("the listener is on %s, not on IPv4 loopback", ln.Addr())
	}
	return ln, nil
}

// PortTunnel is one port-forward connection to a pod (the pods/portforward subresource). Each
// local connection it carries is a pair of streams over it, to one port of the pod.
type PortTunnel interface {
	// Forward carries one local connection to port in the pod until either side ends it, and
	// closes conn. An error the pod reports (nothing listens on the port) ends the whole tunnel,
	// as kubectl does, because the kubelet's side may be unusable afterwards; the error says why.
	Forward(conn net.Conn, port uint16) error
	// Done is closed when the connection to the pod is lost or closed.
	Done() <-chan struct{}
	// Close ends the connection and every local connection it carries.
	Close() error
}

// DialPortForward opens a port-forward connection to a pod, as kubectl port-forward 0.37 does:
// SPDY tunnelled over a WebSocket first, plain SPDY when the WebSocket upgrade fails or an HTTPS
// proxy is in the way. It uses the connection's own client configuration (its credentials, TLS
// and proxy). ctx bounds the dial only; the tunnel lives until it is closed or lost.
func (c *Conn) DialPortForward(ctx context.Context, namespace, pod string) (PortTunnel, error) {
	if namespace == "" || pod == "" {
		return nil, errors.New("port forward: namespace and pod are required")
	}
	if c == nil || c.kube == nil || c.config == nil {
		return nil, errors.New("port forward: not connected")
	}
	req := c.kube.CoreV1().RESTClient().Post().Resource("pods").Namespace(namespace).Name(pod).SubResource("portforward")
	t, err := dialPortForward(ctx, c.config, req.URL())
	if err != nil {
		return nil, fmt.Errorf("port forward to %s/%s: %w", namespace, pod, err)
	}
	return t, nil
}

// dialPortForward dials url with the WebSocket dialer and the SPDY fallback, bounded by ctx: the
// dialers take no context, so a dial that outlives ctx is abandoned and its connection closed
// when it arrives.
func dialPortForward(ctx context.Context, config *rest.Config, u *url.URL) (PortTunnel, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	dialer, rec, err := portForwardDialer(config, u)
	if err != nil {
		return nil, err
	}
	type dialed struct {
		conn     httpstream.Connection
		protocol string
		err      error
	}
	ch := make(chan dialed, 1)
	go func() {
		var d dialed
		defer func() {
			if r := recover(); r != nil {
				slog.Default().Error("cluster: port-forward dial panicked", "panic", fmt.Sprint(r), "stack", string(debug.Stack()))
				d = dialed{err: fmt.Errorf("internal error: %v", r)}
			}
			ch <- d
		}()
		d.conn, d.protocol, d.err = dialer.Dial(portforward.PortForwardProtocolV1Name)
	}()
	var d dialed
	select {
	case d = <-ch:
	case <-ctx.Done():
		go func() {
			if late := <-ch; late.conn != nil {
				_ = late.conn.Close()
			}
		}()
		return nil, ctx.Err()
	}
	if d.err != nil {
		return nil, rec.refusal(d.err)
	}
	// select picks at random when the dial and the cancellation are both ready: the
	// cancellation wins, so a cancelled caller never receives a tunnel.
	if err := ctx.Err(); err != nil {
		_ = d.conn.Close()
		return nil, err
	}
	if d.protocol != portforward.PortForwardProtocolV1Name {
		_ = d.conn.Close()
		return nil, fmt.Errorf("the server speaks %q, not %q", d.protocol, portforward.PortForwardProtocolV1Name)
	}
	return newTunnel(d.conn), nil
}

// portForwardDialer builds the dialer kubectl port-forward builds (k8s.io/kubectl/pkg/cmd/
// portforward, createDialer): WebSocket first, SPDY as the fallback.
func portForwardDialer(config *rest.Config, u *url.URL) (httpstream.Dialer, *statusRecorder, error) {
	transport, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		return nil, nil, err
	}
	// The SPDY dialer's upgrade error is a plain error that keeps the server's message and loses
	// its status code: the recorder keeps the code, so a refusal can be told from a hiccup (D-20).
	rec := &statusRecorder{next: transport}
	spdyDialer := spdy.NewDialerForStreaming(upgrader, &http.Client{Transport: rec}, http.MethodPost, u)
	ws, err := portforward.NewSPDYOverWebsocketDialerForStreaming(u, config)
	if err != nil {
		return nil, nil, err
	}
	return portforward.NewFallbackDialerForStreaming(ws, spdyDialer, func(err error) bool {
		return httpstream.IsUpgradeFailure(err) || httpstream.IsHTTPSProxyError(err)
	}), rec, nil
}

// tunnel is a PortTunnel over one streaming connection.
type tunnel struct {
	conn      httpstream.Connection
	requestID atomic.Int64
	done      chan struct{}
	closeOnce sync.Once
}

func newTunnel(conn httpstream.Connection) *tunnel {
	t := &tunnel{conn: conn, done: make(chan struct{})}
	go func() {
		<-conn.CloseChan()
		t.closeOnce.Do(func() { close(t.done) })
	}()
	return t
}

func (t *tunnel) Done() <-chan struct{} { return t.done }

func (t *tunnel) Close() error {
	err := t.conn.Close()
	t.closeOnce.Do(func() { close(t.done) })
	return err
}

// Forward carries one local connection over an error stream and a data stream with the same
// request id, the port-forward protocol's pair (k8s.io/api/core/v1 PortHeader,
// PortForwardRequestIDHeader).
func (t *tunnel) Forward(conn net.Conn, port uint16) (err error) {
	defer func() { _ = conn.Close() }()
	defer func() {
		if r := recover(); r != nil {
			slog.Default().Error("cluster: port-forward connection panicked", "panic", fmt.Sprint(r), "stack", string(debug.Stack()))
			err = fmt.Errorf("internal error: %v", r)
		}
	}()
	if port == 0 {
		return errors.New("port forward: port 0")
	}
	headers := http.Header{}
	headers.Set(corev1.StreamType, corev1.StreamTypeError)
	headers.Set(corev1.PortHeader, strconv.Itoa(int(port)))
	headers.Set(corev1.PortForwardRequestIDHeader, strconv.FormatInt(t.requestID.Add(1)-1, 10))
	errStream, err := t.conn.CreateStream(headers)
	if err != nil {
		return fmt.Errorf("port %d: create the error stream: %w", port, err)
	}
	_ = errStream.Close() // nothing is written to it
	defer t.conn.RemoveStreams(errStream)
	type readResult struct {
		msg []byte
		err error
	}
	errc := make(chan readResult, 1)
	goRecover("error stream", func() {
		res := readResult{err: errStreamPanicked} // what Forward gets if the read panics
		defer func() { errc <- res }()
		res.msg, res.err = io.ReadAll(io.LimitReader(errStream, maxForwardMessage+1))
	})

	headers.Set(corev1.StreamType, corev1.StreamTypeData)
	data, err := t.conn.CreateStream(headers)
	if err != nil {
		return fmt.Errorf("port %d: create the data stream: %w", port, err)
	}
	defer t.conn.RemoveStreams(data)

	remoteDone := make(chan struct{})
	localFailed := make(chan struct{})
	goRecover("data from the pod", func() {
		defer close(remoteDone)
		_, _ = io.Copy(conn, data) // the pod's side to the local client
	})
	goRecover("data to the pod", func() {
		failed := true // also when the copy panics
		defer func() {
			_ = data.Close() // the pod's side sees the end of the client's data
			if failed {
				close(localFailed)
			}
		}()
		_, err := io.Copy(data, conn)
		failed = err != nil && !isClosedError(err)
	})
	select {
	case <-remoteDone:
	case <-localFailed:
	}
	// Reset before waiting for the error stream: unsent data would otherwise hold it up.
	_ = data.Reset()
	res := <-errc
	switch {
	case res.err != nil && !isClosedError(res.err):
		_ = t.Close()
		return fmt.Errorf("port %d: read the error stream: %w", port, res.err)
	case len(res.msg) > 0:
		_ = t.Close()
		return fmt.Errorf("port %d: %s", port, forwardMessage(res.msg))
	}
	return nil
}

// errStreamPanicked is the error-stream read's result when the read panicked.
var errStreamPanicked = errors.New("internal error reading the error stream")

// goRecover runs one stream copy of a forwarded connection on its own goroutine and turns a panic
// into a log line: the goroutine boundary of cluster I/O (AGENTS.md). fn's own defers still run,
// so the copy reports its end to Forward.
func goRecover(what string, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Default().Error("cluster: a port-forward stream panicked", "stream", what, "panic", fmt.Sprint(r), "stack", string(debug.Stack()))
			}
		}()
		fn()
	}()
}

// isClosedError reports the errors of a connection closed on purpose.
func isClosedError(err error) bool {
	return errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) ||
		strings.Contains(strings.ToLower(err.Error()), "use of closed network connection")
}

// forwardMessage makes the pod's error text fit for the UI: one line, no control characters,
// bounded. It comes from the cluster.
func forwardMessage(msg []byte) string {
	s := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, strings.TrimSpace(string(msg)))
	if len(s) > maxForwardMessage {
		s = strings.ToValidUTF8(s[:maxForwardMessage], "") + "…"
	}
	return s
}
