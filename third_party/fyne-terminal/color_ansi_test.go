package terminal

import (
	"image/color"
	"reflect"
	"testing"
)

// TestSetANSIColors (Cogpit patch): the replaced colours serve SGR 31 and 91 and the 256-colour
// palette's entries 1 and 9.
func TestSetANSIColors(t *testing.T) {
	oldBasic, oldBright := basicColors, brightColors
	defer func() { basicColors, brightColors = oldBasic, oldBright }()

	var c [16]color.Color
	for i := range c {
		c[i] = color.NRGBA{R: uint8(i), A: 0xff}
	}
	SetANSIColors(c)
	for seq, want := range map[string]color.Color{
		esc("[31m"): c[1], esc("[91m"): c[9], esc("[38;5;1m"): c[1], esc("[38;5;9m"): c[9],
	} {
		term := New()
		term.handleOutput([]byte(seq))
		if !reflect.DeepEqual(term.currentFG, want) {
			t.Errorf("%q: foreground %v, want %v", seq, term.currentFG, want)
		}
	}
}
