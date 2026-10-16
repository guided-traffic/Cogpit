package fyneui

import (
	"encoding/json"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"

	"github.com/guided-traffic/cogpit/internal/metrics"
	"github.com/guided-traffic/cogpit/internal/model"
)

// tapSeg taps the middle of a source line's piece, as a click there would.
func tapSeg(l *sourceLine, text string) bool {
	for i, sp := range l.spans(l.Size().Width) {
		if l.segs[i].text == text {
			l.Tapped(&fyne.PointEvent{Position: fyne.NewPos((sp[0]+sp[1])/2, 4)})
			return true
		}
	}
	return false
}

// The drawer's source line names the Prometheus in use with its namespace and service as links
// (Lens's line, captured on a node's drawer): a click opens the Namespace or the Service through
// ui.navigate; the hand pointer and the underline show over a link only; a provider that does
// not say where it runs leaves the line plain.
func TestSourceLineLinksThePrometheus(t *testing.T) {
	sh, f, q, p := promDrawer(t, ThemeDark, withProviders(bothAnswering()))
	l := p.caption
	if l.text != "Displaying metrics from "+promSource {
		t.Fatalf("source line %q", l.text)
	}
	links := l.links()
	if len(links) != 2 || links["monitoring"] == nil || links["prometheus-server"] == nil {
		t.Fatalf("links %v", links)
	}
	if !tapSeg(l, "prometheus-server") {
		t.Fatal("no service piece")
	}
	q.until(t, "ui.navigate to the Service", func() bool { return len(f.callsOf("ui.navigate")) == 1 })
	var nav refParams
	if err := json.Unmarshal(normalize(f.callsOf("ui.navigate")[0].Params), &nav); err != nil ||
		!nav.Ref.Same(model.ObjectRef{Kind: kindServices, Namespace: "monitoring", Name: "prometheus-server"}) || nav.Ref.Context != p.ref.Context {
		t.Errorf("navigate %+v (%v)", nav, err)
	}
	tapSeg(l, "monitoring")
	q.until(t, "ui.navigate to the Namespace", func() bool { return len(f.callsOf("ui.navigate")) == 2 })
	nav = refParams{}
	if err := json.Unmarshal(normalize(f.callsOf("ui.navigate")[1].Params), &nav); err != nil ||
		!nav.Ref.Same(model.ObjectRef{Kind: kindNamespaces, Name: "monitoring"}) || nav.Ref.Context != p.ref.Context {
		t.Errorf("navigate %+v (%v)", nav, err)
	}
	tapSeg(l, "Displaying metrics from Prometheus (")
	settle(sh.win)
	if n := len(f.callsOf("ui.navigate")); n != 2 {
		t.Errorf("a click on the plain text navigated (%d)", n)
	}

	// The pointer: a hand and an underline over a link only.
	sp := l.spans(l.Size().Width)
	l.MouseMoved(&desktop.MouseEvent{PointEvent: fyne.PointEvent{Position: fyne.NewPos((sp[3][0]+sp[3][1])/2, 4)}})
	if l.hover != 3 || l.Cursor() != desktop.PointerCursor {
		t.Errorf("over the service: hover %d, cursor %v", l.hover, l.Cursor())
	}
	l.MouseMoved(&desktop.MouseEvent{PointEvent: fyne.PointEvent{Position: fyne.NewPos(2, 4)}})
	if l.hover != -1 || l.Cursor() != desktop.DefaultCursor {
		t.Errorf("over the text: hover %d, cursor %v", l.hover, l.Cursor())
	}

	// A status without the place (a Prometheus not found): no links.
	segs := sh.sourceSegments("Kubernetes Metrics Server", prometheusGone(), "")
	if len(segs) != 1 || segs[0].onTap != nil {
		t.Errorf("segments without a Prometheus: %+v", segs)
	}
}

// The source line names the Prometheus inside "(namespace/service)" only: a service whose name
// also occurs elsewhere in the line links where the place is.
func TestSourceSegments(t *testing.T) {
	sh := &shell{}
	ps := []metrics.ProviderStatus{{Provider: metrics.PrometheusID, Namespace: "prometheus", ServiceName: "prometheus", Status: metrics.StatusAvailable}}
	segs := sh.sourceSegments("Prometheus (prometheus/prometheus) and Kubernetes Metrics Server", ps, "kind-dev")
	var texts []string
	for _, s := range segs {
		texts = append(texts, s.text)
	}
	want := []string{"Displaying metrics from Prometheus (", "prometheus", "/", "prometheus", ") and Kubernetes Metrics Server"}
	if len(texts) != len(want) {
		t.Fatalf("segments %q", texts)
	}
	for i := range want {
		if texts[i] != want[i] || (segs[i].onTap != nil) != (i == 1 || i == 3) {
			t.Errorf("segment %d: %q (link %v)", i, texts[i], segs[i].onTap != nil)
		}
	}
}
