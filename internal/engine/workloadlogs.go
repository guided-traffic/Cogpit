package engine

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/model"
)

// The logs of a controlling resource (workload.logs): one stream that merges the logs of every
// container of every pod the resource controls (for a Deployment, through its ReplicaSets), each
// line starting with "<pod>/<container> ". The pods come from the cache, never from a list
// request: the stream holds cache handles on pods (and replica sets) while it is open and starts
// the log of a container when the cache shows it running, so pods that join while the tab
// follows (a rollout, a scale-up, a restarted container) join the stream.

// Bounds of a controlling resource's logs.
const (
	// workloadLogStreams bounds the containers streamed at once; the stream says when it leaves
	// some out, and starts them when others end.
	workloadLogStreams = 40
	// workloadLogTail is the history the stream starts with, shared by the containers it opens
	// with, but at least workloadLogMinTail lines each. A container that starts later gets
	// workloadLogTail of its own.
	workloadLogTail    = 1000
	workloadLogMinTail = 100
	// workloadLogLineBytes cuts longer lines (the log view keeps 64 KiB of a line too).
	workloadLogLineBytes = 64 << 10
	// workloadLogRescan is the most often the pods are looked at again after a change.
	workloadLogRescan = 250 * time.Millisecond
)

// NoticePrefix starts the lines Cogpit writes into a log stream itself.
const NoticePrefix = "[cogpit]"

// workloadLogs is the merged stream of a controlling resource's pods. Read returns its lines; Close
// ends every container's stream and releases the cache handles.
type workloadLogs struct {
	s        *Session
	ref      model.ObjectRef
	noun     string // "Deployment web"
	opts     LogOptions
	streamer logStreamer
	uid      string // the resource's UID: its pods' (or replica sets') controller
	pods, rs *cache.Handle

	ctx    context.Context
	cancel context.CancelFunc
	pr     *io.PipeReader
	pw     *io.PipeWriter
	wmu    sync.Mutex // one line at a time
	wake   chan struct{}
	stop   chan struct{}
	unsub  []func()
	wg     sync.WaitGroup

	mu sync.Mutex
	// started holds the "<pod uid>/<container>/<instance>" streams started; scan forgets the ones
	// whose pod or container instance the cache no longer shows, which never come back.
	started   map[string]bool
	streaming map[string]bool // the started streams still running
	active    int
	capped    bool
	tail      int64 // the history each of the first containers starts with

	closeOnce sync.Once
}

// logTarget is one container log to stream: one instance of a container of one pod.
type logTarget struct {
	key, pod, container string
	uid                 string // the pod's
	restarts            int64  // the instance: the container's restart count
}

// openWorkloadLogs opens the merged logs of the resource a logs tab names.
func (s *Session) openWorkloadLogs(ctx context.Context, tab DockTab, b *Backend, streamer logStreamer) (io.ReadCloser, error) {
	ref := *tab.Ref
	obj, err := s.Object(ctx, ref)
	if err != nil {
		return nil, err
	}
	podsP, ok := b.Profile(logKindPods)
	if !ok {
		return nil, fmt.Errorf("%w: pods", ErrUnknownKind)
	}
	noun := ref.Kind + " " + ref.Name
	if p, ok := b.Profile(ref.Kind); ok {
		noun = p.Kind + " " + ref.Name
	}
	wl := &workloadLogs{
		s: s, ref: ref, noun: noun, opts: tab.Logs, streamer: streamer, uid: string(obj.GetUID()),
		wake: make(chan struct{}, 1), stop: make(chan struct{}), started: map[string]bool{}, streaming: map[string]bool{},
	}
	wl.pods = b.Acquire(podsP)
	if ref.Kind == logKindDeployments {
		rsP, ok := b.Profile(logKindReplicaSets)
		if !ok {
			wl.pods.Release()
			return nil, fmt.Errorf("%w: replicasets", ErrUnknownKind)
		}
		wl.rs = b.Acquire(rsP)
	}
	if !wl.opts.Previous {
		// Subscribed before the first scan, so that no pod is missed between the two.
		for _, h := range []*cache.Handle{wl.pods, wl.rs} {
			if h != nil {
				wl.unsub = append(wl.unsub, h.Subscribe(func(cache.Event) { signal(wl.wake) }))
			}
		}
	}
	for _, h := range []*cache.Handle{wl.pods, wl.rs} {
		if h == nil {
			continue
		}
		if err := waitSynced(ctx, h, h.Scope().String()); err != nil {
			wl.release()
			return nil, err
		}
	}
	wl.ctx, wl.cancel = context.WithCancel(ctx)
	wl.pr, wl.pw = io.Pipe()

	first := wl.scan()
	tail := int64(workloadLogTail)
	if n := len(first); n > 0 {
		tail = max(int64(workloadLogMinTail), int64(workloadLogTail/min(n, workloadLogStreams)))
	}
	if len(first) == 0 {
		wl.noticeAsync(wl.emptyNotice())
	}
	wl.tail = tail
	wl.start(first, tail)
	if wl.opts.Previous {
		// The previous instances' logs are complete: the stream ends after them.
		go func() {
			wl.wg.Wait()
			_ = wl.pw.Close()
		}()
		return wl, nil
	}
	go func() {
		defer s.recoverTo("workload logs")
		runCoalesced(wl.stop, wl.wake, workloadLogRescan, nil, func() { wl.start(wl.scan(), workloadLogTail) })
	}()
	return wl, nil
}

func (wl *workloadLogs) emptyNotice() string {
	if wl.opts.Previous {
		return fmt.Sprintf("%s no container of %s has a previous instance", NoticePrefix, wl.noun)
	}
	return fmt.Sprintf("%s %s has no running containers yet; their logs show here when they start", NoticePrefix, wl.noun)
}

// scan returns the containers whose logs are not streamed yet: those of the resource's pods that
// run or have run (for Previous: that have a previous instance), in pod and container order. It
// forgets the started streams whose pod or container instance the cache no longer shows (a pod
// replaced in a rollout, a container restarted): their keys never come back, since a pod's UID is
// never reused and a restart count only grows, so the set holds no more than the pods' containers.
func (wl *workloadLogs) scan() []logTarget {
	owners := map[string]bool{wl.uid: true}
	if wl.rs != nil {
		for _, rs := range wl.rs.ListNamespace(wl.ref.Namespace) {
			if controllerUID(rs) == wl.uid {
				owners[string(rs.GetUID())] = true
			}
		}
	}
	wl.mu.Lock()
	defer wl.mu.Unlock()
	var out []logTarget
	shown := map[string]bool{} // the keys of every container instance the cache shows
	for _, pod := range wl.pods.ListNamespace(wl.ref.Namespace) {
		if !owners[controllerUID(pod)] {
			continue
		}
		statuses := map[string]any{}
		for _, st := range model.NestedSlice(pod, "status", "containerStatuses") {
			statuses[model.NestedString(st, "name")] = st
		}
		for _, c := range model.NestedSlice(pod, "spec", "containers") {
			name := model.NestedString(c, "name")
			st, ok := statuses[name]
			if name == "" || !ok {
				continue
			}
			restarts, _ := model.NestedInt64(st, "restartCount")
			instance := fmt.Sprint(restarts)
			if wl.opts.Previous {
				instance = "previous"
			}
			key := string(pod.GetUID()) + "/" + name + "/" + instance
			shown[key] = true
			switch {
			case wl.opts.Previous:
				if model.NestedMap(st, "lastState", "terminated") == nil {
					continue
				}
			case model.NestedMap(st, "state", "running") == nil && model.NestedMap(st, "state", "terminated") == nil:
				continue // waiting: no log yet
			}
			if !wl.started[key] {
				out = append(out, logTarget{key: key, pod: pod.GetName(), container: name, uid: string(pod.GetUID()), restarts: restarts})
			}
		}
	}
	for key := range wl.started {
		if !shown[key] && !wl.streaming[key] {
			delete(wl.started, key)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].pod != out[j].pod {
			return out[i].pod < out[j].pod
		}
		return out[i].container < out[j].container
	})
	return out
}

// controllerUID is the UID of an object's controller, or "".
func controllerUID(obj *unstructured.Unstructured) string {
	for _, o := range obj.GetOwnerReferences() {
		if o.Controller != nil && *o.Controller {
			return string(o.UID)
		}
	}
	return ""
}

// start streams the targets, as many as the bound allows.
func (wl *workloadLogs) start(targets []logTarget, tail int64) {
	wl.mu.Lock()
	defer wl.mu.Unlock()
	for _, t := range targets {
		if wl.ctx.Err() != nil {
			return
		}
		if wl.started[t.key] { // another start took it since the scan (previous instances)
			continue
		}
		if wl.active >= workloadLogStreams {
			if !wl.capped {
				wl.capped = true
				wl.noticeAsync(fmt.Sprintf("%s showing %d containers' logs at once; the others join as these end", NoticePrefix, workloadLogStreams))
			}
			return
		}
		wl.started[t.key], wl.streaming[t.key] = true, true
		wl.active++
		wl.wg.Add(1)
		go wl.stream(t, tail)
	}
}

// Resuming a follow stream the server ended while its container still runs (the API server or
// kubelet closing a long stream, a broken connection).
var (
	// workloadLogResumes bounds the resumptions in a row that bring no new line; then the stream
	// says so and stops following that container.
	workloadLogResumes = 5
	// workloadLogBackoff is the wait before a resumption, doubled for each further one in a row,
	// at most workloadLogMaxBackoff.
	workloadLogBackoff    = 500 * time.Millisecond
	workloadLogMaxBackoff = 8 * time.Second
)

// workloadLogSteady is how long a stream must have stayed open for its end to count as a new
// drop rather than another of the same: the count of resumptions in a row starts again.
const workloadLogSteady = 30 * time.Second

// logCursor is where a container's log stands: the newest timestamp written and how many lines
// carried it, so that a resumption drops what it has already written.
type logCursor struct {
	last  time.Time
	atIt  int
	lines int // lines written, with or without a timestamp
}

// stream copies one container's log into the merged stream, line by line, each with its prefix.
// A follow stream the server ends while the cache still shows the container instance running is
// opened again from the newest line's timestamp (sinceTime), after a backoff, at most
// workloadLogResumes times in a row without a new line.
func (wl *workloadLogs) stream(t logTarget, tail int64) {
	defer wl.wg.Done()
	defer func() {
		wl.mu.Lock()
		wl.active--
		delete(wl.streaming, t.key)
		wl.mu.Unlock()
		if wl.opts.Previous {
			// No rescan runs for previous instances: start the ones left out here, before this
			// stream's Done, so that the merged stream does not end in between.
			wl.start(wl.scan(), wl.tail)
			return
		}
		signal(wl.wake) // a container left out may start now
	}()
	defer wl.s.recoverTo("workload logs")
	prefix := t.pod + "/" + t.container
	follow := !wl.opts.Previous
	var cur logCursor
	drops := 0 // resumptions in a row without a new line
	for attempt := 0; ; attempt++ {
		opts := cluster.LogOptions{
			Namespace: wl.ref.Namespace, Pod: t.pod, Container: t.container,
			// A follow stream asks for timestamps whether the tab shows them or not: a resumption
			// starts from them. The tab gets them stripped.
			Follow: follow, Previous: wl.opts.Previous, Timestamps: wl.opts.Timestamps || follow, TailLines: tail,
		}
		if !cur.last.IsZero() {
			opts.TailLines, opts.SinceTime = 0, cur.last.Truncate(time.Second)
		}
		opened := time.Now()
		rc, err := wl.streamer.StreamLogs(wl.ctx, opts)
		fresh := 0
		if err != nil {
			if wl.ctx.Err() != nil {
				return
			}
			if attempt == 0 {
				wl.write(NoticePrefix, []byte(prefix+": "+oneLine(err.Error())))
				return
			}
		} else {
			var open bool
			fresh, open = wl.copyLines(rc, prefix, &cur, follow)
			_ = rc.Close()
			if !open {
				return // the tab closed
			}
		}
		// Ended: by the container (it exited or restarted; the scan starts its next instance), by
		// the tab, or by the server while the container runs on.
		if !follow || wl.ctx.Err() != nil || !wl.stillRunning(t) {
			return
		}
		if cur.last.IsZero() && cur.lines > 0 {
			return // lines without timestamps: a resumption could not tell which it has
		}
		if fresh > 0 || time.Since(opened) >= workloadLogSteady {
			drops = 0
		}
		drops++
		if drops > workloadLogResumes {
			wl.write(NoticePrefix, []byte(fmt.Sprintf("%s: the log stream ended %d times in a row without new lines; Cogpit stopped following it", prefix, drops-1)))
			return
		}
		wait := time.NewTimer(min(workloadLogBackoff<<(drops-1), workloadLogMaxBackoff))
		select {
		case <-wl.ctx.Done():
			wait.Stop()
			return
		case <-wait.C:
		}
		// Asked again after the wait: a stream also ends when its container restarts, and the
		// cache may show the restart late. A resumption names the container, not the instance,
		// and would follow the new instance, whose own stream the scan starts.
		if !wl.stillRunning(t) {
			return
		}
	}
}

// copyLines copies one opened stream's lines into the merged stream and returns how many it
// wrote; open is false once the merged stream is closed. With stamped (a follow stream, which asks
// for timestamps), each line's leading timestamp moves cur, lines at or before cur's are dropped
// (a resumption sends the second it starts in again), and the timestamp is cut off unless the tab
// shows timestamps.
func (wl *workloadLogs) copyLines(rc io.Reader, prefix string, cur *logCursor, stamped bool) (fresh int, open bool) {
	// What the stream had when this one opened: lines before it, and the first atIt lines at it,
	// are written already.
	had, skip := cur.last, cur.atIt
	br := bufio.NewReaderSize(rc, workloadLogLineBytes)
	for {
		line, err := br.ReadSlice('\n')
		full := errors.Is(err, bufio.ErrBufferFull)
		out, keep := line, len(line) > 0
		if keep && stamped {
			if ts, rest, ok := splitLogTimestamp(line); ok {
				switch {
				case ts.Before(had):
					keep = false
				case ts.Equal(had) && skip > 0:
					skip--
					keep = false
				case ts.After(cur.last):
					cur.last, cur.atIt = ts, 1
				default:
					cur.atIt++
				}
				if !wl.opts.Timestamps {
					out = rest
				}
			}
		}
		if keep {
			if !wl.write(prefix, out) {
				return fresh, false
			}
			fresh++
			cur.lines++
		}
		for full { // the rest of a line longer than the bound is dropped
			_, err = br.ReadSlice('\n')
			full = errors.Is(err, bufio.ErrBufferFull)
		}
		if err != nil {
			return fresh, true
		}
	}
}

// splitLogTimestamp splits the RFC 3339 timestamp the API server puts before a line when asked
// for timestamps from the line.
func splitLogTimestamp(line []byte) (time.Time, []byte, bool) {
	head, rest, found := bytes.Cut(line, []byte{' '})
	if !found {
		head, rest = bytes.TrimRight(line, "\r\n"), nil
	}
	ts, err := time.Parse(time.RFC3339Nano, string(head))
	if err != nil {
		return time.Time{}, line, false
	}
	return ts, rest, true
}

// stillRunning reports whether the cache shows the target's container instance running: the same
// pod (by UID), the same restart count.
func (wl *workloadLogs) stillRunning(t logTarget) bool {
	pod, ok := wl.pods.Get(wl.ref.Namespace, t.pod)
	if !ok || string(pod.GetUID()) != t.uid {
		return false
	}
	for _, st := range model.NestedSlice(pod, "status", "containerStatuses") {
		if model.NestedString(st, "name") == t.container {
			restarts, _ := model.NestedInt64(st, "restartCount")
			return restarts == t.restarts && model.NestedMap(st, "state", "running") != nil
		}
	}
	return false
}

// write adds one line to the merged stream: the prefix, a space, the line, a line feed. It
// reports false once the stream is closed.
func (wl *workloadLogs) write(prefix string, line []byte) bool {
	for len(line) > 0 && (line[len(line)-1] == '\n' || line[len(line)-1] == '\r') {
		line = line[:len(line)-1]
	}
	buf := make([]byte, 0, len(prefix)+len(line)+2)
	buf = append(buf, prefix...)
	buf = append(buf, ' ')
	buf = append(buf, line...)
	buf = append(buf, '\n')
	wl.wmu.Lock()
	defer wl.wmu.Unlock()
	_, err := wl.pw.Write(buf)
	return err == nil
}

// noticeAsync writes one of Cogpit's own lines without waiting for the reader.
func (wl *workloadLogs) noticeAsync(text string) {
	wl.wg.Add(1)
	go func() {
		defer wl.wg.Done()
		prefix, rest, _ := strings.Cut(text, " ")
		wl.write(prefix, []byte(rest))
	}()
}

// oneLine makes an error from the cluster one line without control characters.
func oneLine(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return ' '
		}
		return r
	}, s)
}

// Read returns the merged lines.
func (wl *workloadLogs) Read(p []byte) (int, error) { return wl.pr.Read(p) }

// Close ends every container's stream and releases the cache handles. It is idempotent and does
// not wait for the streams' goroutines.
func (wl *workloadLogs) Close() error {
	wl.closeOnce.Do(func() {
		wl.cancel()
		close(wl.stop)
		_ = wl.pr.Close() // unblocks writers
		wl.release()
	})
	return nil
}

func (wl *workloadLogs) release() {
	for _, u := range wl.unsub {
		u()
	}
	wl.unsub = nil
	wl.pods.Release()
	if wl.rs != nil {
		wl.rs.Release()
	}
}
