package termview

import (
	"io"
	"strconv"
	"unicode/utf8"
)

// Bounds of the escape filter.
const (
	filterReadSize = 32 << 10 // what one Read of the session asks for; the widget reads 32 KiB too
	maxCSIParams   = 128      // parameter bytes of one control sequence; a longer one is dropped
	maxCSINumber   = 9999     // larger numeric parameters are clamped to this
	maxStringLen   = 1 << 20  // an unterminated OSC/DCS/APC/PM/SOS string ends after this many bytes
)

const (
	esc = 0x1b
	can = 0x18 // CAN and SUB abort an escape sequence (VT100)
	sub = 0x1a
)

// csiFinals are the control sequences fyne-io/terminal (91063bb29d69) implements and that are safe
// to pass on. Left out: media copy (i), whose "print" mode makes the widget buffer all further
// output without a bound, and window operations (t). Everything else the widget ignores anyway.
const csiFinals = "@ABCDdHfGhLlmJKPrsSucXMTb"

// filter states.
const (
	stGround    = iota
	stEsc       // after ESC
	stNF        // ESC followed by intermediate bytes (charset designation and the like)
	stCSI       // after ESC [
	stString    // inside OSC, DCS, APC, PM or SOS: dropped up to its terminator
	stStringEsc // ESC inside such a string
)

// escFilter sits between a terminal session and the fyne-io/terminal widget. Session output is
// untrusted (docs/security/trust-boundaries.md), and the widget, upstream at 91063bb29d69 and
// 46cd12724ff3, acts on sequences a container must not be able to send:
//
//   - OSC 7 ("current directory") calls os.Chdir in Cogpit's own process; OSC 0/1/2 set the
//     title. All OSC strings are dropped, and with them OSC 52 (clipboard write), which the widget
//     does not implement today but a newer one might. DCS, APC, PM and SOS strings are dropped
//     too; upstream buffers them, and OSC strings, without a bound and in quadratic time.
//   - Some sequences panic the widget inside fyne.Do, on the main goroutine, which ends the app:
//     a control sequence whose final character is not ASCII ("ESC [ é"), set/reset mode without
//     the "?" marker ("ESC [ h"), delete-character with the cursor past the row's last cell, and
//     reverse index ("ESC M") before the screen's rows exist (on the reader goroutine).
//   - Large counts (repeat, insert/delete lines, scroll) make the widget loop for as long as the
//     count.
//
// Cogpit builds against a patched copy (third_party/fyne-terminal, PATCHES.md) that survives all
// of these and is fuzzed on its own (FuzzHandleOutput there, FuzzRawWidget here); the filter stays
// in front of it for what nobody has found yet. It parses the stream (ECMA-48, as xterm does) and
// passes on only plain text, the C0 controls a terminal needs, and a known set of escape and
// control sequences, re-emitted in a canonical form with numbers clamped. Invalid UTF-8 is dropped
// (the widget drops it too), and every Read ends on a rune boundary, because the widget decodes
// each Read on its own and would lose a character split across two.
//
// Insert-character ("ESC [ @") is passed on although it panicked upstream's widget when the cursor
// was past the row's last cell or on a row nothing was written to: whether that is the case
// depends on the screen's state, which the filter does not track, and dropping the sequence would
// break line editing in shells. The patched copy fixes it.
type escFilter struct {
	src  io.Reader
	in   []byte
	work []byte // carry plus the new input, while a rune is split across reads
	out  []byte // filtered output not yet returned
	off  int
	err  error

	carry  []byte // the start of a rune whose remaining bytes have not arrived yet
	state  int
	seq    []byte // parameters of the current control sequence, or intermediates of an nF escape
	seqBad bool   // the current sequence is too long or malformed: drop it at its end
	strLen int
}

func newEscFilter(src io.Reader) *escFilter {
	return &escFilter{src: src, in: make([]byte, filterReadSize)}
}

// Read implements io.Reader. Errors of the source are passed on once its output is consumed.
func (f *escFilter) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	for f.off == len(f.out) {
		if f.err != nil {
			return 0, f.err
		}
		f.out, f.off = f.out[:0], 0
		n, err := f.src.Read(f.in)
		f.feed(f.in[:n], err != nil)
		if err != nil {
			f.err = err
		}
	}
	n := copy(p, f.out[f.off:])
	if end := f.off + n; end < len(f.out) {
		k := end
		for k > f.off && !utf8.RuneStart(f.out[k]) {
			k--
		}
		if k > f.off { // else p is shorter than one rune: hand out the bytes as they are
			n = k - f.off
		}
	}
	f.off += n
	return n, nil
}

// feed runs one chunk of input through the state machine.
func (f *escFilter) feed(chunk []byte, eof bool) {
	data := chunk
	if len(f.carry) > 0 {
		f.work = append(append(f.work[:0], f.carry...), chunk...)
		f.carry = f.carry[:0]
		data = f.work
	}
	for i := 0; i < len(data); {
		b := data[i]
		if b < utf8.RuneSelf {
			f.step(rune(b), data[i:i+1])
			i++
			continue
		}
		if !utf8.FullRune(data[i:]) {
			if !eof {
				f.carry = append(f.carry[:0], data[i:]...)
			}
			return
		}
		r, size := utf8.DecodeRune(data[i:])
		if r == utf8.RuneError && size == 1 {
			i++ // not UTF-8: dropped, in every state
			continue
		}
		f.step(r, data[i:i+size])
		i += size
	}
}

// passC0 reports whether a C0 control is passed on: bell, backspace, tab, line feeds, carriage
// return and the G0/G1 shifts. The others the widget would print as cells.
func passC0(r rune) bool {
	return r >= 0x07 && r <= 0x0f
}

func (f *escFilter) emit(b ...byte) { f.out = append(f.out, b...) }

func (f *escFilter) step(r rune, raw []byte) {
	switch f.state {
	case stGround:
		switch {
		case r == esc:
			f.state = stEsc
		case r < 0x20:
			if passC0(r) {
				f.emit(raw...)
			}
		case r == 0x7f, r >= 0x80 && r < 0xa0: // DEL and the C1 controls
		default:
			f.emit(raw...)
		}

	case stEsc:
		f.escape(r)

	case stNF:
		switch {
		case r >= 0x20 && r <= 0x2f:
			if len(f.seq) < 4 {
				f.seq = append(f.seq, byte(r))
			} else {
				f.seqBad = true
			}
		case r >= 0x30 && r <= 0x7e:
			// The widget implements G0/G1 designation only ("ESC ( 0", "ESC ) B").
			if !f.seqBad && len(f.seq) == 1 && (f.seq[0] == '(' || f.seq[0] == ')') {
				f.emit(esc, f.seq[0], byte(r))
			}
			f.state = stGround
		default:
			f.abort(r, raw)
		}

	case stCSI:
		switch {
		case r >= 0x30 && r <= 0x3f, r >= 0x20 && r <= 0x2f: // parameters, intermediates
			if len(f.seq) < maxCSIParams {
				f.seq = append(f.seq, byte(r))
			} else {
				f.seqBad = true
			}
		case r >= 0x40 && r <= 0x7e:
			if !f.seqBad {
				f.emitCSI(byte(r))
			}
			f.state = stGround
		case r < 0x20 && r != esc && r != can && r != sub:
			// C0 controls inside a control sequence take effect and the sequence goes on.
			// Passed on now, they reach the widget before the sequence, outside of it.
			if passC0(r) {
				f.emit(raw...)
			}
		case r == 0x7f:
		default:
			f.abort(r, raw)
		}

	case stString:
		f.strLen += len(raw)
		switch {
		case r == 0x07, r == can, r == sub:
			f.state = stGround
		case r == esc:
			f.state = stStringEsc
		case f.strLen > maxStringLen:
			f.state = stGround
		}

	case stStringEsc:
		if r == '\\' { // ST
			f.state = stGround
			return
		}
		f.state = stEsc
		f.escape(r)
	}
}

// abort ends the current sequence without passing it on and handles r as if it came after it.
// A sequence is passed on only once it is complete, so the widget never sees part of one.
func (f *escFilter) abort(r rune, raw []byte) {
	switch r {
	case esc:
		f.state = stEsc
	case can, sub:
		f.state = stGround
	default:
		f.state = stGround
		f.step(r, raw)
	}
}

// escape handles the character after ESC.
func (f *escFilter) escape(r rune) {
	switch {
	case r == esc: // ESC ESC: the first one is dropped
	case r == can || r == sub:
		f.state = stGround
	case r == '[':
		f.state, f.seq, f.seqBad = stCSI, f.seq[:0], false
	case r == ']' || r == 'P' || r == '_' || r == '^' || r == 'X': // OSC, DCS, APC, PM, SOS
		f.state, f.strLen = stString, 0
	case r >= 0x20 && r <= 0x2f:
		f.state, f.seq, f.seqBad = stNF, append(f.seq[:0], byte(r)), false
	case r == '7' || r == '8' || r == 'D' || r == '=' || r == '>':
		// Save and restore cursor, index, keypad modes.
		f.emit(esc, byte(r))
		f.state = stGround
	case r == 'M':
		// Reverse index. The widget implements it as "scroll the region down one line" and
		// indexes rows that may not exist yet; "ESC [ T" does the same through bounds-checked
		// calls.
		f.emit(esc, '[', 'T')
		f.state = stGround
	default:
		// Other escapes the widget ignores; a control character or non-ASCII rune after ESC is
		// swallowed with it, as the widget does.
		f.state = stGround
	}
}

// emitCSI passes on a complete control sequence in canonical form, or drops it.
func (f *escFilter) emitCSI(final byte) {
	params := f.seq
	var marker byte
	if len(params) > 0 && (params[0] == '?' || params[0] == '>' || params[0] == '=') {
		marker, params = params[0], params[1:]
	}
	for _, b := range params {
		if (b < '0' || b > '9') && b != ';' {
			return // intermediates, sub-parameters (':'), misplaced markers: not for this widget
		}
	}
	if !containsByte(csiFinals, final) {
		return
	}
	switch final {
	case 'h', 'l':
		// The widget implements DEC private modes only, and indexes past the end of an
		// unmarked one.
		if marker != '?' {
			return
		}
	case 'c':
		// Device attributes: any marker is fine.
	default:
		if marker != 0 {
			return
		}
	}
	if final == 'P' {
		// Delete characters slices the row up to the cursor and panics when the cursor is past
		// the row's last cell. Erasing the character under the cursor first extends the row to
		// the cursor and changes nothing visible: delete removes that character anyway.
		f.emit(esc, '[', 'X')
	}
	f.emit(esc, '[')
	if marker != 0 {
		f.emit(marker)
	}
	f.out = appendClamped(f.out, params)
	f.emit(final)
}

// appendClamped appends params with every number above maxCSINumber replaced by it.
func appendClamped(dst, params []byte) []byte {
	for i := 0; i < len(params); {
		if params[i] == ';' {
			dst = append(dst, ';')
			i++
			continue
		}
		j := i
		for j < len(params) && params[j] != ';' {
			j++
		}
		if n, err := strconv.Atoi(string(params[i:j])); err != nil || n > maxCSINumber {
			dst = strconv.AppendInt(dst, maxCSINumber, 10)
		} else {
			dst = append(dst, params[i:j]...)
		}
		i = j
	}
	return dst
}

func containsByte(s string, b byte) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return true
		}
	}
	return false
}
