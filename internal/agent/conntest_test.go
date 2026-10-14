package agent

import (
	"context"
	"net"
	"strings"
	"sync"
	"testing"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/testkit/fakellm"
)

// states lists the steps' states.
func states(steps []engine.ChatTestStep) string {
	var out []string
	for _, s := range steps {
		out = append(out, string(s.Layer)+"="+string(s.State))
	}
	return strings.Join(out, " ")
}

// TestConnectionTestLayers: each layer passes or fails on its own, a failure leaves the later ones
// not reached, progress is reported after every change, and no request carries cluster data.
func TestConnectionTestLayers(t *testing.T) {
	h := newHarness(t, engine.ChatLimits{})
	conn := engine.ChatConnection{Name: "t", Kind: engine.ChatLMStudio, BaseURL: h.llm.Base, Model: "test-model"}

	t.Run("pass", func(t *testing.T) {
		h.llm.Script(fakellm.Reply{ToolCalls: []fakellm.ToolCall{{Name: "get_time", Args: `{}`}}})
		var mu sync.Mutex
		var reports [][]engine.ChatTestStep
		steps := h.a.TestConnection(context.Background(), conn, nil, func(s []engine.ChatTestStep) {
			mu.Lock()
			reports = append(reports, s)
			mu.Unlock()
		})
		if got := states(steps); got != "tcp=pass http=pass auth=pass tools=pass" {
			t.Fatalf("%s (%+v)", got, steps)
		}
		if !strings.Contains(steps[1].Detail, "HTTP 200, plain http") || !strings.Contains(steps[2].Detail, "1 models, no key needed") ||
			!strings.Contains(steps[3].Detail, "test-model called get_time") {
			t.Errorf("details %+v", steps)
		}
		mu.Lock()
		defer mu.Unlock()
		if len(reports) != 9 || states(reports[0]) != "tcp=pending http=pending auth=pending tools=pending" ||
			states(reports[1]) != "tcp=running http=pending auth=pending tools=pending" {
			t.Errorf("%d reports, first %s", len(reports), states(reports[0]))
		}
	})

	t.Run("auth", func(t *testing.T) {
		h.llm.RequireKey("right-key-123")
		defer h.llm.RequireKey("")
		steps := h.a.TestConnection(context.Background(), conn, nil, nil)
		if got := states(steps); got != "tcp=pass http=pass auth=fail tools=notReached" || !strings.Contains(steps[1].Detail, "HTTP 401") ||
			!strings.Contains(steps[2].Detail, "wants an API key") {
			t.Errorf("%s %+v", got, steps)
		}
		wrong := "wrong-key-456"
		steps = h.a.TestConnection(context.Background(), conn, &wrong, nil)
		if got := states(steps); got != "tcp=pass http=pass auth=fail tools=notReached" || !strings.Contains(steps[2].Detail, "refused the API key") ||
			strings.Contains(steps[2].Detail, wrong) {
			t.Errorf("%s %+v", got, steps)
		}
		right := "right-key-123"
		h.llm.Script(fakellm.Reply{ToolCalls: []fakellm.ToolCall{{Name: "get_time", Args: `{}`}}})
		if got := states(h.a.TestConnection(context.Background(), conn, &right, nil)); got != "tcp=pass http=pass auth=pass tools=pass" {
			t.Errorf("with the right key: %s", got)
		}
	})

	t.Run("tools", func(t *testing.T) {
		h.llm.Script(fakellm.Reply{Text: []string{"It is noon."}})
		steps := h.a.TestConnection(context.Background(), conn, nil, nil)
		if got := states(steps); got != "tcp=pass http=pass auth=pass tools=fail" || !strings.Contains(steps[3].Detail, "without a tool call") {
			t.Errorf("%s %+v", got, steps)
		}
	})

	t.Run("models path", func(t *testing.T) {
		wrongBase := conn
		wrongBase.BaseURL = h.llm.Root + "/openai" // no models list there
		steps := h.a.TestConnection(context.Background(), wrongBase, nil, nil)
		if got := states(steps); got != "tcp=pass http=pass auth=fail tools=notReached" || !strings.Contains(steps[2].Detail, "HTTP 404") {
			t.Errorf("%s %+v", got, steps)
		}
	})

	t.Run("tcp", func(t *testing.T) {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		addr := l.Addr().String()
		_ = l.Close()
		closed := conn
		closed.BaseURL = "http://" + addr + "/v1"
		steps := h.a.TestConnection(context.Background(), closed, nil, nil)
		if got := states(steps); got != "tcp=fail http=notReached auth=notReached tools=notReached" || !strings.Contains(steps[0].Detail, addr) {
			t.Errorf("%s %+v", got, steps)
		}
	})

	for _, r := range h.llm.Requests() {
		for _, data := range []string{"shop", "web-0", secretValue, `"context"`} {
			if strings.Contains(string(r.Body), data) {
				t.Errorf("the connection test sent %q", data)
			}
		}
	}
}

// TestModelsAndEndpoints: the models list is read from each kind's path and sorted.
func TestModelsAndEndpoints(t *testing.T) {
	for _, tc := range []struct {
		kind               engine.ChatConnectionKind
		base, chat, models string
	}{
		{engine.ChatLMStudio, "http://h:1234/v1/", "http://h:1234/v1/chat/completions", "http://h:1234/v1/models"},
		{engine.ChatLiteLLM, "https://llm.example", "https://llm.example/chat/completions", "https://llm.example/models"},
		{engine.ChatLiteLLM, "https://llm.example/v1", "https://llm.example/v1/chat/completions", "https://llm.example/v1/models"},
		{engine.ChatOpenWebUI, "https://webui.example", "https://webui.example/api/chat/completions", "https://webui.example/api/models"},
		{engine.ChatOpenWebUI, "https://webui.example/api/", "https://webui.example/api/chat/completions", "https://webui.example/api/models"},
		{engine.ChatOpenAI, "https://api.example/v1", "https://api.example/v1/chat/completions", "https://api.example/v1/models"},
	} {
		ep, err := endpointFor(tc.kind, tc.base)
		if err != nil || ep.chat != tc.chat || ep.models != tc.models {
			t.Errorf("%s %s: %+v %v", tc.kind, tc.base, ep, err)
		}
	}
	for _, bad := range []string{"ftp://h/v1", "http://user:pw@h/v1", "http://h/v1?x=1", "h:1234", "", "http:///v1"} {
		if _, err := endpointFor(engine.ChatOpenAI, bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}

	h := newHarness(t, engine.ChatLimits{})
	h.llm.Models(0, "zeta", "alpha", "alpha")
	models, err := h.a.Models(context.Background(), h.conn.ID)
	if err != nil || strings.Join(models, ",") != "alpha,zeta" {
		t.Errorf("models %v %v", models, err)
	}
	if got, err := parseModels([]byte(`[{"name":"b"},{"id":"a"}]`)); err != nil || strings.Join(got, ",") != "a,b" {
		t.Errorf("a bare array: %v %v", got, err)
	}
}
