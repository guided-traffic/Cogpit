package cluster

import (
	"context"
	"io"
)

// LogOptions select a container log stream.
type LogOptions struct {
	Namespace  string
	Pod        string
	Container  string // empty: the pod's only or default container
	Follow     bool
	Previous   bool
	Timestamps bool
	TailLines  int64 // 0: the API server's default (all lines)
}

// StreamLogs opens a container log stream. The caller closes it.
func (c *Conn) StreamLogs(ctx context.Context, opts LogOptions) (io.ReadCloser, error) {
	_, _ = ctx, opts
	return nil, ErrNotImplemented
}
