package testkit

import (
	"bytes"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

// FreezableProxy is a TCP proxy in front of a server (a FakeAPIServer) that can go silent, as a
// cluster does for a computer that slept or lost its network (D-58): frozen, it accepts
// connections and reads what clients send but forwards nothing either way, so that requests wait
// for an answer that never comes, as on a dead TCP connection or a peer behind a dropped route;
// thawed, it closes the connections it froze (the peer forgot them, and answers a resent packet
// with a reset) and forwards new ones again. It listens on loopback.
type FreezableProxy struct {
	ln     net.Listener
	target string

	// Addr is the proxy's host:port; URL its https URL (the FakeAPIServer's certificate is for
	// 127.0.0.1, so a client trusts it through the proxy).
	Addr, URL string

	mu      sync.Mutex
	frozen  bool
	conns   map[*proxyConn]struct{}
	accepts int
}

type proxyConn struct {
	client, server net.Conn
	frozenAt       bool // it was open when the proxy froze, or opened while frozen
}

// NewFreezableProxy starts a proxy to target (host:port). It closes with the test.
func NewFreezableProxy(t testing.TB, target string) *FreezableProxy {
	t.Helper()
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	p := &FreezableProxy{ln: ln, target: target, Addr: ln.Addr().String(), conns: map[*proxyConn]struct{}{}}
	p.URL = "https://" + p.Addr
	go p.accept()
	t.Cleanup(p.Close)
	return p
}

// ProxyKubeconfig returns the FakeAPIServer's kubeconfig with its server replaced by the proxy's.
func (p *FreezableProxy) ProxyKubeconfig(s *FakeAPIServer, contextName, namespace string) []byte {
	return bytes.ReplaceAll(s.Kubeconfig(contextName, namespace), []byte(s.URL), []byte(p.URL))
}

// Freeze makes the proxy go silent: the open connections forward nothing more, new ones are
// accepted and then hang.
func (p *FreezableProxy) Freeze() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.frozen {
		return
	}
	p.frozen = true
	for c := range p.conns {
		c.frozenAt = true
	}
}

// Thaw closes every connection the freeze caught and forwards again.
func (p *FreezableProxy) Thaw() {
	p.mu.Lock()
	if !p.frozen {
		p.mu.Unlock()
		return
	}
	p.frozen = false
	var dead []*proxyConn
	for c := range p.conns {
		if c.frozenAt {
			dead = append(dead, c)
			delete(p.conns, c)
		}
	}
	p.mu.Unlock()
	for _, c := range dead {
		c.close()
	}
}

// Frozen reports whether the proxy is frozen.
func (p *FreezableProxy) Frozen() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.frozen
}

// Accepts returns how many connections the proxy accepted so far.
func (p *FreezableProxy) Accepts() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.accepts
}

// Close stops the proxy and closes every connection.
func (p *FreezableProxy) Close() {
	_ = p.ln.Close()
	p.mu.Lock()
	conns := p.conns
	p.conns = map[*proxyConn]struct{}{}
	p.frozen = false
	p.mu.Unlock()
	for c := range conns {
		c.close()
	}
}

func (c *proxyConn) close() {
	_ = c.client.Close()
	if c.server != nil {
		_ = c.server.Close()
	}
}

func (p *FreezableProxy) accept() {
	for {
		client, err := p.ln.Accept()
		if err != nil {
			return
		}
		c := &proxyConn{client: client}
		p.mu.Lock()
		p.accepts++
		c.frozenAt = p.frozen
		p.conns[c] = struct{}{}
		p.mu.Unlock()
		go p.serve(c)
	}
}

// serve connects to the target once the proxy is not frozen and copies both ways, each chunk only
// while the proxy is thawed and the connection was not caught by a freeze.
func (p *FreezableProxy) serve(c *proxyConn) {
	if !p.waitLive(c) {
		_, _ = io.Copy(io.Discard, c.client) // hang: read and drop until the thaw closes it
		return
	}
	server, err := net.DialTimeout("tcp", p.target, 5*time.Second)
	if err != nil {
		_ = c.client.Close()
		return
	}
	p.mu.Lock()
	c.server = server
	p.mu.Unlock()
	go p.pipe(c, server, c.client)
	p.pipe(c, c.client, server)
}

// waitLive reports whether c may forward: false for a connection the freeze caught.
func (p *FreezableProxy) waitLive(c *proxyConn) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return !c.frozenAt
}

func (p *FreezableProxy) pipe(c *proxyConn, dst io.Writer, src io.Reader) {
	buf := make([]byte, 32<<10)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if !p.waitLive(c) {
				_, _ = io.Copy(io.Discard, src) // caught by a freeze: silent from now on
				return
			}
			if _, werr := dst.Write(buf[:n]); werr != nil {
				c.close()
				return
			}
		}
		if err != nil {
			c.close()
			return
		}
	}
}
