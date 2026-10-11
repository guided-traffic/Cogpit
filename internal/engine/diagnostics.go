package engine

import (
	"fmt"
	"time"

	"github.com/guided-traffic/cogpit/internal/cache"
)

// diagnosticsInterval is the shortest time between two EventDiagnostics.
const diagnosticsInterval = time.Second

// Diagnostics returns the request counters and cache stats of the connected backend.
func (s *Session) Diagnostics() Diagnostics {
	d := Diagnostics{Context: s.UIState().Context, Snapshots: s.state.snapshots.Load(), Informers: []cache.InformerStats{}}
	b := s.Backend()
	if b == nil {
		return d
	}
	d.Context, d.Demo = b.Context, b.Demo
	if b.Conn != nil {
		if c := b.Conn.Counter(); c != nil {
			d.Requests = c.Snapshot()
		}
	}
	if b.Cache != nil {
		d.Informers = b.Cache.Stats()
	}
	if b.DemoStats != nil {
		d.DemoStats = s.demoStats(b)
	}
	return d
}

// demoStats calls the demo cluster's counters; a panic there must not break the status bar.
func (s *Session) demoStats(b *Backend) (out any) {
	defer func() {
		if r := recover(); r != nil {
			s.log.Error("engine: demo stats panicked", "panic", fmt.Sprint(r))
			out = nil
		}
	}()
	return b.DemoStats()
}

// diagnosticsLoop emits EventDiagnostics at most once per second while connected, and only when
// a counter moved.
func (s *Session) diagnosticsLoop(stop <-chan struct{}) {
	t := time.NewTicker(diagnosticsInterval)
	defer t.Stop()
	var last string
	for {
		select {
		case <-stop:
			return
		case <-t.C:
		}
		d := s.Diagnostics()
		var events int64
		for _, inf := range d.Informers {
			events += inf.Events
		}
		fp := fmt.Sprint(d.Snapshots, d.Requests.Total, len(d.Informers), events)
		if fp == last {
			continue
		}
		last = fp
		select {
		case <-stop:
			return
		default:
		}
		s.emit(Event{Type: EventDiagnostics})
	}
}
