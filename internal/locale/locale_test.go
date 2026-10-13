package locale

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/model"
)

// TestGoLayout: Unicode's patterns as macOS and CLDR write them, converted to Go layouts: the
// regions' short dates and medium times, quoted literals, fractions, zones and names.
func TestGoLayout(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"dd.MM.yy", "02.01.2006"},                     // de_DE: the two-digit year written with four
		{"HH:mm:ss", "15:04:05"},                       // de_DE, en_GB, fr_FR
		{"M/d/yy", "1/2/2006"},                         // en_US
		{"h:mm:ss\u202fa", "3:04:05 PM"},               // en_US, its narrow no-break space a space
		{"M/d/yy h:mm:ss a", "1/2/2006 3:04:05 PM"},    // en_US, date and time in one
		{"dd/MM/y", "02/01/2006"},                      // en_GB, fr_FR
		{"y/MM/dd", "2006/01/02"},                      // ja_JP
		{"H:mm:ss", "15:04:05"},                        // ja_JP: Go has no unpadded 24-hour field
		{"y. MM. dd.", "2006. 01. 02."},                // hu_HU
		{"HH.mm.ss", "15.04.05"},                       // da_DK
		{"HH 'h' mm 'min' ss 's'", "15 h 04 min 05 s"}, // quoted literals
		{"h 'o''clock' a", "3 o'clock PM"},             // a quote inside quoted text
		{"HH''mm", "15'04"},                            // a quote outside
		{"HH:mm:ss.SSS", "15:04:05.000"},
		{"HH:mm:ss,SS", "15:04:05,00"},
		{"HH:mm:ss z", "15:04:05 MST"},
		{"HH:mm:ss Z", "15:04:05 -0700"},
		{"HH:mm:ssXXX", "15:04:05Z07:00"},
		{"HH:mm:ss xx", "15:04:05 -0700"},
		{"EEEE, d. MMMM y", "Monday, 2. January 2006"},
		{"EEE d MMM yyyy", "Mon 2 Jan 2006"},
		{"y年M月d日", "2006年1月2日"},
		{"K:mm a", "3:04 PM"},
		{"kk:mm", "15:04"},
	} {
		got, err := GoLayout(tc.in)
		if err != nil || got != tc.want {
			t.Errorf("GoLayout(%q) = %q, %v; want %q", tc.in, got, err, tc.want)
		}
	}
	for _, in := range []string{
		"", strings.Repeat("d", maxPattern+1),
		"G y",           // an era
		"QQQ y",         // a quarter
		"w",             // a week
		"Bh:mm:ss",      // zh_TW's flexible day period
		"'unterminated", // an open quote
		"dd'1'MM",       // a digit Go would read as a field
		"dd_MM",         // Go's "_2"
		"'Jan' d",       // a word Go would read as a field
		"MMM'u'",        // a lower-case letter after a month name
		"Ms",            // fields that run together: "15" reads as an hour
		"MMMMM",         // a narrow month name
		"ss SSS",        // a fraction without its full stop
	} {
		if got, err := GoLayout(in); !errors.Is(err, errNotConvertible) {
			t.Errorf("GoLayout(%q) = %q, %v; want refused", in, got, err)
		}
	}
}

// TestFromPatterns: a date and a time pattern make a layout; one that does not convert, or lacks
// what a date or time needs, falls back to ISO 8601's and says why.
func TestFromPatterns(t *testing.T) {
	l, err := FromPatterns("dd.MM.yy", "HH:mm:ss")
	if err != nil || l != (model.TimeLayout{Date: "02.01.2006", Time: "15:04:05"}) {
		t.Fatalf("FromPatterns = %+v, %v", l, err)
	}
	at := time.Date(2026, 9, 10, 17, 41, 1, 0, time.UTC)
	if got := l.In(time.UTC).Format(at); got != "10.09.2026 17:41:01" {
		t.Errorf("the owner's format: %q", got) // D-31's example
	}
	l, err = FromPatterns("dd.MM", "G HH:mm")
	if err == nil || l != (model.TimeLayout{}) || !strings.Contains(err.Error(), "no day, month and year") ||
		!strings.Contains(err.Error(), "time pattern") {
		t.Errorf("FromPatterns of bad patterns = %+v, %v", l, err)
	}
	if l, err := FromPatterns("y-MM-dd", "mm:ss"); err == nil || l.Date != "2006-01-02" || l.Time != "" {
		t.Errorf("a time without hours = %+v, %v", l, err)
	}
}

// TestEnvironment: the POSIX locale's variables in POSIX's order, the table's formats by territory
// and else by language, ISO 8601 for C, POSIX, an unknown locale or none. Never the developer's
// own environment.
func TestEnvironment(t *testing.T) {
	at := time.Date(2026, 9, 10, 17, 41, 1, 0, time.UTC)
	for _, tc := range []struct {
		env  map[string]string
		want string
	}{
		{map[string]string{"LANG": "de_DE.UTF-8"}, "10.09.2026 17:41:01"},
		{map[string]string{"LANG": "en_US.UTF-8"}, "9/10/2026 5:41:01 PM"},
		{map[string]string{"LANG": "en_US.UTF-8", "LC_TIME": "en_GB.UTF-8"}, "10/09/2026 17:41:01"},
		{map[string]string{"LC_ALL": "ja_JP.UTF-8", "LC_TIME": "en_GB.UTF-8", "LANG": "de_DE"}, "2026/09/10 17:41:01"},
		{map[string]string{"LANG": "de_AT@euro"}, "10.09.2026 17:41:01"},
		{map[string]string{"LANG": "de_LU.UTF-8"}, "10.09.2026 17:41:01"}, // by its language
		{map[string]string{"LANG": "fr-CA"}, "2026-09-10 17:41:01"},
		{map[string]string{"LANG": "hu_HU.UTF-8"}, "2026. 09. 10. 17:41:01"},
		{map[string]string{"LANG": "C"}, "2026-09-10 17:41:01"},
		{map[string]string{"LC_ALL": "POSIX"}, "2026-09-10 17:41:01"},
		{map[string]string{"LANG": "C.UTF-8"}, "2026-09-10 17:41:01"},
		{map[string]string{"LANG": "ko_KR.UTF-8"}, "2026-09-10 17:41:01"}, // left out of the table
		{map[string]string{}, "2026-09-10 17:41:01"},
	} {
		f := Environment(func(k string) string { return tc.env[k] })
		if got := f.Layout.In(time.UTC).Format(at); got != tc.want {
			t.Errorf("%v: %q (%s), want %q", tc.env, got, f.Source, tc.want)
		}
		if f.Source == "" {
			t.Errorf("%v: no source", tc.env)
		}
	}
}

// TestTableConverts: every locale of the table converts without a fallback.
func TestTableConverts(t *testing.T) {
	for name, p := range cldr {
		if _, err := FromPatterns(p.date, p.time); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}
