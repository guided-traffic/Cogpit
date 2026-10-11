package terminal

import (
	"errors"
	"io"
	"sync"
)

// maxQueuedInput bounds the keystrokes and pasted text waiting for a session to take them.
const maxQueuedInput = 1 << 20

// errInputFull is returned by Write when the session has not taken maxQueuedInput bytes yet.
var errInputFull = errors.New("terminal: input queue full; the session is not reading")

// inputQueue decouples the writer of keystrokes (the UI's main goroutine) from the session's
// sink, which can stall: a busy program that does not read its terminal, a full PTY buffer, an
// exec stream that is still connecting. Write never blocks; Read blocks until input arrives.
type inputQueue struct {
	mu     sync.Mutex
	cond   *sync.Cond
	buf    []byte
	closed bool
}

func newInputQueue() *inputQueue {
	q := &inputQueue{}
	q.cond = sync.NewCond(&q.mu)
	return q
}

// Write queues p as a whole, or fails without queueing any of it.
func (q *inputQueue) Write(p []byte) (int, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return 0, io.ErrClosedPipe
	}
	if len(q.buf)+len(p) > maxQueuedInput {
		return 0, errInputFull
	}
	q.buf = append(q.buf, p...)
	q.cond.Signal()
	return len(p), nil
}

// Read takes queued input; it blocks while the queue is empty and returns io.EOF once the queue
// is closed. Input still queued at Close is dropped: the session it was meant for has ended.
func (q *inputQueue) Read(p []byte) (int, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for len(q.buf) == 0 && !q.closed {
		q.cond.Wait()
	}
	if q.closed {
		return 0, io.EOF
	}
	n := copy(p, q.buf)
	rest := copy(q.buf, q.buf[n:])
	q.buf = q.buf[:rest]
	return n, nil
}

// Close ends the queue; pending and future Reads return io.EOF, Writes fail.
func (q *inputQueue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.closed = true
	q.buf = nil
	q.cond.Broadcast()
	return nil
}
