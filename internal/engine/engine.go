// Package engine is Cogpit's core: the Session that every origin (GUI, CLI, external agents, the
// built-in agent) dispatches commands to, the command registry, the UI state, table snapshots,
// detail models, events and the confirmation policy (architecture §4).
//
// Layer L3. The engine reaches clusters only through packages cluster, cache and terminal.
// Everything that crosses the engine's boundary is JSON-serializable.
package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"

	"github.com/guided-traffic/cogpit/internal/model"
)

// Origin says who dispatched a command. Confirmation rules depend on it (D-2).
type Origin string

// Origins.
const (
	OriginGUI           Origin = "gui"
	OriginCLI           Origin = "cli"
	OriginExternalAgent Origin = "externalAgent"
	OriginBuiltInAgent  Origin = "builtInAgent"
)

// IsAgent reports whether the origin is an agent.
func (o Origin) IsAgent() bool { return o == OriginExternalAgent || o == OriginBuiltInAgent }

func (o Origin) valid() bool {
	switch o {
	case OriginGUI, OriginCLI, OriginExternalAgent, OriginBuiltInAgent:
		return true
	}
	return false
}

// Invocation is one request to run a command.
type Invocation struct {
	ID     string          `json:"id"`               // "ui.navigate"
	Params json.RawMessage `json:"params,omitempty"` // validated against the command's schema
	Origin Origin          `json:"origin"`
}

// Errors returned by Dispatch. Match them with errors.Is.
var (
	ErrUnknownCommand          = errors.New("unknown command")
	ErrDeclined                = errors.New("the user declined the command")
	ErrConfirmationUnavailable = errors.New("the command needs a confirmation and no GUI is available to confirm it")
	ErrNotConnected            = errors.New("no cluster connected")
)

// ParamsError reports parameters that do not match a command's schema.
type ParamsError struct {
	Command string
	Err     error
}

func (e *ParamsError) Error() string {
	return fmt.Sprintf("command %s: invalid params: %v", e.Command, e.Err)
}
func (e *ParamsError) Unwrap() error { return e.Err }

// DisabledError reports a command that cannot run in the current state, with the reason.
type DisabledError struct {
	Command string
	Reason  string
}

func (e *DisabledError) Error() string {
	return fmt.Sprintf("command %s is disabled: %s", e.Command, e.Reason)
}

// Options configure a Session.
type Options struct {
	// Connector lists and connects kubeconfig contexts (and the demo cluster). Nil: no cluster
	// can be connected.
	Connector Connector
	// Confirmer asks the user to confirm. Nil means headless: commands that need a confirmation
	// are refused with ErrConfirmationUnavailable.
	Confirmer Confirmer
	// Audit receives one JSON line per agent-initiated command. Nil discards.
	Audit io.Writer
	// SnapshotInterval is the minimum time between two table snapshots of a view; zero means
	// 100 ms, at most ten snapshots per second (architecture §5).
	SnapshotInterval time.Duration
	// Now is the clock; nil means time.Now.
	Now func() time.Time
	// Logger receives the engine's logs; nil means slog.Default().
	Logger *slog.Logger
}

// Session is the engine for one desktop app or headless process. It is safe for concurrent use.
type Session struct {
	reg  *Registry
	opts Options
	log  *slog.Logger

	mu          sync.Mutex
	ui          UIState
	confirmer   Confirmer
	agentWrites map[string]bool // contexts where the user allowed agent writes (D-2)

	subMu   sync.Mutex
	subs    map[int]func(Event)
	nextSub int

	auditMu sync.Mutex

	state *state    // views, backend and streams: views.go
	edits editStore // the editor tabs' texts and base versions: editor.go
}

// New returns a Session with an empty registry. Register commands (package commands) before use.
func New(opts Options) *Session {
	if opts.SnapshotInterval <= 0 {
		opts.SnapshotInterval = 100 * time.Millisecond
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	s := &Session{
		reg:         NewRegistry(),
		opts:        opts,
		log:         opts.Logger,
		confirmer:   opts.Confirmer,
		agentWrites: map[string]bool{},
		subs:        map[int]func(Event){},
		ui: UIState{
			Connection: Connection{State: Disconnected},
			Sort:       SortSpec{Column: "name"},
		},
	}
	s.state = newState(s)
	return s
}

// Registry returns the command registry: menus, context menus, the palette and MCP tools come
// from it.
func (s *Session) Registry() *Registry { return s.reg }

// Now returns the session clock's time.
func (s *Session) Now() time.Time { return s.opts.Now() }

// Logger returns the session's logger.
func (s *Session) Logger() *slog.Logger { return s.log }

// SetConfirmer installs the confirmer (the GUI installs itself once its window exists).
func (s *Session) SetConfirmer(c Confirmer) {
	s.mu.Lock()
	s.confirmer = c
	s.mu.Unlock()
}

// SetAgentWrites allows or forbids mutating agent commands without asking, for one context
// (D-2). Destructive commands always ask.
func (s *Session) SetAgentWrites(context string, allowed bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if allowed {
		s.agentWrites[context] = true
	} else {
		delete(s.agentWrites, context)
	}
}

// NeedsConfirmation is the D-2 policy: destructive commands always need the user's confirmation;
// mutating commands need it from agents unless the user allowed agent writes for the context;
// read and UI commands never do.
func NeedsConfirmation(d Danger, o Origin, agentWritesAllowed bool) bool {
	switch d {
	case DangerDestructive:
		return true
	case DangerMutating:
		return o.IsAgent() && !agentWritesAllowed
	}
	return false
}

// Dispatch validates and runs one command: unknown ids, bad parameters and disabled commands are
// refused; commands that need a confirmation wait for the user; agent-initiated commands are
// audited; a panicking command returns an error instead of ending the process.
func (s *Session) Dispatch(ctx context.Context, inv Invocation) (result any, err error) {
	cmd, ok := s.reg.Get(inv.ID)
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownCommand, inv.ID)
	}
	if !inv.Origin.valid() {
		return nil, fmt.Errorf("command %s: invalid origin %q", inv.ID, inv.Origin)
	}
	// auditContext is the context the command was dispatched for (or, once confirmed, the one the
	// user confirmed), not whatever is connected when it finishes.
	var auditContext string
	if inv.Origin.IsAgent() {
		defer func() { s.audit(inv, auditContext, auditDone, err) }()
	}
	p, err := cmd.decode(inv.Params)
	if err != nil {
		return nil, err
	}
	if err := s.safeCall(cmd.ID, func() error { return cmd.enabled(s, p) }); err != nil {
		return nil, &DisabledError{Command: cmd.ID, Reason: err.Error()}
	}

	s.mu.Lock()
	uiContext := s.ui.Context
	allowed := s.agentWrites[uiContext]
	confirmer := s.confirmer
	s.mu.Unlock()
	auditContext = uiContext

	danger := cmd.Danger
	if err := s.safeCall(cmd.ID, func() error { danger = cmd.dangerFor(s, p); return nil }); err != nil {
		return nil, err
	}
	if NeedsConfirmation(danger, inv.Origin, allowed) {
		var targets []model.ObjectRef
		if err := s.safeCall(cmd.ID, func() (err error) { targets, err = cmd.targets(s, p); return err }); err != nil {
			return nil, err
		}
		if confirmer == nil {
			return nil, fmt.Errorf("command %s: %w", cmd.ID, ErrConfirmationUnavailable)
		}
		req := newConfirmRequest(cmd, danger, inv.Origin, uiContext, targets)
		ok, err := confirmer.Confirm(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("command %s: confirmation: %w", cmd.ID, err)
		}
		if !ok {
			return nil, fmt.Errorf("command %s: %w", cmd.ID, ErrDeclined)
		}
		ctx = context.WithValue(ctx, confirmedKey{}, req)
		auditContext = req.Context
	} else if allowed && danger == DangerMutating && inv.Origin.IsAgent() {
		// The user's allowance of agent writes is for one context: bind the command to it the way
		// a confirmation would, so that it refuses to act if the cluster changes before it runs.
		ctx = context.WithValue(ctx, confirmedKey{}, newConfirmRequest(cmd, danger, inv.Origin, uiContext, nil))
	}
	if inv.Origin.IsAgent() && (danger == DangerMutating || danger == DangerDestructive) {
		// A write is logged before it runs too, so that one still running when Cogpit quits is in
		// the log.
		s.audit(inv, auditContext, auditRequested, nil)
	}

	err = s.safeCall(cmd.ID, func() (err error) { result, err = cmd.run(ctx, s, p); return err })
	return result, err
}

// safeCall runs fn and turns a panic into an error, logged with its stack.
func (s *Session) safeCall(id string, fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			s.log.Error("command panicked", "command", id, "panic", fmt.Sprint(r), "stack", string(debug.Stack()))
			err = fmt.Errorf("command %s failed: internal error: %v", id, r)
		}
	}()
	return fn()
}

// UIState returns a copy of the current UI state.
func (s *Session) UIState() UIState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ui.Clone()
}

// UpdateUI changes the UI state with fn, bumps its revision, lets the views follow the change
// (acquire and release cache handles, rebuild snapshots) and emits EventUIState. Commands call it;
// the GUI never does directly. It returns the new state.
func (s *Session) UpdateUI(fn func(*UIState)) UIState {
	s.mu.Lock()
	old := s.ui.Clone()
	fn(&s.ui)
	s.ui.Revision = old.Revision + 1
	now := s.ui.Clone()
	s.mu.Unlock()

	s.state.uiChanged(old, now)
	s.emit(Event{Type: EventUIState, UIState: &now})
	return now
}

// Subscribe registers fn for every engine event. fn runs on engine goroutines and must not block;
// the GUI hands events to its main goroutine with fyne.Do. The returned function unsubscribes.
func (s *Session) Subscribe(fn func(Event)) (cancel func()) {
	s.subMu.Lock()
	id := s.nextSub
	s.nextSub++
	s.subs[id] = fn
	s.subMu.Unlock()
	return func() {
		s.subMu.Lock()
		delete(s.subs, id)
		s.subMu.Unlock()
	}
}

// emit delivers an event to every subscriber. A panicking subscriber is logged and skipped.
func (s *Session) emit(ev Event) {
	s.subMu.Lock()
	fns := make([]func(Event), 0, len(s.subs))
	for _, fn := range s.subs {
		fns = append(fns, fn)
	}
	s.subMu.Unlock()
	for _, fn := range fns {
		func() {
			defer func() {
				if r := recover(); r != nil {
					s.log.Error("event subscriber panicked", "event", ev.Type, "panic", fmt.Sprint(r))
				}
			}()
			fn(ev)
		}()
	}
}

// Emit delivers an event to every subscriber. Commands use it for toasts.
func (s *Session) Emit(ev Event) { s.emit(ev) }

// Close disconnects and stops every view. The session must not be used afterwards.
func (s *Session) Close() { s.state.close() }
