package engine

import (
	"context"
	"errors"
	"io"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/terminal"
)

// errNotImplemented marks a Phase 0 stub.
var errNotImplemented = errors.New("not implemented")

// state holds the connected backend, the open views with their cache handles and coalesced
// snapshots, and the drawer. Phase 1 implements it.
type state struct {
	s *Session
}

func newState(s *Session) *state { return &state{s: s} }

// uiChanged follows a UI state change: acquire the cache handle of a newly opened view, release
// the old one, rebuild the snapshot when namespaces, filter or sort changed, follow the drawer.
func (st *state) uiChanged(old, now UIState) { _, _ = old, now }

func (st *state) close() {}

// Backend returns the connected backend, or nil.
func (s *Session) Backend() *Backend { return nil }

// Contexts lists the contexts the connector offers.
func (s *Session) Contexts() ([]ContextInfo, error) { return nil, errNotImplemented }

// Connect connects a context through the connector, replacing the current backend, and opens the
// default view. The UI state follows the connection's lifecycle.
func (s *Session) Connect(ctx context.Context, name string) error {
	_, _ = ctx, name
	return errNotImplemented
}

// Table returns the latest snapshot of a view, or nil if the view is not open.
func (s *Session) Table(view string) *TableSnapshot {
	_ = view
	return nil
}

// Detail builds the drawer model for ref from the cache.
func (s *Session) Detail(ref model.ObjectRef) (*DetailModel, error) {
	_ = ref
	return nil, errNotImplemented
}

// Objects returns the cached objects of a kind, in the given namespaces (empty: all), acquiring a
// cache handle for the call and waiting for the initial list (bounded by ctx). The handle is
// released afterwards and the informer stays warm. Objects are shared and must not be modified.
func (s *Session) Objects(ctx context.Context, kind string, namespaces []string) ([]*unstructured.Unstructured, error) {
	_, _, _ = ctx, kind, namespaces
	return nil, errNotImplemented
}

// Object returns one cached object.
func (s *Session) Object(ctx context.Context, ref model.ObjectRef) (*unstructured.Unstructured, error) {
	_, _ = ctx, ref
	return nil, errNotImplemented
}

// Diagnostics returns the request counters and cache stats of the connected backend.
func (s *Session) Diagnostics() Diagnostics { return Diagnostics{} }

// OpenLogs opens the log stream of a logs dock tab. The caller closes it.
func (s *Session) OpenLogs(ctx context.Context, tabID string) (io.ReadCloser, error) {
	_, _ = ctx, tabID
	return nil, errNotImplemented
}

// OpenTerminal starts the session of a terminal or shell dock tab. The caller closes it.
func (s *Session) OpenTerminal(ctx context.Context, tabID string) (terminal.Session, error) {
	_, _ = ctx, tabID
	return nil, errNotImplemented
}
