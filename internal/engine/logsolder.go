package engine

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"sync"
	"time"

	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/model"
)

// Older lines of a logs tab (ui-parity › Logs, "Load older lines when scrolling up" and "Jump to
// a time and date", gap G8). A tab starts with the last 1,000 lines (shared by a controlling
// resource's containers) and holds at most what its view keeps (10,000 lines). The API has no
// "before" cursor: older lines come from asking again for more of the tail than the tab holds
// (tailLines), or from a time on (sinceTime with tailLines as the bound, for a jump), and keeping
// what is older than the tab's oldest line of each container. Only the user's or an agent's action
// asks (logs.loadOlder, logs.jumpTo, which the GUI carries out with OlderLogs), never a timer;
// every request is a read of pods/log and bounded by the lines the tab holds plus what it has
// room for.

// Bounds of older lines.
const (
	// olderLogStep is how many older lines one request for them adds, shared by the containers
	// it asks, at least workloadLogMinTail each.
	olderLogStep = 1000
	// olderLogRequests bounds the requests running at once.
	olderLogRequests = 8
	// olderLogBytes bounds the bytes of the lines a request keeps, shared by the containers it
	// asks: what a tab's view keeps at most (32 MiB), so that long lines cannot make it grow beyond.
	olderLogBytes = 32 << 20
)

// HeldLog is what a logs tab holds of one container's log.
type HeldLog struct {
	// Source is "<pod>/<container>" in a controlling resource's tab, the prefix of its lines; ""
	// in a pod's tab.
	Source string `json:"source"`
	// Lines is how many lines the tab holds of it.
	Lines int `json:"lines"`
	// Oldest is the oldest held line's timestamp; zero when the tab holds none.
	Oldest time.Time `json:"oldest"`
	// Whole: the tab holds the container's whole log; it is not asked again.
	Whole bool `json:"whole,omitempty"`
}

// OlderRequest is what a logs tab holds and wants.
type OlderRequest struct {
	Held []HeldLog `json:"held"`
	// Since asks for the lines written at or after it (a jump to that time); zero asks for the
	// next olderLogStep lines before the oldest held ones.
	Since time.Time `json:"since"`
	// Room is how many lines the tab can still take.
	Room int `json:"room"`
}

// OlderLines are a logs tab's older lines.
type OlderLines struct {
	// Lines are the older lines, oldest first, as the tab's stream writes them: the
	// "<pod>/<container> " prefix in a controlling resource's tab, then the server's timestamp and
	// the text. At most the request's Room; lines at the oldest held line's timestamp are among
	// them, for the tab to drop those it holds.
	Lines []string `json:"lines"`
	// Whole lists the sources whose whole log the tab holds with these lines.
	Whole []string `json:"whole,omitempty"`
	// Complete: the tab holds the whole log of every container it shows (with Since: every line
	// since then).
	Complete bool `json:"complete"`
	// Notes are Cogpit's own lines about containers whose log it could not read, one each.
	Notes []string `json:"notes,omitempty"`
}

// olderSource is one container a request for older lines asks.
type olderSource struct {
	prefix, pod, container string
}

// OlderLogs reads the older lines of an open logs tab from the API server: for each container the
// tab shows that it does not hold whole, the tail it holds plus a step (or, with Since, the lines
// since then up to the room it has), keeping what is older than its oldest held line.
func (s *Session) OlderLogs(ctx context.Context, tabID string, req OlderRequest) (OlderLines, error) {
	tab, b, err := s.streamTab(tabID, DockLogs)
	if err != nil {
		return OlderLines{}, err
	}
	return s.olderLogs(ctx, tab, b.Conn, req)
}

func (s *Session) olderLogs(ctx context.Context, tab DockTab, streamer logStreamer, req OlderRequest) (OlderLines, error) {
	out := OlderLines{Lines: []string{}}
	if req.Room <= 0 {
		return out, nil
	}
	sources, err := s.olderSources(ctx, tab)
	if err != nil {
		return out, err
	}
	held := map[string]HeldLog{}
	for _, h := range req.Held {
		held[h.Source] = h
	}
	var ask []olderSource
	for _, src := range sources {
		h := held[src.prefix]
		covered := !req.Since.IsZero() && !h.Oldest.IsZero() && !req.Since.Before(h.Oldest)
		if !h.Whole && !covered {
			ask = append(ask, src)
		}
	}
	out.Complete = len(ask) == 0
	if len(ask) == 0 {
		return out, nil
	}
	step := olderLogStep
	if !req.Since.IsZero() {
		step = req.Room
	}
	step = max(workloadLogMinTail, min(step, req.Room)/len(ask))

	type result struct {
		src     olderSource
		lines   []stampedLog
		n, tail int64 // lines the server sent, of the tail asked for
		trimmed bool  // the byte bound left its oldest lines out
		err     error
	}
	maxBytes := olderLogBytes / len(ask)
	results := make([]result, len(ask))
	sem := make(chan struct{}, olderLogRequests)
	var wg sync.WaitGroup
	for i, src := range ask {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer s.recoverTo("older logs")
			sem <- struct{}{}
			defer func() { <-sem }()
			tail := int64(held[src.prefix].Lines + step)
			opts := cluster.LogOptions{
				Namespace: tab.Ref.Namespace, Pod: src.pod, Container: src.container,
				Previous: tab.Logs.Previous, Timestamps: true, TailLines: tail,
			}
			if !req.Since.IsZero() {
				opts.SinceTime = req.Since.Truncate(time.Second)
			}
			lines, n, trimmed, err := readOlder(ctx, streamer, opts, held[src.prefix].Oldest, maxBytes)
			results[i] = result{src: src, lines: lines, n: int64(n), tail: tail, trimmed: trimmed, err: err}
		}()
	}
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return OlderLines{}, err
	}

	var all []stampedLog
	cut := map[string]bool{}
	for _, r := range results {
		if r.err != nil {
			out.Notes = append(out.Notes, NoticePrefix+" "+orPod(r.src)+": "+oneLine(r.err.Error()))
			continue
		}
		cut[r.src.prefix] = r.trimmed
		for i := range r.lines {
			r.lines[i].source = r.src.prefix
		}
		all = append(all, r.lines...)
	}
	slices.SortStableFunc(all, func(a, b stampedLog) int { return a.at.Compare(b.at) })
	if len(all) > req.Room { // the newest of them: the oldest held lines stay next to them
		for _, l := range all[:len(all)-req.Room] {
			cut[l.source] = true
		}
		all = all[len(all)-req.Room:]
	}
	// A container the server sent fewer lines of than asked, none of them cut by the room, is
	// held from its start (or, with Since, from then on).
	out.Complete = true
	for _, r := range results {
		reached := r.err == nil && r.n < r.tail && !cut[r.src.prefix]
		out.Complete = out.Complete && reached
		if reached && req.Since.IsZero() {
			out.Whole = append(out.Whole, r.src.prefix)
		}
	}
	for _, l := range all {
		if l.source != "" {
			out.Lines = append(out.Lines, l.source+" "+l.line)
		} else {
			out.Lines = append(out.Lines, l.line)
		}
	}
	return out, nil
}

// orPod names a source in a note: its prefix, or the pod and container of a pod's tab.
func orPod(src olderSource) string {
	if src.prefix != "" {
		return src.prefix
	}
	return src.pod + "/" + src.container
}

// olderSources returns the containers a tab shows: a pod's tab's container, or the selected
// containers of a controlling resource's pods that have a log (for Previous, a previous instance).
func (s *Session) olderSources(ctx context.Context, tab DockTab) ([]olderSource, error) {
	if tab.Ref.Kind == logKindPods {
		return []olderSource{{pod: tab.Ref.Name, container: tab.Container}}, nil
	}
	pods, err := s.workloadPods(ctx, *tab.Ref)
	if err != nil {
		return nil, err
	}
	sel := selectionOf(tab)
	var out []olderSource
	for _, pod := range pods {
		for _, cs := range podContainerStatuses(pod) {
			t := logTarget{pod: pod.GetName(), container: cs.name, named: cs.init && !cs.sidecar}
			switch {
			case !sel.selects(t):
				continue
			case tab.Logs.Previous:
				if model.NestedMap(cs.status, "lastState", "terminated") == nil {
					continue
				}
			case model.NestedMap(cs.status, "state", "running") == nil && model.NestedMap(cs.status, "state", "terminated") == nil:
				continue // waiting: no log yet
			}
			out = append(out, olderSource{prefix: t.pod + "/" + t.container, pod: t.pod, container: t.container})
		}
	}
	return out, nil
}

// stampedLog is one line of an older-lines response: its timestamp and the line as the server
// sent it (timestamp included, no line feed), and the source it belongs to.
type stampedLog struct {
	at     time.Time
	line   string
	source string
}

// readOlder reads one container's lines and keeps those older than before (all when before is
// zero), and those at it, which the tab may or may not hold. It returns them, oldest first, how
// many lines the server sent, and whether maxBytes left the oldest of them out (it keeps the
// newest that fit). Lines without a timestamp cannot be placed and are left out; longer lines are
// cut as the stream cuts them.
func readOlder(ctx context.Context, streamer logStreamer, opts cluster.LogOptions, before time.Time, maxBytes int) ([]stampedLog, int, bool, error) {
	rc, err := streamer.StreamLogs(ctx, opts)
	if err != nil {
		return nil, 0, false, err
	}
	defer rc.Close()
	var out []stampedLog
	n, first, size, trimmed := 0, 0, 0, false
	br := bufio.NewReaderSize(rc, workloadLogLineBytes)
	for int64(n) < opts.TailLines { // the server sends no more; a bound all the same
		line, err := br.ReadSlice('\n')
		full := errors.Is(err, bufio.ErrBufferFull)
		if len(line) > 0 {
			n++
			if ts, _, ok := splitLogTimestamp(line); ok && (before.IsZero() || !ts.After(before)) {
				l := stampedLog{at: ts, line: string(trimLineEnd(line))}
				out = append(out, l)
				size += len(l.line)
				for size > maxBytes && first < len(out) {
					size -= len(out[first].line)
					out[first] = stampedLog{} // its text can go
					first++
					trimmed = true
				}
			}
		}
		for full {
			_, err = br.ReadSlice('\n')
			full = errors.Is(err, bufio.ErrBufferFull)
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, n, false, fmt.Errorf("reading the log: %w", err)
		}
	}
	return out[first:], n, trimmed, nil
}

// trimLineEnd drops a line's line feed and carriage return.
func trimLineEnd(line []byte) []byte {
	for len(line) > 0 && (line[len(line)-1] == '\n' || line[len(line)-1] == '\r') {
		line = line[:len(line)-1]
	}
	return line
}
