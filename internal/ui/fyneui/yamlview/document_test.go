package yamlview

import (
	"strings"
	"testing"
)

func docOf(s string) *document {
	d := newDocument()
	d.setText(s)
	return &d
}

func TestDocumentSetTextRoundTrip(t *testing.T) {
	for _, s := range []string{"", "a", "a\n", "\n\n", "a\nb\nc", "x\n\ny\n"} {
		d := docOf(s)
		if got := d.joined(); got != s {
			t.Errorf("round trip %q: got %q", s, got)
		}
		if n := strings.Count(s, "\n") + 1; len(d.lines) != n {
			t.Errorf("%q: %d lines, want %d", s, len(d.lines), n)
		}
	}
}

func TestDocumentCRLF(t *testing.T) {
	d := docOf("a: 1\r\nb: 2\r\n")
	if got := d.joined(); got != "a: 1\nb: 2\n" {
		t.Errorf("CRLF: got %q", got)
	}
	if d.lines[0] != "a: 1" {
		t.Errorf("line 0 = %q", d.lines[0])
	}
}

func TestDocumentSetTextSharesMemory(t *testing.T) {
	s := strings.Repeat("key: value\n", 1000)
	d := docOf(s)
	// The lines are slices of s: the joined text is s itself and no line was copied.
	if d.joined() != s {
		t.Fatal("joined differs")
	}
	if got := strings.Join(d.lines, "\n"); got != s {
		t.Fatal("lines differ")
	}
}

func TestDocumentInsert(t *testing.T) {
	d := docOf("ab\ncd")
	if p := d.insert(pos{0, 1}, "X"); p != (pos{0, 2}) || d.joined() != "aXb\ncd" {
		t.Fatalf("insert rune: %v %q", p, d.joined())
	}
	if p := d.insert(pos{0, 3}, "1\n2\n3"); p != (pos{2, 1}) || d.joined() != "aXb1\n2\n3\ncd" {
		t.Fatalf("insert lines at end of line: %v %q", p, d.joined())
	}
	if p := d.insert(pos{0, 0}, "\n"); p != (pos{1, 0}) || d.joined() != "\naXb1\n2\n3\ncd" {
		t.Fatalf("newline at start: %v %q", p, d.joined())
	}
	if p := d.insert(pos{99, 99}, "!"); p != (pos{4, 3}) || !strings.HasSuffix(d.joined(), "cd!") {
		t.Fatalf("clamped insert: %v %q", p, d.joined())
	}
	if p := d.insert(pos{0, 0}, "a\r\nb\rc"); p != (pos{2, 1}) || !strings.HasPrefix(d.joined(), "a\nb\nc\n") {
		t.Fatalf("newline normalisation: %v %q", p, d.joined())
	}
}

func TestDocumentRemove(t *testing.T) {
	d := docOf("one\ntwo\nthree")
	if got := d.remove(pos{0, 3}, pos{1, 0}); got != "\n" || d.joined() != "onetwo\nthree" {
		t.Fatalf("join lines: removed %q, doc %q", got, d.joined())
	}
	if got := d.remove(pos{1, 2}, pos{0, 1}); got != "netwo\nth" || d.joined() != "oree" {
		t.Fatalf("reversed range across lines: removed %q, doc %q", got, d.joined())
	}
	if got := d.remove(pos{0, 1}, pos{0, 1}); got != "" || d.joined() != "oree" {
		t.Fatalf("empty range: removed %q, doc %q", got, d.joined())
	}
	if got := d.remove(pos{-5, 0}, pos{50, 0}); got != "oree" || d.joined() != "" || len(d.lines) != 1 {
		t.Fatalf("clamped full range: removed %q, doc %q", got, d.joined())
	}
}

func TestDocumentPrevNext(t *testing.T) {
	d := docOf("aé\nb")
	p := pos{0, 0}
	var walk []pos
	for range 6 {
		walk = append(walk, p)
		p = d.next(p)
	}
	want := []pos{{0, 0}, {0, 1}, {0, 3}, {1, 0}, {1, 1}, {1, 1}}
	for i := range want {
		if walk[i] != want[i] {
			t.Fatalf("next walk %v, want %v", walk, want)
		}
	}
	for i := len(want) - 2; i > 0; i-- {
		if got := d.prev(want[i]); got != want[i-1] {
			t.Errorf("prev(%v) = %v, want %v", want[i], got, want[i-1])
		}
	}
	if got := d.prev(pos{0, 0}); got != (pos{0, 0}) {
		t.Errorf("prev at start = %v", got)
	}
}

func TestAlignOffset(t *testing.T) {
	cases := []struct {
		line string
		off  int
		want int
	}{
		{"aé", 2, 1},        // inside é
		{"a€b", 3, 1},       // inside €
		{"\x80\x80", 1, 1},  // invalid bytes are one-byte runes
		{"a\xe2\x82", 2, 2}, // truncated sequence
		{"€\x80", 3, 3},     // stray continuation after a valid rune
		{"\x80\x80\x80\x80", 3, 3},
	}
	for _, tc := range cases {
		if got := alignOffset(tc.line, tc.off); got != tc.want {
			t.Errorf("alignOffset(%q, %d) = %d, want %d", tc.line, tc.off, got, tc.want)
		}
	}
}

func TestColumns(t *testing.T) {
	line := "a\tb世c"
	cols := []int{0, 1, 4, 5, 5, 5, 7, 8} // per byte offset; offsets inside 世 map to its start
	for off, want := range cols {
		if got := displayCol(line, alignOffset(line, off)); got != want {
			t.Errorf("displayCol(%d) = %d, want %d", off, got, want)
		}
	}
	for col, want := range map[int]int{0: 0, 1: 1, 2: 1, 3: 2, 4: 2, 5: 3, 6: 3, 7: 6, 8: 7, 99: 7} {
		if got := offsetForCol(line, col); got != want {
			t.Errorf("offsetForCol(%d) = %d, want %d", col, got, want)
		}
	}
}

func TestStatePrefix(t *testing.T) {
	d := docOf("a: |\n  x\n  y\nb: 1\nc: 2")
	if st := d.stateAt(2); st.ctx() != ctxBlock {
		t.Fatalf("line 2 state %d, want block scalar", st)
	}
	if len(d.states) != 3 {
		t.Fatalf("states computed for %d lines, want only the prefix up to line 2", len(d.states))
	}
	// Line 3 still starts inside the scalar; reading it ends the scalar.
	if toks, _ := tokenize(d.lines[3], d.stateAt(3)); len(toks) == 0 || toks[0].kind != tokKey {
		t.Fatalf("line 3 tokens %v, want a key", toks)
	}
	if st := d.stateAt(4); st != ctxNormal {
		t.Fatalf("line 4 state %d, want normal", st)
	}
	// An edit cuts the prefix back to the edited line and the scalar's end moves with it.
	d.insert(pos{0, 3}, "x")
	if len(d.states) != 1 {
		t.Fatalf("after edit %d states kept, want 1", len(d.states))
	}
	if st := d.stateAt(1); st != ctxNormal {
		t.Fatalf("line 1 after removing the header: state %d", st)
	}
}

func TestRuneWidth(t *testing.T) {
	for r, want := range map[rune]int{'a': 1, 'é': 1, '世': 2, '한': 2, '😀': 2, 'Ａ': 2, '→': 1} {
		if got := runeWidth(r); got != want {
			t.Errorf("runeWidth(%q) = %d, want %d", r, got, want)
		}
	}
}

// FuzzDocumentEdits applies random inserts and removals to the document and to a plain string and
// checks that both agree: the line model never loses or invents text.
func FuzzDocumentEdits(f *testing.F) {
	f.Add("a: 1\nb: 2\n", []byte{0, 3, 1, 'x', 2, 5, 9, 0, 1, '\n'})
	f.Add("", []byte{1, 0, 1, '\n', 1, 0, 1, '\r'})
	f.Add("é世\x80\n\t", []byte{3, 1, 0, 2, 2, 4})
	f.Fuzz(func(t *testing.T, text string, ops []byte) {
		d := docOf(text)
		ref := normaliseNewlines(text)
		if d.joined() != ref {
			t.Fatalf("setText: %q != %q", d.joined(), ref)
		}
		for i := 0; i+2 < len(ops) && i < 300; i += 3 {
			a := d.clamp(pos{int(ops[i+1]) % (len(d.lines) + 1), int(ops[i+2])})
			fa := flatOffset(d, a)
			switch ops[i] % 4 {
			case 0, 1: // insert one byte, a newline sometimes
				s := string(rune(ops[i+2]))
				if ops[i]%4 == 1 {
					s = "\n" + s
				}
				end := d.insert(a, s)
				ns := normaliseNewlines(s)
				ref = ref[:fa] + ns + ref[fa:]
				if flatOffset(d, end) != fa+len(ns) {
					t.Fatalf("insert end %v at %d, want %d", end, flatOffset(d, end), fa+len(ns))
				}
			case 2: // remove from a to a clamped second position
				b := d.clamp(pos{int(ops[i+2]) % (len(d.lines) + 1), int(ops[i+1])})
				fb := flatOffset(d, b)
				lo, hi := min(fa, fb), max(fa, fb)
				if got := d.remove(a, b); got != ref[lo:hi] {
					t.Fatalf("removed %q, want %q", got, ref[lo:hi])
				}
				ref = ref[:lo] + ref[hi:]
			case 3: // state scan to a line
				d.stateAt(int(ops[i+2]))
			}
			if got := d.joined(); got != ref {
				t.Fatalf("after op %d: %q != %q", i/3, got, ref)
			}
			if d.extent < maxLineLen(d) {
				t.Fatalf("extent %d below the longest line %d", d.extent, maxLineLen(d))
			}
		}
	})
}

// flatOffset converts a position into a byte offset of the joined text.
func flatOffset(d *document, p pos) int {
	n := 0
	for i := 0; i < p.line; i++ {
		n += len(d.lines[i]) + 1
	}
	return n + p.off
}

func maxLineLen(d *document) int {
	n := 0
	for _, l := range d.lines {
		n = max(n, displayCol(l, len(l)))
	}
	return n
}
