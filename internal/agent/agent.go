package agent

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// Files the agent keeps in Cogpit's configuration directory, owner-only (0600 in 0700).
const (
	// ConnectionsFile holds the LLM connections: name, kind, base URL, model, limits, the private
	// mark, whether a key is in the keychain. Never a key.
	ConnectionsFile = "llm-connections.json"
	// ConversationsFile holds the last MaxConversations conversations per context, scrubbed; never
	// a private one (H-96).
	ConversationsFile = "chat-conversations.json"
)

// MaxConversations is how many conversations of a context are kept (D-10).
const MaxConversations = 20

// Errors of the chat. They carry no key and no cluster data.
var (
	ErrNoConnection     = errors.New("no LLM connection: add one in the chat's settings")
	ErrUnknownConn      = errors.New("no such LLM connection")
	ErrUnknownConv      = errors.New("no such conversation")
	ErrTurnRunning      = errors.New("a turn of this conversation is still running: stop it or wait for it")
	ErrOtherContext     = errors.New("this conversation belongs to another cluster than the connected one: connect its context to go on, or start a new conversation")
	ErrNotConnected     = errors.New("no cluster is connected: connect the conversation's context to go on")
	ErrEmptyMessage     = errors.New("the message is empty")
	ErrConnectionFields = errors.New("a connection needs a name, a kind and a base URL")
	ErrNoModel          = errors.New("the connection has no model: choose one in the chat's settings")
)

// Options configure the agent.
type Options struct {
	// Session is the engine the chat's tool calls are dispatched to.
	Session *engine.Session
	// Dir is Cogpit's configuration directory (app.Dirs.Config), where ConnectionsFile and
	// ConversationsFile are kept; empty keeps both in memory only.
	Dir string
	// Keys keeps the API keys; nil means NewSystemKeyStore().
	Keys KeyStore
	// HTTPClient sends the requests; nil means a client without redirects (newHTTPClient).
	HTTPClient *http.Client
	// Logger receives the agent's logs (never a key, a message or a tool result); nil means the
	// session's.
	Logger *slog.Logger
	// Now is the clock; nil means the session's.
	Now func() time.Time
	// NotifyEvery is the least time between two EventChat of a conversation while an answer
	// streams; zero means 75 ms. A state change notifies at once.
	NotifyEvery time.Duration
}

// Agent implements engine.ChatAgent.
type Agent struct {
	s    *engine.Session
	opts Options
	log  *slog.Logger
	keys KeyStore
	hc   *http.Client

	toolsOnce sync.Once
	tools     *toolset
	toolsErr  error

	mu        sync.Mutex
	conns     []engine.ChatConnection
	convs     map[string]*conversation
	noOptions map[string]bool // connections whose endpoint refused stream_options

	saveMu sync.Mutex // serialises the files' writes
}

var _ engine.ChatAgent = (*Agent)(nil)

// New returns the agent with the connections and conversations kept in opts.Dir. A file that
// cannot be read is logged and left alone (the agent starts empty and does not overwrite it until
// something changes).
func New(opts Options) *Agent {
	if opts.Keys == nil {
		opts.Keys = NewSystemKeyStore()
	}
	if opts.HTTPClient == nil {
		opts.HTTPClient = newHTTPClient()
	}
	if opts.Logger == nil {
		opts.Logger = opts.Session.Logger()
	}
	if opts.Now == nil {
		opts.Now = opts.Session.Now
	}
	if opts.NotifyEvery <= 0 {
		opts.NotifyEvery = 75 * time.Millisecond
	}
	a := &Agent{s: opts.Session, opts: opts, log: opts.Logger, keys: opts.Keys, hc: opts.HTTPClient,
		convs: map[string]*conversation{}, noOptions: map[string]bool{}}
	a.loadConnections()
	a.loadConversations()
	return a
}

// toolset returns the tools, built on first use (the registry is complete by then).
func (a *Agent) toolset() (*toolset, error) {
	a.toolsOnce.Do(func() { a.tools, a.toolsErr = newToolset(a.s.Registry()) })
	return a.tools, a.toolsErr
}

// newID returns a random id.
func newID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// --- connections ---

// Connections implements engine.ChatAgent: the default connection first, then by name.
func (a *Agent) Connections() ([]engine.ChatConnection, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.connectionsLocked(), nil
}

func (a *Agent) connectionsLocked() []engine.ChatConnection {
	out := slices.Clone(a.conns)
	slices.SortStableFunc(out, func(x, y engine.ChatConnection) int {
		if x.Default != y.Default {
			if x.Default {
				return -1
			}
			return 1
		}
		return strings.Compare(strings.ToLower(x.Name), strings.ToLower(y.Name))
	})
	return out
}

// validKind reports whether k is one of the kinds.
func validKind(k engine.ChatConnectionKind) bool {
	switch k {
	case engine.ChatLMStudio, engine.ChatLiteLLM, engine.ChatOpenWebUI, engine.ChatOpenAI:
		return true
	}
	return false
}

// normalize checks a connection's fields and tidies them.
func normalize(c engine.ChatConnection) (engine.ChatConnection, error) {
	c.Name, c.Model = strings.TrimSpace(c.Name), strings.TrimSpace(c.Model)
	// The model may be chosen later: the settings form lists the endpoint's models for a saved
	// connection (Models takes its id); a turn refuses to start without one.
	if c.Name == "" || !validKind(c.Kind) || strings.TrimSpace(c.BaseURL) == "" {
		return c, ErrConnectionFields
	}
	u, err := parseBaseURL(c.BaseURL)
	if err != nil {
		return c, err
	}
	c.BaseURL = u.String()
	if c.Limits.RepeatStop < 0 || c.Limits.MaxToolCalls < 0 || c.Limits.MaxTokens < 0 || c.Limits.MaxDuration < 0 {
		return c, errors.New("a connection's limits cannot be negative")
	}
	return c, nil
}

// SaveConnection implements engine.ChatAgent.
func (a *Agent) SaveConnection(c engine.ChatConnection, apiKey *string) (engine.ChatConnection, error) {
	c, err := normalize(c)
	if err != nil {
		return engine.ChatConnection{}, err
	}
	wantDefault := c.Default
	a.mu.Lock()
	i := -1
	if c.ID != "" {
		i = slices.IndexFunc(a.conns, func(x engine.ChatConnection) bool { return x.ID == c.ID })
		if i < 0 {
			a.mu.Unlock()
			return engine.ChatConnection{}, ErrUnknownConn
		}
		c.HasKey = a.conns[i].HasKey
		c.Default = a.conns[i].Default
	} else {
		c.ID, c.HasKey = newID(), false
		c.Default = len(a.conns) == 0
	}
	a.mu.Unlock()

	// The key goes to the keychain first: a connection never claims a key it does not have.
	if apiKey != nil {
		if *apiKey == "" {
			if err := a.keys.Delete(c.ID); err != nil {
				return engine.ChatConnection{}, err
			}
			c.HasKey = false
		} else {
			if err := a.keys.Set(c.ID, *apiKey); err != nil {
				return engine.ChatConnection{}, err
			}
			c.HasKey = true
		}
	}

	a.mu.Lock()
	if i >= 0 {
		// The list may have changed while the keychain was asked: find it again.
		if i = slices.IndexFunc(a.conns, func(x engine.ChatConnection) bool { return x.ID == c.ID }); i < 0 {
			a.mu.Unlock()
			_ = a.keys.Delete(c.ID)
			return engine.ChatConnection{}, ErrUnknownConn
		}
		a.conns[i] = c
	} else {
		a.conns = append(a.conns, c)
	}
	if wantDefault {
		for i := range a.conns {
			a.conns[i].Default = a.conns[i].ID == c.ID
		}
		c.Default = true
	}
	delete(a.noOptions, c.ID) // the endpoint may have changed
	a.mu.Unlock()
	a.saveConnections()
	a.s.NotifyChat("", 0)
	return c, nil
}

// DeleteConnection implements engine.ChatAgent.
func (a *Agent) DeleteConnection(id string) error {
	a.mu.Lock()
	i := slices.IndexFunc(a.conns, func(x engine.ChatConnection) bool { return x.ID == id })
	if i < 0 {
		a.mu.Unlock()
		return ErrUnknownConn
	}
	wasDefault := a.conns[i].Default
	a.conns = slices.Delete(a.conns, i, i+1)
	if wasDefault && len(a.conns) > 0 {
		a.conns[0].Default = true
	}
	a.mu.Unlock()
	err := a.keys.Delete(id)
	a.saveConnections()
	a.s.NotifyChat("", 0)
	return err
}

// SetDefaultConnection implements engine.ChatAgent.
func (a *Agent) SetDefaultConnection(id string) error {
	a.mu.Lock()
	if !slices.ContainsFunc(a.conns, func(x engine.ChatConnection) bool { return x.ID == id }) {
		a.mu.Unlock()
		return ErrUnknownConn
	}
	for i := range a.conns {
		a.conns[i].Default = a.conns[i].ID == id
	}
	a.mu.Unlock()
	a.saveConnections()
	a.s.NotifyChat("", 0)
	return nil
}

// connection returns a connection by id; "" or an id that is gone gives the default one.
func (a *Agent) connectionLocked(id string) (engine.ChatConnection, bool) {
	if i := slices.IndexFunc(a.conns, func(x engine.ChatConnection) bool { return x.ID == id }); i >= 0 && id != "" {
		return a.conns[i], true
	}
	if i := slices.IndexFunc(a.conns, func(x engine.ChatConnection) bool { return x.Default }); i >= 0 {
		return a.conns[i], true
	}
	if len(a.conns) > 0 {
		return a.conns[0], true
	}
	return engine.ChatConnection{}, false
}

// keyFor returns the stored key of a connection that has one.
func (a *Agent) keyFor(c engine.ChatConnection) (string, error) {
	if !c.HasKey {
		return "", nil
	}
	k, err := a.keys.Get(c.ID)
	if errors.Is(err, ErrNoKey) {
		return "", errors.New("the connection's API key is no longer in the keychain: enter it again in the chat's settings")
	}
	return k, err
}

// clientFor returns a client for a connection with key.
func (a *Agent) clientFor(c engine.ChatConnection, key string) (*client, error) {
	ep, err := endpointFor(c.Kind, c.BaseURL)
	if err != nil {
		return nil, err
	}
	return &client{hc: a.hc, ep: ep, key: key}, nil
}

// Models implements engine.ChatAgent.
func (a *Agent) Models(ctx context.Context, connectionID string) ([]string, error) {
	a.mu.Lock()
	i := slices.IndexFunc(a.conns, func(x engine.ChatConnection) bool { return x.ID == connectionID })
	var c engine.ChatConnection
	if i >= 0 {
		c = a.conns[i]
	}
	a.mu.Unlock()
	if i < 0 {
		return nil, ErrUnknownConn
	}
	key, err := a.keyFor(c)
	if err != nil {
		return nil, err
	}
	cl, err := a.clientFor(c, key)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	models, err := cl.listModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("models of %s: %w", c.Name, err)
	}
	return models, nil
}
