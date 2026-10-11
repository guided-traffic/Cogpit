package engine

import (
	"bufio"
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

	mu      sync.Mutex
	started map[string]bool // "<pod uid>/<container>/<instance>" streams started
	active  int
	capped  bool

	closeOnce sync.Once
}

// logTarget is one container log to stream.
type logTarget struct {
	key, pod, container string
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
		wake: make(chan struct{}, 1), stop: make(chan struct{}), started: map[string]bool{},
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
// run or have run (for Previous: that have a previous instance), in pod and container order.
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
			switch {
			case wl.opts.Previous:
				if model.NestedMap(st, "lastState", "terminated") == nil {
					continue
				}
				instance = "previous"
			case model.NestedMap(st, "state", "running") == nil && model.NestedMap(st, "state", "terminated") == nil:
				continue // waiting: no log yet
			}
			key := string(pod.GetUID()) + "/" + name + "/" + instance
			if !wl.started[key] {
				out = append(out, logTarget{key: key, pod: pod.GetName(), container: name})
			}
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
		if wl.active >= workloadLogStreams {
			if !wl.capped {
				wl.capped = true
				wl.noticeAsync(fmt.Sprintf("%s showing %d containers' logs at once; the others join as these end", NoticePrefix, workloadLogStreams))
			}
			return
		}
		wl.started[t.key] = true
		wl.active++
		wl.wg.Add(1)
		go wl.stream(t, tail)
	}
}

// stream copies one container's log into the merged stream, line by line, each with its prefix.
func (wl *workloadLogs) stream(t logTarget, tail int64) {
	defer wl.wg.Done()
	defer func() {
		wl.mu.Lock()
		wl.active--
		wl.mu.Unlock()
		signal(wl.wake) // a container left out may start now
	}()
	defer wl.s.recoverTo("workload logs")
	prefix := t.pod + "/" + t.container
	rc, err := wl.streamer.StreamLogs(wl.ctx, cluster.LogOptions{
		Namespace: wl.ref.Namespace, Pod: t.pod, Container: t.container,
		Follow: !wl.opts.Previous, Previous: wl.opts.Previous, Timestamps: wl.opts.Timestamps, TailLines: tail,
	})
	if err != nil {
		if wl.ctx.Err() == nil {
			wl.write(NoticePrefix, []byte(prefix+": "+oneLine(err.Error())))
		}
		return
	}
	defer rc.Close()
	br := bufio.NewReaderSize(rc, workloadLogLineBytes)
	for {
		line, err := br.ReadSlice('\n')
		full := errors.Is(err, bufio.ErrBufferFull)
		if len(line) > 0 && !wl.write(prefix, line) {
			return
		}
		for full { // the rest of a line longer than the bound is dropped
			_, err = br.ReadSlice('\n')
			full = errors.Is(err, bufio.ErrBufferFull)
		}
		if err != nil {
			return
		}
	}
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
