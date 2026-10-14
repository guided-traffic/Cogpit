package fyneui

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

// TestWrapFlowBreaksAWordWiderThanTheLine: a digest wider than the line breaks where it must, every
// line fits, the sentence after it flows on from the digest's rest, and nothing is lost; text
// without such a word wraps as wrapLines wraps it.
func TestWrapFlowBreaksAWordWiderThanTheLine(t *testing.T) {
	test.NewTempApp(t)
	const size, width = 14, 300
	digest := "sha256:" + strings.Repeat("0123456789abcdef", 4)
	text := "Image: alpine:3.24, pinned by digest " + digest + ". The pod ends by itself."
	lines := wrapFlow(text, width, size, fyne.TextStyle{})
	for _, l := range lines {
		if w := fyne.MeasureText(l, size, fyne.TextStyle{}).Width; w > width {
			t.Errorf("line %q is %.0f wide, more than %d", l, w, width)
		}
	}
	if got := strings.ReplaceAll(strings.Join(lines, ""), " ", ""); got != strings.ReplaceAll(text, " ", "") {
		t.Errorf("text lost: %q", lines)
	}
	if !strings.Contains(strings.Join(lines, "|"), "cdef. The") {
		t.Errorf("the sentence after the digest does not flow on from its rest: %q", lines)
	}
	plain := "the node's processes and the shell enters the node's namespaces as root"
	if got, want := wrapFlow(plain, width, size, fyne.TextStyle{}), wrapLines(plain, width, size, fyne.TextStyle{}); strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("plain text wrapped %q, want %q", got, want)
	}
}
