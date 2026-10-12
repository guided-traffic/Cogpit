package cluster

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	// SinceTime starts the log at the lines written at or after it; zero: from the start (or the
	// tail). The API carries it in whole seconds (RFC 3339): a caller that resumes a stream gets
	// the lines of that second again and drops the ones it has.
	SinceTime time.Time
}

// StreamLogs opens a container log stream through the pod's log subresource. The caller closes
// it; cancelling ctx ends it too. The stream's content is the container's output: untrusted.
func (c *Conn) StreamLogs(ctx context.Context, opts LogOptions) (io.ReadCloser, error) {
	if opts.Namespace == "" || opts.Pod == "" {
		return nil, errors.New("logs: namespace and pod are required")
	}
	if opts.TailLines < 0 {
		return nil, fmt.Errorf("logs: tail lines must not be negative, got %d", opts.TailLines)
	}
	if c == nil || c.kube == nil {
		return nil, errors.New("logs: not connected")
	}
	lo := &corev1.PodLogOptions{
		Container:  opts.Container,
		Follow:     opts.Follow,
		Previous:   opts.Previous,
		Timestamps: opts.Timestamps,
	}
	if opts.TailLines > 0 {
		tail := opts.TailLines
		lo.TailLines = &tail
	}
	if !opts.SinceTime.IsZero() {
		lo.SinceTime = &metav1.Time{Time: opts.SinceTime}
	}
	stream, err := c.kube.CoreV1().Pods(opts.Namespace).GetLogs(opts.Pod, lo).Stream(ctx)
	if err != nil {
		return nil, fmt.Errorf("logs of %s/%s: %w", opts.Namespace, opts.Pod, err)
	}
	return stream, nil
}
