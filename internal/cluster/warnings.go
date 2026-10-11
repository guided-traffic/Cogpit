package cluster

import (
	"context"
	"log/slog"
	"sync"
)

const (
	// maxWarnings bounds the distinct warnings a connection remembers; later new ones are counted,
	// not logged.
	maxWarnings = 64
	// maxWarningLen cuts a warning's text in the log; an admission webhook writes what it wants.
	maxWarningLen = 512
)

// warningLog handles the API server's warnings (the Warning header with code 299: deprecated
// kinds and versions, admission webhooks' warnings) for one connection. client-go's default logs
// every response that carries one, and the API server attaches a deprecation warning to every
// request for a deprecated kind: v1 Endpoints since Kubernetes 1.33 warn on every list, every
// watch the reflector reconnects, and every get, as long as a view, the drawer or an agent reads
// them. warningLog logs each distinct warning once per connection instead, and keeps them for
// Conn.Warnings.
type warningLog struct {
	context string
	log     *slog.Logger // nil: slog.Default()

	mu       sync.Mutex
	seen     map[string]bool
	order    []string
	received int
}

func newWarningLog(context string) *warningLog {
	return &warningLog{context: context, seen: map[string]bool{}}
}

// HandleWarningHeaderWithContext implements rest.WarningHandlerWithContext.
func (w *warningLog) HandleWarningHeaderWithContext(_ context.Context, code int, _ string, text string) {
	if code != 299 || text == "" {
		return
	}
	w.mu.Lock()
	w.received++
	first := !w.seen[text] && len(w.order) < maxWarnings
	if first {
		w.seen[text] = true
		w.order = append(w.order, text)
	}
	w.mu.Unlock()
	if !first {
		return
	}
	if len(text) > maxWarningLen {
		text = text[:maxWarningLen] + "…"
	}
	log := w.log
	if log == nil {
		log = slog.Default()
	}
	log.Warn("cluster: the API server warns (logged once per connection)", "context", w.context, "warning", text)
}

// warnings returns the distinct warnings in the order they first came, and how many arrived.
func (w *warningLog) warnings() ([]string, int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]string(nil), w.order...), w.received
}

// Warnings returns the distinct warnings the API server sent on this connection, in the order
// they first came (at most 64), and how many warnings arrived in all, repeats included.
func (c *Conn) Warnings() (distinct []string, received int) {
	if c.warnings == nil {
		return nil, 0
	}
	return c.warnings.warnings()
}
