package agent

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// The connection test (the owner's requirement of 2026-10-14 late evening): each layer on its own,
// in order, never with cluster data. TCP: the host and port answer. HTTP: an HTTP(S) answer from
// the API, the TLS handshake included (any status passes, 401 too: the server is there). Auth: the
// models list answers without 401 or 403, with the key. Tools: a minimal chat request offering one
// harmless tool (get_time) and asking for it returns a tool call. A failed layer leaves the later
// ones not reached. Each layer has its own bound.

// Bounds of the test's layers. A local model can take half a minute and more to load for the first
// request, so the tool layer waits longest.
var (
	testTCPTimeout   = 5 * time.Second
	testHTTPTimeout  = 10 * time.Second
	testAuthTimeout  = 15 * time.Second
	testToolsTimeout = 120 * time.Second
)

// testToolName is the harmless tool of the Tools layer.
const testToolName = "get_time"

// TestConnection implements engine.ChatAgent.
func (a *Agent) TestConnection(ctx context.Context, c engine.ChatConnection, apiKey *string, progress func([]engine.ChatTestStep)) []engine.ChatTestStep {
	steps := make([]engine.ChatTestStep, len(engine.ChatTestLayers))
	for i, l := range engine.ChatTestLayers {
		steps[i] = engine.ChatTestStep{Layer: l, State: engine.ChatTestPending}
	}
	report := func() {
		if progress != nil {
			progress(slices.Clone(steps))
		}
	}
	report()
	fail := func(i int, detail string) []engine.ChatTestStep {
		steps[i].State, steps[i].Detail = engine.ChatTestFail, detail
		for j := i + 1; j < len(steps); j++ {
			steps[j].State = engine.ChatTestNotReached
		}
		report()
		return slices.Clone(steps)
	}

	ep, err := endpointFor(c.Kind, c.BaseURL)
	if err != nil {
		return fail(0, err.Error())
	}
	if !validKind(c.Kind) {
		return fail(0, "unknown connection kind")
	}
	key, err := a.testKey(c, apiKey)
	if err != nil {
		return fail(0, err.Error())
	}
	if _, err := normalize(c); errors.Is(err, ErrCAFileScheme) || errors.Is(err, ErrCAFilePath) {
		return fail(0, err.Error())
	}
	// The CA file is read now; a file that cannot be used fails the TLS layer, after TCP.
	cl, clErr := a.clientFor(c, key)
	layers := []func(context.Context) (string, error){
		func(ctx context.Context) (string, error) { return testTCP(ctx, ep) },
		func(ctx context.Context) (string, error) {
			if clErr != nil {
				return "", clErr
			}
			return a.testHTTP(ctx, cl)
		},
		func(ctx context.Context) (string, error) { return testAuth(ctx, cl, c.Model) },
		func(ctx context.Context) (string, error) { return testTools(ctx, cl, c.Model) },
	}
	bounds := []time.Duration{testTCPTimeout, testHTTPTimeout, testAuthTimeout, testToolsTimeout}
	for i, run := range layers {
		steps[i].State = engine.ChatTestRunning
		report()
		start := time.Now()
		detail, err := a.testLayer(ctx, bounds[i], run)
		steps[i].Duration = time.Since(start).Round(time.Millisecond)
		if err != nil {
			return fail(i, err.Error())
		}
		steps[i].State, steps[i].Detail = engine.ChatTestPass, detail
		if engine.ChatTestLayers[i] == engine.ChatTestHTTP && ep.plainRemote {
			steps[i].Warning = engine.ChatPlainHTTPWarning // the owner's decision of 2026-10-15 (H-95)
		}
		report()
	}
	return slices.Clone(steps)
}

// testLayer runs one layer within its bound, a panic turned into a failure.
func (a *Agent) testLayer(ctx context.Context, bound time.Duration, run func(context.Context) (string, error)) (detail string, err error) {
	ctx, cancel := context.WithTimeout(ctx, bound)
	defer cancel()
	defer func() {
		if r := recover(); r != nil {
			a.log.Error("connection test panicked", "panic", fmt.Sprint(r))
			detail, err = "", errors.New("internal error in Cogpit")
		}
	}()
	detail, err = run(ctx)
	if err != nil && errors.Is(context.Cause(ctx), context.DeadlineExceeded) {
		err = fmt.Errorf("no answer within %s", bound)
	}
	return detail, err
}

// testKey is the key the test uses: the given one, else the stored one of c.ID.
func (a *Agent) testKey(c engine.ChatConnection, apiKey *string) (string, error) {
	if apiKey != nil {
		return *apiKey, nil
	}
	if c.ID == "" {
		return "", nil
	}
	a.mu.Lock()
	i := slices.IndexFunc(a.conns, func(x engine.ChatConnection) bool { return x.ID == c.ID })
	var stored engine.ChatConnection
	if i >= 0 {
		stored = a.conns[i]
	}
	a.mu.Unlock()
	if i < 0 {
		return "", nil
	}
	return a.keyFor(stored)
}

// hostPort is the endpoint's host and port, the scheme's default port filled in.
func hostPort(ep endpoint) string {
	u, err := url.Parse(ep.models)
	if err != nil {
		return ep.host
	}
	if u.Port() != "" {
		return u.Host
	}
	if u.Scheme == schemeHTTPS {
		return net.JoinHostPort(u.Hostname(), "443")
	}
	return net.JoinHostPort(u.Hostname(), "80")
}

// dialTCP opens the TCP layer's connection; tests route it to a fake endpoint.
var dialTCP = (&net.Dialer{}).DialContext

func testTCP(ctx context.Context, ep endpoint) (string, error) {
	addr := hostPort(ep)
	conn, err := dialTCP(ctx, "tcp", addr)
	if err != nil {
		var dnsErr *net.DNSError
		if errors.As(err, &dnsErr) {
			return "", fmt.Errorf("%s: the host name is not known", addr)
		}
		if errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
			return "", fmt.Errorf("%s: no answer", addr)
		}
		return "", fmt.Errorf("%s: %s", addr, shortNetError(err))
	}
	_ = conn.Close()
	return addr + " answers", nil
}

// shortNetError is the last part of a dial error ("connection refused").
func shortNetError(err error) string {
	s := err.Error()
	if i := strings.LastIndex(s, ": "); i >= 0 {
		s = s[i+2:]
	}
	return s
}

func (a *Agent) testHTTP(ctx context.Context, cl *client) (string, error) {
	resp, err := cl.get(ctx, cl.ep.models, false)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxModelsBytes))
	detail := fmt.Sprintf("HTTP %d", resp.StatusCode)
	switch {
	case resp.TLS != nil:
		detail += ", " + tlsVersion(resp.TLS.Version)
	case cl.ep.plainRemote:
		detail += ", plain http to another machine" // TestConnection adds the warning
	default:
		detail += ", plain http"
		if cl.key != "" {
			detail += " (the API key travels unencrypted)"
		}
	}
	return detail, nil
}

func tlsVersion(v uint16) string {
	switch v {
	case tls.VersionTLS13:
		return "TLS 1.3"
	case tls.VersionTLS12:
		return "TLS 1.2"
	}
	return "TLS " + tls.VersionName(v)
}

func testAuth(ctx context.Context, cl *client, model string) (string, error) {
	models, err := cl.listModels(ctx)
	if err != nil {
		var he *HTTPError
		if errors.As(err, &he) && he.Status == http.StatusNotFound {
			return "", fmt.Errorf("no models list at %s (HTTP 404): check the base URL and the kind", cl.ep.models)
		}
		if errors.As(err, &he) && (he.Status == http.StatusUnauthorized || he.Status == http.StatusForbidden) && cl.key == "" {
			return "", fmt.Errorf("the endpoint wants an API key (HTTP %d)", he.Status)
		}
		return "", err
	}
	detail := fmt.Sprintf("%d models", len(models))
	if cl.key == "" {
		detail += ", no key needed"
	} else {
		detail += ", key accepted"
	}
	if model != "" && !slices.Contains(models, model) {
		detail += fmt.Sprintf("; %q is not among them", model)
	}
	return detail, nil
}

func testTools(ctx context.Context, cl *client, model string) (string, error) {
	if model == "" {
		return "", errors.New("no model chosen")
	}
	req := chatRequest{
		Model: model,
		Messages: []message{
			{Role: "system", Content: "You are testing a tool connection. Call the get_time tool; do not answer in text."},
			{Role: "user", Content: "What time is it? Use the get_time tool."},
		},
		Tools: []toolDef{{Type: typeFunction, Function: toolDefFunc{Name: testToolName,
			Description: "Returns the current time.", Parameters: json.RawMessage(`{"type":"object","properties":{}}`)}}},
		Stream: true,
	}
	body, _ := json.Marshal(req)
	resp, err := cl.complete(ctx, body, nil)
	if err != nil {
		return "", err
	}
	for _, tc := range resp.ToolCalls {
		if tc.Function.Name == testToolName {
			return model + " called " + testToolName, nil
		}
	}
	if len(resp.ToolCalls) > 0 {
		return "", fmt.Errorf("the model called %q instead of %s", resp.ToolCalls[0].Function.Name, testToolName)
	}
	return "", errors.New("the model answered without a tool call: tool calling may be off for this model or this server")
}
