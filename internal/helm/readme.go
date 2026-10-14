package helm

import (
	"strings"
	"unicode/utf8"
)

// A chart's README (G11, HELM2): the chart drawer shows the README of the version it shows, read
// from that version's archive, which is downloaded the way an install downloads it (FetchChart:
// bounded, checked against the index's digest, the repository's credentials only for its own
// host). The README is found as helm show readme finds it: the first file at the archive's root
// named README.md, README.txt or README, in any case. It is remote text written by the chart's
// author: Cogpit keeps at most MaxReadmeBytes of it and drops what could disguise it on screen
// (control characters, bidirectional overrides); the window renders it as Markdown without
// loading its images.

// MaxReadmeBytes bounds the README Cogpit keeps of a chart; the rest is cut and the drawer says so
// (Cogpit's default).
const MaxReadmeBytes = 256 << 10

// readmeNames are the names of a chart's README, as helm show readme looks for them.
var readmeNames = []string{"readme.md", "readme.txt", "readme"}

// Readme is a chart archive's README as Cogpit keeps it.
type Readme struct {
	// File is the README's name in the archive ("README.md"); empty when the archive holds none.
	File string `json:"file,omitempty"`
	// Text is the README: at most MaxReadmeBytes, valid UTF-8, without control characters (but
	// tabs and line breaks) or bidirectional overrides.
	Text string `json:"text,omitempty"`
	// Size is the README's size in bytes as the archive holds it.
	Size int `json:"size,omitempty"`
	// Truncated: Text holds the README's first MaxReadmeBytes only.
	Truncated bool `json:"truncated,omitempty"`
}

// Readme returns the chart's README; File is empty when the archive holds none.
func (c *Chart) Readme() Readme {
	if c == nil || c.chart == nil {
		return Readme{}
	}
	for _, f := range c.chart.Files {
		if f == nil {
			continue
		}
		if IsReadmeName(f.Name) {
			text, cut := CleanReadme(f.Data)
			return Readme{File: f.Name, Text: text, Size: len(f.Data), Truncated: cut}
		}
	}
	return Readme{}
}

// IsReadmeName reports whether a file at a chart's root is its README, by its name.
func IsReadmeName(name string) bool {
	for _, n := range readmeNames {
		if strings.EqualFold(name, n) {
			return true
		}
	}
	return false
}

// CleanReadme returns a README's text as Cogpit keeps it: its first MaxReadmeBytes (cut at the
// last line break before the bound when there is one in its last 4 KiB, else at a character),
// invalid UTF-8 replaced, control characters but tab and line feed dropped (a carriage return
// ends a line), and the bidirectional embeddings, overrides and isolates dropped, which could make
// a command in it read otherwise than it runs. cut reports whether anything was cut off.
func CleanReadme(data []byte) (text string, cut bool) {
	if len(data) > MaxReadmeBytes {
		data, cut = data[:MaxReadmeBytes], true
		if i := lastLineBreak(data); i >= MaxReadmeBytes-4<<10 {
			data = data[:i+1]
		} else {
			data = dropPartialRune(data)
		}
	}
	s := strings.ToValidUTF8(string(data), "�")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == '\r':
			b.WriteByte('\n')
		case r == '\t' || r == '\n':
			b.WriteRune(r)
		case r < 0x20 || r == 0x7f || (r >= 0x80 && r < 0xa0):
		case r >= 0x202a && r <= 0x202e, r >= 0x2066 && r <= 0x2069:
		default:
			b.WriteRune(r)
		}
	}
	s = b.String()
	if len(s) > MaxReadmeBytes { // replaced invalid bytes grew it
		s, cut = s[:MaxReadmeBytes], true
		for len(s) > 0 && !utf8.ValidString(s) {
			s = s[:len(s)-1]
		}
	}
	return s, cut
}

// dropPartialRune drops a character the cut split from data's end.
func dropPartialRune(data []byte) []byte {
	for i := len(data) - 1; i >= 0 && i >= len(data)-utf8.UTFMax; i-- {
		if utf8.RuneStart(data[i]) {
			if !utf8.FullRune(data[i:]) {
				return data[:i]
			}
			break
		}
	}
	return data
}

// lastLineBreak is the index of data's last '\n', or -1.
func lastLineBreak(data []byte) int {
	for i := len(data) - 1; i >= 0; i-- {
		if data[i] == '\n' {
			return i
		}
	}
	return -1
}
