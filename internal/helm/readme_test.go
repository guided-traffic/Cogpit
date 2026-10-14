package helm

import (
	"context"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/guided-traffic/cogpit/internal/testkit"
)

// The README is found as helm show readme finds it, at the archive's root in any case: README.md,
// README.txt or README; a README in a subfolder is not the chart's, and a chart without one has
// none.
func TestChartReadme(t *testing.T) {
	md := webChart
	md.Readme = "# Web\n\nA **web** server.\n"
	txt := webChart
	txt.Version, txt.Readme, txt.ReadmeName = "0.1.0", "plain text\n", "readme.TXT"
	none := webChart
	none.Version = "0.0.9"
	r := testkit.NewHelmRepo(t, md, txt, none)
	f := &Fetcher{Client: r.Client()}
	idx, err := f.DownloadIndex(context.Background(), PublicRepository("test", r.URL), t.TempDir()+"/index.yaml")
	if err != nil {
		t.Fatal(err)
	}
	web, _ := idx.Chart("web")
	for _, tc := range []struct {
		version, file, text string
	}{
		{"0.2.0", "README.md", md.Readme},
		{"0.1.0", "readme.TXT", "plain text\n"},
		{"0.0.9", "", ""},
	} {
		v, _ := web.Version(tc.version)
		c, err := f.FetchChart(context.Background(), PublicRepository("test", r.URL), "web", v)
		if err != nil {
			t.Fatal(err)
		}
		got := c.Readme()
		if got.File != tc.file || got.Text != tc.text || got.Truncated || got.Size != len(tc.text) {
			t.Errorf("%s: README %+v", tc.version, got)
		}
	}

	nested := BuildChartWithFile(t, "docs/README.md", "not the chart's")
	if got := nested.Readme(); got.File != "" {
		t.Errorf("a README in a subfolder: %+v", got)
	}
	var empty *Chart
	if got := empty.Readme(); got.File != "" {
		t.Errorf("no chart: %+v", got)
	}
}

// BuildChartWithFile is webChart with one more file in its archive, loaded as an archive.
func BuildChartWithFile(t *testing.T, name, text string) *Chart {
	t.Helper()
	c := webChart
	c.Readme, c.ReadmeName = text, name
	r := testkit.NewHelmRepo(t, c)
	ch, err := LoadChartArchive(r.Archive("web-0.2.0.tgz"))
	if err != nil {
		t.Fatal(err)
	}
	return ch
}

// A README is remote text: it is cut at MaxReadmeBytes (at a line break near the bound, else at a
// character, never inside one), invalid UTF-8 is replaced, control characters but tab and line
// feed are dropped, a carriage return ends a line, and bidirectional overrides are dropped.
func TestCleanReadme(t *testing.T) {
	text, cut := CleanReadme([]byte("a\tb\r\nc\rd\x00e\x1bf\u202egh\u2066i\x7fj\xffk\u0085l"))
	if text != "a\tb\nc\ndef"+"gh"+"ij�kl" || cut {
		t.Errorf("cleaned %q %v", text, cut)
	}

	lines := strings.Repeat("0123456789abcdef\n", MaxReadmeBytes/17+100)
	text, cut = CleanReadme([]byte(lines))
	if !cut || len(text) > MaxReadmeBytes || !strings.HasSuffix(text, "\n") || len(text) < MaxReadmeBytes-17 {
		t.Errorf("cut lines: %d bytes, cut %v, ends %q", len(text), cut, text[len(text)-3:])
	}

	runes := strings.Repeat("ä", MaxReadmeBytes) // two bytes each: the bound falls inside none
	text, cut = CleanReadme([]byte("x" + runes))
	if !cut || len(text) > MaxReadmeBytes || !utf8.ValidString(text) || strings.ContainsRune(text, utf8.RuneError) {
		t.Errorf("cut runes: %d bytes, cut %v, valid %v", len(text), cut, utf8.ValidString(text))
	}

	short := "# Title\n"
	if text, cut := CleanReadme([]byte(short)); text != short || cut {
		t.Errorf("short %q %v", text, cut)
	}
	invalid := strings.Repeat("\xffa", MaxReadmeBytes/2) // each invalid byte becomes three
	if text, cut := CleanReadme([]byte(invalid)); !cut || len(text) > MaxReadmeBytes || !utf8.ValidString(text) {
		t.Errorf("invalid: %d bytes, cut %v", len(text), cut)
	}
}

// A README is untrusted input: whatever its bytes, cleaning keeps the bounds and never panics.
func FuzzCleanReadme(f *testing.F) {
	f.Add([]byte("# Web\n\n```sh\nhelm install\n```\n"))
	f.Add([]byte("\u202e\x00\r\xff"))
	f.Fuzz(func(t *testing.T, data []byte) {
		text, _ := CleanReadme(data)
		if len(text) > MaxReadmeBytes || !utf8.ValidString(text) || strings.ContainsAny(text, "\x00\r\u202e") {
			t.Errorf("cleaned %q", text)
		}
	})
}
