package testkit

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// Container logs of a FakeAPIServer: GET .../pods/<name>/log with follow, timestamps, sinceTime
// (RFC 3339, whole seconds, as the API carries it) and tailLines, as the API server serves them.

type fakeLogLine struct {
	at   time.Time
	text string
}

// fakeLogs holds the containers' lines and the open follow streams, by "namespace/pod/container".
type fakeLogs struct {
	lines     map[string][]fakeLogLine
	followers map[string][]chan fakeLogLine
}

// AddLog appends a line to a container's log and sends it to the streams that follow it.
func (s *FakeAPIServer) AddLog(namespace, pod, container string, at time.Time, text string) {
	key := namespace + "/" + pod + "/" + container
	l := fakeLogLine{at: at, text: text}
	s.mu.Lock()
	if s.logs.lines == nil {
		s.logs.lines, s.logs.followers = map[string][]fakeLogLine{}, map[string][]chan fakeLogLine{}
	}
	s.logs.lines[key] = append(s.logs.lines[key], l)
	followers := append([]chan fakeLogLine(nil), s.logs.followers[key]...)
	s.mu.Unlock()
	for _, ch := range followers {
		select {
		case ch <- l:
		default: // a stream that does not read loses the line, as a slow client would
		}
	}
}

// LogFollowers returns how many streams follow a container's log now.
func (s *FakeAPIServer) LogFollowers(namespace, pod, container string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.logs.followers[namespace+"/"+pod+"/"+container])
}

func (s *FakeAPIServer) serveLog(w http.ResponseWriter, r *http.Request, req fakeRequest) {
	s.count("get", "pods/log")
	q := r.URL.Query()
	container := q.Get("container")
	key := req.namespace + "/" + req.name + "/" + container
	timestamps := q.Get("timestamps") == "true"
	var since time.Time
	if v := q.Get("sinceTime"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeStatus(w, http.StatusBadRequest, "BadRequest", "invalid sinceTime")
			return
		}
		since = t
	}
	tail, _ := strconv.Atoi(q.Get("tailLines"))
	var ch chan fakeLogLine
	s.mu.Lock()
	var initial []fakeLogLine
	for _, l := range s.logs.lines[key] {
		if since.IsZero() || !l.at.Before(since) {
			initial = append(initial, l)
		}
	}
	if tail > 0 && len(initial) > tail {
		initial = initial[len(initial)-tail:]
	}
	if q.Get("follow") == "true" {
		if s.logs.followers == nil {
			s.logs.lines, s.logs.followers = map[string][]fakeLogLine{}, map[string][]chan fakeLogLine{}
		}
		ch = make(chan fakeLogLine, 64)
		s.logs.followers[key] = append(s.logs.followers[key], ch)
	}
	s.mu.Unlock()
	if ch != nil {
		defer func() {
			s.mu.Lock()
			list := s.logs.followers[key]
			for i, c := range list {
				if c == ch {
					s.logs.followers[key] = append(list[:i:i], list[i+1:]...)
					break
				}
			}
			s.mu.Unlock()
		}()
	}
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	flusher, _ := w.(http.Flusher)
	write := func(l fakeLogLine) bool {
		line := l.text + "\n"
		if timestamps {
			line = l.at.UTC().Format(time.RFC3339Nano) + " " + line
		}
		if _, err := fmt.Fprint(w, line); err != nil {
			return false
		}
		if flusher != nil {
			flusher.Flush()
		}
		return true
	}
	for _, l := range initial {
		if !write(l) {
			return
		}
	}
	if ch == nil {
		return
	}
	for {
		select {
		case l := <-ch:
			if !write(l) {
				return
			}
		case <-r.Context().Done():
			return
		case <-s.stopping:
			return
		}
	}
}
