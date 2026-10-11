package termview

import (
	"bytes"
	"errors"
	"io"
	"regexp"
	"strings"
	"testing"
	"testing/iotest"
	"unicode/utf8"
)

// filterAll runs input through the filter, fed one byte per Read to exercise every split.
func filterAll(t testing.TB, input string) string {
	t.Helper()
	whole, err := io.ReadAll(newEscFilter(strings.NewReader(input)))
	if err != nil {
		t.Fatalf("filter: %v", err)
	}
	split, err := io.ReadAll(newEscFilter(iotest.OneByteReader(strings.NewReader(input))))
	if err != nil {
		t.Fatalf("filter, one byte per read: %v", err)
	}
	if !bytes.Equal(whole, split) {
		t.Fatalf("filter output depends on read boundaries for %q:\nwhole %q\nsplit %q", input, whole, split)
	}
	return string(whole)
}

func TestEscFilter(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"plain text", "hello, world\r\n", "hello, world\r\n"},
		{"UTF-8", "grüße 集群 🚀\r\n", "grüße 集群 🚀\r\n"},
		{"C0 a terminal needs", "a\x07b\x08c\td\ne\x0bf\x0cg\rh\x0ei\x0fj", "a\x07b\x08c\td\ne\x0bf\x0cg\rh\x0ei\x0fj"},
		{"other C0, DEL and C1 dropped", "a\x00b\x01c\x7fd\u0085e\u009bf\u009dg", "abcdefg"},
		{"invalid UTF-8 dropped", "a\xffb\xc3c", "abc"},

		// Strings: dropped whole, whatever they say.
		{"OSC 52 clipboard write, BEL", "a\x1b]52;c;SGVsbG8=\x07b", "ab"},
		{"OSC 52 clipboard write, ST", "a\x1b]52;c;SGVsbG8=\x1b\\b", "ab"},
		{"OSC 7 working directory", "a\x1b]7;file:///etc\x1b\\b", "ab"},
		{"OSC 0 title", "\x1b]0;you have been pwned\x07$ ", "$ "},
		{"OSC 8 hyperlink", "\x1b]8;;https://example.invalid\x1b\\link\x1b]8;;\x1b\\", "link"},
		{"DCS", "a\x1bP+q544e\x1b\\b", "ab"},
		{"APC", "a\x1b_payload\x1b\\b", "ab"},
		{"PM and SOS", "a\x1b^pm\x1b\\b\x1bXsos\x1b\\c", "abc"},
		{"string aborted by CAN", "a\x1b]0;t\x18b", "ab"},
		{"ESC inside a string starts a new sequence", "a\x1b]0;t\x1b[1mb", "a\x1b[1mb"},

		// Attempts to smuggle a string past the filter.
		{"ESC ESC ]", "\x1b\x1b]7;file:///tmp\x07x", "x"},
		{"ESC invalid ]", "\x1b\xff]7;file:///tmp\x07x", "x"},
		{"ESC NUL ]", "\x1b\x00]7;file:///tmp\x07x", "]7;file:///tmp\x07x"},
		{"string then bare ]", "\x1b\x1b]0;x\x07]7;file:///etc\x07", "]7;file:///etc\x07"},
		{"CSI cut by ESC [", "\x1b[5\x1b[ix", "x"},
		{"CSI cut by ESC ]", "\x1b[1;\x1b]52;c;QQ==\x07x", "x"},

		// Control sequences: canonical ones pass, others are dropped.
		{"SGR", "\x1b[1;31mred\x1b[0m", "\x1b[1;31mred\x1b[0m"},
		{"SGR true colour", "\x1b[38;2;255;128;0mx", "\x1b[38;2;255;128;0mx"},
		{"cursor and erase", "\x1b[H\x1b[2J\x1b[10;20H\x1b[K\x1b[3A\x1b[2C", "\x1b[H\x1b[2J\x1b[10;20H\x1b[K\x1b[3A\x1b[2C"},
		{"DEC private mode", "\x1b[?1049h\x1b[?25l\x1b[?2004h", "\x1b[?1049h\x1b[?25l\x1b[?2004h"},
		{"device attributes", "\x1b[c\x1b[>c", "\x1b[c\x1b[>c"},
		{"mode without ? panics the widget", "\x1b[h\x1b[4h\x1b[0lx", "x"},
		{"non-ASCII final panics the widget", "\x1b[éx", "éx"},
		{"media copy (print mode)", "\x1b[5isecret\x1b[4i", "secret"},
		{"window operations", "\x1b[8;100;200t\x1b[21tx", "x"},
		{"intermediates", "\x1b[2 qx\x1b[!p", "x"},
		{"colon sub-parameters", "\x1b[38:2::255:0:0mx", "x"},
		{"marker on SGR", "\x1b[?1mx", "x"},
		{"unknown final", "\x1b[6nx\x1b[5~", "x"},
		{"repeat count clamped", "a\x1b[999999999b", "a\x1b[9999b"},
		{"overflowing number clamped", "\x1b[99999999999999999999999L", "\x1b[9999L"},
		{"leading zeros kept", "\x1b[00005A", "\x1b[00005A"},
		{"delete characters is preceded by erase", "\x1b[3P", "\x1b[X\x1b[3P"},
		{"C0 inside CSI takes effect first", "\x1b[1\n2m", "\n\x1b[12m"},
		{"CSI aborted by CAN", "\x1b[12\x18x", "x"},

		// Other escapes.
		{"save, restore, index, keypad", "\x1b7\x1b8\x1bD\x1b=\x1b>", "\x1b7\x1b8\x1bD\x1b=\x1b>"},
		{"reverse index becomes scroll down", "\x1bM", "\x1b[T"},
		{"charset designation", "\x1b(0qqq\x1b(B", "\x1b(0qqq\x1b(B"},
		{"other nF escapes", "\x1b#8\x1b%Gx", "x"},
		{"other escapes", "\x1bc\x1bHx", "x"},
		{"ESC before control char", "\x1b\nx", "x"},
		{"trailing ESC", "x\x1b", "x"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := filterAll(t, c.in); got != c.want {
				t.Errorf("filter(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestEscFilterBoundsStrings(t *testing.T) {
	// An unterminated OSC ends after maxStringLen bytes; what follows is shown again.
	in := "\x1b]" + strings.Repeat("A", maxStringLen+10) + "visible"
	if got, want := filterAll(t, in), strings.Repeat("A", 9)+"visible"; got != want {
		t.Errorf("tail after an overlong string = %q, want %q", got, want)
	}
	// An overlong control sequence is dropped whole.
	in = "\x1b[" + strings.Repeat("1;", maxCSIParams) + "mx"
	if got := filterAll(t, in); got != "x" {
		t.Errorf("overlong CSI: %q, want x", got)
	}
}

func TestEscFilterReadsEndOnRuneBoundaries(t *testing.T) {
	in := strings.Repeat("ä集🚀", 5000)
	f := newEscFilter(strings.NewReader(in))
	var out []byte
	p := make([]byte, 7) // smaller than the filter's buffer and not a multiple of any rune size
	for {
		n, err := f.Read(p)
		if !utf8.Valid(p[:n]) {
			t.Fatalf("Read returned a split rune: %q", p[:n])
		}
		out = append(out, p[:n]...)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if string(out) != in {
		t.Error("output differs from input")
	}
}

func TestEscFilterPassesErrorsAfterOutput(t *testing.T) {
	boom := errors.New("boom")
	f := newEscFilter(io.MultiReader(strings.NewReader("abc"), iotest.ErrReader(boom)))
	got, err := io.ReadAll(f)
	if string(got) != "abc" || !errors.Is(err, boom) {
		t.Errorf("ReadAll = %q, %v; want abc, boom", got, err)
	}
}

// canonical is everything the filter may emit after an ESC.
var canonical = regexp.MustCompile(`^\x1b(\[[?>]?[0-9;]*[@A-DdHfGhLlmJKPrsSucXMTb]|[78D=>]|[()][\x30-\x7e])`)

// FuzzEscFilter checks the filter's invariants on arbitrary input: valid UTF-8 out, no string
// introducer, every escape in canonical form, numbers clamped.
func FuzzEscFilter(f *testing.F) {
	for _, seed := range []string{
		"\x1b]52;c;SGVsbG8=\x07", "\x1b\x1b]7;file:///tmp\x07", "\x1b[5\x1b[i", "\x1b[é", "\x1b[h",
		"\x1b[999999b", "\x1bP+q\x1b\\", "\x1b(0", "\x1b[1\n2m", "plain ✓", "\x1b[?1049h\x1b[3P\x1bM",
	} {
		f.Add([]byte(seed))
	}
	clampedNumber := regexp.MustCompile(`[0-9]{5,}`)
	f.Fuzz(func(t *testing.T, in []byte) {
		out := filterAll(t, string(in))
		if !utf8.ValidString(out) {
			t.Fatalf("invalid UTF-8 out: %q", out)
		}
		for i := 0; i < len(out); i++ {
			if out[i] != esc {
				continue
			}
			m := canonical.FindString(out[i:])
			if m == "" {
				t.Fatalf("non-canonical escape at %d in %q (input %q)", i, out, in)
			}
			if strings.HasPrefix(m, "\x1b[") {
				for _, num := range clampedNumber.FindAllString(m, -1) {
					if strings.TrimLeft(num, "0") != "" && len(strings.TrimLeft(num, "0")) > 4 {
						t.Fatalf("unclamped number %s in %q", num, m)
					}
				}
			}
		}
	})
}
