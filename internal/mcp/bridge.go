package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// The stdio bridge (architecture §7, `cogpit-cli mcp`): an MCP server on a local transport whose
// tools are the running app's, each call forwarded to the app's Streamable HTTP endpoint with the
// token from the app's token file. The app's engine dispatches it with origin ExternalAgent, so
// its confirmation dialog and audit log apply as for any external agent.

// codeAppUnavailable marks a forwarded call that never reached a running Cogpit.
const codeAppUnavailable = "appUnavailable"

// ErrAppNotRunning says that no running Cogpit serves MCP here: there is no endpoint file, or
// nothing listens at its address any more (Cogpit crashed). Match it with errors.Is.
var ErrAppNotRunning = errors.New("no running Cogpit serves MCP")

// BridgeOptions configure DialApp.
type BridgeOptions struct {
	// TokenFile is the app's token file; the endpoint file is beside it. Empty means
	// <user config dir>/Cogpit/mcp-token.
	TokenFile string
	// Version is the bridge's version, reported to the app as its client version.
	Version string
	// Logger receives the bridge's logs; nil means slog.Default(). The token is never logged.
	Logger *slog.Logger
}

// Bridge serves a running app's tools over a local transport. Create it with DialApp, call Serve,
// and Close it.
type Bridge struct {
	opts      BridgeOptions
	tokenPath string
	log       *slog.Logger

	tools        []*sdk.Tool
	instructions string
	appVersion   string

	mu sync.Mutex
	up *upstream // the current connection to the app; replaced when the app restarts
}

// upstream is one connection to one start of the app.
type upstream struct {
	ep   Endpoint
	sess *sdk.ClientSession
}

// DialApp connects to the running app's endpoint and lists its tools. It returns ErrAppNotRunning
// when there is no endpoint file or nothing listens at its address; any other error means
// something answered there that did not take the token, or the files are not Cogpit's.
func DialApp(ctx context.Context, opts BridgeOptions) (*Bridge, error) {
	tokenPath := opts.TokenFile
	if tokenPath == "" {
		p, err := defaultTokenPath()
		if err != nil {
			return nil, err
		}
		tokenPath = p
	}
	if opts.Version == "" {
		opts.Version = "dev"
	}
	log := opts.Logger
	if log == nil {
		log = slog.Default()
	}
	b := &Bridge{opts: opts, tokenPath: tokenPath, log: log.With("component", "mcp-bridge")}
	up, err := b.dial(ctx)
	if err != nil {
		return nil, err
	}
	b.up = up
	ir := up.sess.InitializeResult()
	if ir != nil {
		b.instructions = ir.Instructions
		if ir.ServerInfo != nil {
			b.appVersion = ir.ServerInfo.Version
		}
	}
	for tool, err := range up.sess.Tools(ctx, nil) {
		if err != nil {
			_ = up.sess.Close()
			return nil, fmt.Errorf("mcp: list the running Cogpit's tools: %w", err)
		}
		b.tools = append(b.tools, tool)
	}
	b.log.Info("bridging to the running Cogpit", "url", up.ep.URL, "tools", len(b.tools))
	return b, nil
}

// dial reads the endpoint and connects to it. A token the server refuses is read again a few
// times: the app writes a new token and endpoint on every start, and the bridge may have read
// the files of two starts.
func (b *Bridge) dial(ctx context.Context) (*upstream, error) {
	const attempts = 4
	for i := 0; ; i++ {
		ep, err := ReadEndpoint(b.tokenPath)
		if errors.Is(err, ErrNoEndpoint) {
			return nil, fmt.Errorf("%w: %v", ErrAppNotRunning, err)
		}
		if err != nil {
			return nil, err
		}
		d := &delivery{}
		sess, err := b.connect(context.WithValue(ctx, deliveryKey{}, d), ep)
		switch {
		case err == nil:
			return &upstream{ep: ep, sess: sess}, nil
		case ctx.Err() != nil:
			return nil, ctx.Err()
		case d.dialFailed.Load():
			// The files of a Cogpit that is gone (a crash leaves them).
			return nil, fmt.Errorf("%w: nothing listens at %s", ErrAppNotRunning, ep.URL)
		case !d.unauthorized.Load() || i == attempts-1:
			if d.unauthorized.Load() {
				return nil, fmt.Errorf("mcp: the Cogpit at %s does not accept the token in %s", ep.URL, b.tokenPath)
			}
			return nil, fmt.Errorf("mcp: connect to the running Cogpit at %s: %w", ep.URL, err)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
}

// connect opens an MCP client session to ep.
func (b *Bridge) connect(ctx context.Context, ep Endpoint) (*sdk.ClientSession, error) {
	client := sdk.NewClient(&sdk.Implementation{Name: "cogpit-cli", Title: "Cogpit stdio bridge", Version: b.opts.Version}, &sdk.ClientOptions{
		Logger: sdkLogger(b.log),
	})
	return client.Connect(ctx, &sdk.StreamableClientTransport{
		Endpoint:   ep.URL,
		HTTPClient: bearerClient(ep.Token),
		// Cogpit sends no server-initiated messages, and a call's stream ends with the app: a
		// broken stream is not retried, the call fails (and is forwarded again only if it never
		// reached the app, see call).
		DisableStandaloneSSE: true,
		MaxRetries:           -1,
	}, nil)
}

// Serve serves the app's tools over t until the client disconnects or ctx ends: the same names,
// schemas, annotations and instructions, every call forwarded to the app.
func (b *Bridge) Serve(ctx context.Context, t sdk.Transport) error {
	version := b.appVersion
	if version == "" {
		version = b.opts.Version
	}
	srv := sdk.NewServer(&sdk.Implementation{Name: serverName, Title: "Cogpit", Version: version}, &sdk.ServerOptions{
		Instructions: b.instructions,
		Logger:       sdkLogger(b.log),
		Capabilities: &sdk.ServerCapabilities{Tools: &sdk.ToolCapabilities{}},
	})
	for _, tool := range b.tools {
		if err := addTool(srv, tool, b.forward(tool)); err != nil {
			b.log.Error("tool left out of the bridge", "tool", tool.Name, "err", err)
		}
	}
	return srv.Run(ctx, t)
}

// Close ends the connection to the app.
func (b *Bridge) Close() error {
	b.mu.Lock()
	up := b.up
	b.up = nil
	b.mu.Unlock()
	if up == nil {
		return nil
	}
	return up.sess.Close()
}

// forward is a tool's handler: the call goes to the app as it came.
func (b *Bridge) forward(tool *sdk.Tool) sdk.ToolHandler {
	return func(ctx context.Context, req *sdk.CallToolRequest) (res *sdk.CallToolResult, err error) {
		defer func() {
			if r := recover(); r != nil {
				b.log.Error("forwarding a tool call panicked", "tool", tool.Name, "panic", fmt.Sprint(r), "stack", string(debug.Stack()))
				res, err = toolError(codeInternal, fmt.Sprintf("%q failed with an internal error in Cogpit's stdio bridge.", tool.Title)), nil
			}
		}()
		var args json.RawMessage
		if req != nil && req.Params != nil {
			args = req.Params.Arguments
		}
		res, err = b.call(ctx, tool.Name, args)
		if err == nil {
			return res, nil
		}
		var werr *jsonrpc.Error
		if errors.As(err, &werr) {
			return nil, werr // the app's own answer, passed on as it is
		}
		if ctx.Err() != nil {
			return toolError(codeCancelled, fmt.Sprintf("%q was cancelled before it finished (%v). If it was waiting for the "+
				"user's confirmation, it did not run.", tool.Title, ctx.Err())), nil
		}
		return toolError(codeAppUnavailable, unavailable(tool, err)), nil
	}
}

// unavailable says what a model should know when the app could not be reached.
func unavailable(tool *sdk.Tool, err error) string {
	if errors.Is(err, ErrAppNotRunning) {
		return fmt.Sprintf("%q did not run: Cogpit is not running, or the user switched its MCP server off in the status bar. "+
			"Ask the user to start Cogpit or switch the MCP server on.", tool.Title)
	}
	return fmt.Sprintf("%q could not reach Cogpit (%v). If Cogpit quit while the call was running, it may or may not have run; "+
		"check before retrying.", tool.Title, err)
}

// call forwards one call. A call the app never received (nothing listened, or the server refused
// the token: the app restarted with a new port and token, or its server was switched off) is
// sent once more after reading the endpoint again; a call the app did receive is never repeated.
func (b *Bridge) call(ctx context.Context, name string, args json.RawMessage) (*sdk.CallToolResult, error) {
	params := &sdk.CallToolParams{Name: name}
	if len(args) > 0 {
		params.Arguments = args
	}
	up := b.current()
	for attempt := 0; ; attempt++ {
		if up == nil {
			var err error
			if up, err = b.reconnect(ctx, nil); err != nil {
				return nil, err
			}
		}
		d := &delivery{}
		res, err := up.sess.CallTool(context.WithValue(ctx, deliveryKey{}, d), params)
		if err == nil || ctx.Err() != nil || attempt > 0 || !d.undelivered() {
			return res, err
		}
		b.log.Info("the running Cogpit did not take the call; reading its endpoint again", "tool", name)
		if up, err = b.reconnect(ctx, up); err != nil {
			return nil, err
		}
	}
}

func (b *Bridge) current() *upstream {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.up
}

// reconnect replaces failed (nil: none) with a connection to the endpoint the files name now,
// unless another call did so already.
func (b *Bridge) reconnect(ctx context.Context, failed *upstream) (*upstream, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.up != nil && b.up != failed {
		return b.up, nil
	}
	up, err := b.dial(ctx)
	if err != nil {
		return nil, err
	}
	if b.up != nil {
		_ = b.up.sess.Close()
	}
	b.up = up
	b.log.Info("bridging to the running Cogpit", "url", up.ep.URL)
	return up, nil
}

// deliveryKey carries a *delivery in a request's context to bearerTransport.
type deliveryKey struct{}

// delivery records why requests of one call or connect did not reach Cogpit's commands.
type delivery struct {
	dialFailed   atomic.Bool // nothing listened at the address
	unauthorized atomic.Bool // the server refused the token, before any command ran
}

func (d *delivery) undelivered() bool { return d.dialFailed.Load() || d.unauthorized.Load() }

// bearerClient is the HTTP client of one connection to the app: it sends the token on every
// request, follows no redirect (the token goes to the endpoint and nowhere else) and uses no
// proxy.
func bearerClient(token string) *http.Client {
	base := http.DefaultTransport.(*http.Transport).Clone()
	base.Proxy = nil
	return &http.Client{
		Transport:     &bearerTransport{token: token, base: base},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

type bearerTransport struct {
	token string
	base  http.RoundTripper
}

func (t *bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())
	r.Header.Set("Authorization", "Bearer "+t.token)
	resp, err := t.base.RoundTrip(r)
	if d, ok := req.Context().Value(deliveryKey{}).(*delivery); ok {
		var op *net.OpError
		switch {
		case err != nil && errors.As(err, &op) && op.Op == "dial":
			d.dialFailed.Store(true)
		case err == nil && resp.StatusCode == http.StatusUnauthorized:
			d.unauthorized.Store(true)
		}
	}
	return resp, err
}
