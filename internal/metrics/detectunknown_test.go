package metrics_test

import (
	"context"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/metrics"
)

// statusOf is a provider's status on the status line, "" when the hub does not list it.
func statusOf(h *metrics.Hub, id string) metrics.Status {
	for _, p := range h.Providers() {
		if p.Provider == id {
			return p.Status
		}
	}
	return ""
}

// DetectUnknown (D-76): with nothing subscribed, the providers nothing looked for are detected
// once, in the background, and done runs after; a second call starts nothing, a provider turned
// off is not asked, and a detection the timeout cut short is not tried again before the retry TTL.
func TestDetectUnknownOnce(t *testing.T) {
	d := newDual(t)
	if s := statusOf(d.h, metrics.PrometheusID); s != metrics.StatusUnknown {
		t.Fatalf("Prometheus before: %s", s)
	}
	done := make(chan struct{})
	if !d.h.DetectUnknown(func() { close(done) }) {
		t.Fatal("no detection started")
	}
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("the detection did not end")
	}
	if l, h := d.liveDet.n.Load(), d.histDet.n.Load(); l != 1 || h != 1 {
		t.Fatalf("detected %d and %d times", l, h)
	}
	if s, p := statusOf(d.h, metrics.ServerID), statusOf(d.h, metrics.PrometheusID); s != metrics.StatusAvailable || p != metrics.StatusAvailable {
		t.Fatalf("after: %s, %s", s, p)
	}
	if d.h.DetectUnknown(nil) {
		t.Error("a second detection started with nothing unknown")
	}
	if gets, lists := d.live.counts(); gets+lists != 0 {
		t.Errorf("the detection polled: %d gets, %d lists", gets, lists)
	}

	// Prometheus off (the Metrics Server alone): it is not asked; the Metrics Server, forgotten by
	// the change, is.
	d.h.SetSource(metrics.SourceMetricsServer)
	done2 := make(chan struct{})
	if !d.h.DetectUnknown(func() { close(done2) }) {
		t.Fatal("the Metrics Server, unknown again, was not detected")
	}
	<-done2
	if l, h := d.liveDet.n.Load(), d.histDet.n.Load(); l != 2 || h != 1 {
		t.Errorf("after the source changed: detected %d and %d times", l, h)
	}
}

// slowDetector answers only when its context ends: a cluster that does not answer.
type slowDetector struct{ calls chan struct{} }

func (d slowDetector) Detect(ctx context.Context) (metrics.Provider, metrics.State) {
	d.calls <- struct{}{}
	<-ctx.Done()
	return nil, metrics.State{Status: metrics.StatusUnavailable}
}

func TestDetectUnknownTimeoutWaitsForTheRetry(t *testing.T) {
	clk := &testClock{t: time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)}
	det := slowDetector{calls: make(chan struct{}, 4)}
	h := metrics.NewHubFor(metrics.Sources{Live: identified{&countingDetector{}, metrics.ServerID, metrics.ServerLabel}, History: det},
		metrics.Options{RequestTimeout: 20 * time.Millisecond, Now: clk.Now})
	t.Cleanup(h.Close)
	ended := make(chan struct{})
	if !h.DetectUnknown(func() { close(ended) }) {
		t.Fatal("no detection started")
	}
	<-ended
	<-det.calls
	if h.DetectUnknown(nil) {
		t.Error("tried again at once after a timeout")
	}
	clk.Advance(metrics.DefaultRetryDetectTTL)
	again := make(chan struct{})
	if !h.DetectUnknown(func() { close(again) }) {
		t.Fatal("not tried again after the retry TTL")
	}
	<-again
	<-det.calls
}
