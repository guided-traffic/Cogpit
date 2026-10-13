package locale

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/guided-traffic/cogpit/internal/model"
)

// maxPattern bounds the patterns GoLayout converts: CLDR's are a few characters long, a user's own
// macOS format rarely more than 30.
const maxPattern = 128

var errNotConvertible = errors.New("not convertible to a Go layout")

// GoLayout converts a date or time pattern of Unicode's (UTS #35: macOS's, ICU's, CLDR's) to a Go
// layout (package time): "dd.MM.yy" → "02.01.2006", "h:mm:ss a" → "3:04:05 PM". A year of any
// length is written with four digits. Quoted text ('h') is literal, and two quotes in a row are
// one; narrow and non-breaking spaces become spaces. The fields Go can write convert, some approximated: H and k
// write 24-hour hours with two digits (Go has no unpadded 24-hour field), K writes 12-hour hours,
// z, v and V the zone's abbreviation, Z, x and X its offset, and month and weekday names and a, the
// day period, are written in English. A pattern with another field (an era, a quarter, a week, a
// day period other than AM and PM), literal text Go would read as a field (digits, "Jan", "Mon",
// "MST", "PM"), an unterminated quote, or more than maxPattern bytes is refused.
func GoLayout(pattern string) (string, error) {
	if pattern == "" || len(pattern) > maxPattern {
		return "", fmt.Errorf("%w: empty or longer than %d bytes", errNotConvertible, maxPattern)
	}
	var b strings.Builder
	var want []string // the elements written, in order: what the layout must read back as
	rs := []rune(pattern)
	prev := "" // the last field written
	for i := 0; i < len(rs); {
		r := rs[i]
		switch {
		case r == '\'':
			text, next, err := quoted(rs, i)
			if err != nil {
				return "", err
			}
			if err := literal(&b, text, prev); err != nil {
				return "", err
			}
			prev, i = "", next
		case r < unicode.MaxASCII && unicode.IsLetter(r):
			j := i
			for j < len(rs) && rs[j] == r {
				j++
			}
			field, elem, err := goField(r, j-i, b.String())
			if err != nil {
				return "", err
			}
			b.WriteString(field)
			want = append(want, elem)
			prev, i = field, j
		default:
			if err := literal(&b, string(r), prev); err != nil {
				return "", err
			}
			prev, i = "", i+1
		}
	}
	out := b.String()
	if got := model.LayoutElements(out); !slices.Equal(got, want) {
		// Fields that run into each other read as other fields ("1" and "5" as "15").
		return "", fmt.Errorf("%w: %q reads as %v, not %v", errNotConvertible, out, got, want)
	}
	return out, nil
}

// quoted reads the quoted literal at rs[i] (a quote): its text and the index after it. Two quotes
// in a row are a quote, inside quoted text or outside.
func quoted(rs []rune, i int) (string, int, error) {
	if i+1 < len(rs) && rs[i+1] == '\'' {
		return "'", i + 2, nil
	}
	var b strings.Builder
	for j := i + 1; j < len(rs); j++ {
		if rs[j] != '\'' {
			b.WriteRune(rs[j])
			continue
		}
		if j+1 < len(rs) && rs[j+1] == '\'' {
			b.WriteRune('\'')
			j++
			continue
		}
		return b.String(), j + 1, nil
	}
	return "", 0, fmt.Errorf("%w: unterminated quote", errNotConvertible)
}

// literal writes literal text, refusing what Go would read as a field: digits, an underscore
// (Go's "_2"), "Jan", "Mon", "MST", "PM" or "pm", and a lower-case letter right after a month or
// weekday name (Go would not read "Janu" as "Jan").
func literal(b *strings.Builder, text, prev string) error {
	text = strings.Map(func(r rune) rune {
		if r == '\u202f' || r == '\u00a0' { // CLDR's narrow and non-breaking spaces ("h:mm:ss\u202fa")
			return ' '
		}
		return r
	}, text)
	if strings.ContainsAny(text, "0123456789_") {
		return fmt.Errorf("%w: literal %q holds digits or an underscore", errNotConvertible, text)
	}
	for _, w := range []string{"Jan", "Mon", "MST", "PM", "pm"} {
		if strings.Contains(text, w) {
			return fmt.Errorf("%w: literal %q reads as a Go field", errNotConvertible, text)
		}
	}
	if (prev == "Jan" || prev == "Mon") && text != "" && text[0] >= 'a' && text[0] <= 'z' {
		return fmt.Errorf("%w: literal %q after a name", errNotConvertible, text)
	}
	b.WriteString(text)
	return nil
}

// goField returns the Go layout field of a run of n pattern letters r, and the element it is
// (model.LayoutElements' names). before is the layout so far: a fraction needs a separator.
func goField(r rune, n int, before string) (string, string, error) {
	pick := func(elem string, fields ...string) (string, string, error) {
		if n > len(fields) || fields[n-1] == "" {
			return "", "", fmt.Errorf("%w: %d × %q", errNotConvertible, n, r)
		}
		return fields[n-1], elem, nil
	}
	switch r {
	case 'y', 'u', 'Y': // the year, of any length, with four digits (yy expanded)
		return "2006", "year", nil
	case 'M', 'L':
		return pick("month", "1", "01", "Jan", "January")
	case 'd':
		return pick("day", "2", "02")
	case 'E':
		return pick("weekday", "Mon", "Mon", "Mon", "Monday")
	case 'c', 'e': // numeric below 3
		return pick("weekday", "", "", "Mon", "Monday")
	case 'a':
		return pick("ampm", "PM", "PM", "PM")
	case 'h', 'K':
		return pick("hour", "3", "03")
	case 'H', 'k':
		return pick("hour", "15", "15")
	case 'm':
		return pick("minute", "4", "04")
	case 's':
		return pick("second", "5", "05")
	case 'S': // fractional seconds: Go writes them after a full stop or a comma
		if n > 9 || !strings.HasSuffix(before, ".") && !strings.HasSuffix(before, ",") {
			return "", "", fmt.Errorf("%w: fractional seconds without a separator", errNotConvertible)
		}
		return strings.Repeat("0", n), "fraction", nil
	case 'z', 'v', 'V', 'O':
		return pick("zone", "MST", "MST", "MST", "MST")
	case 'Z':
		return pick("zone", "-0700", "-0700", "-0700", "-07:00", "Z07:00")
	case 'x':
		return pick("zone", "-07", "-0700", "-07:00", "-0700", "-07:00")
	case 'X':
		return pick("zone", "Z07", "Z0700", "Z07:00", "Z0700", "Z07:00")
	}
	return "", "", fmt.Errorf("%w: field %q", errNotConvertible, r)
}
