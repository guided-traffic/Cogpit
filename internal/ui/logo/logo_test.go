package logo

import (
	"image"
	"image/color"
	"math"
	"os"
	"regexp"
	"strconv"
	"testing"
)

var (
	plate  = color.NRGBA{R: 0x00, G: 0x2b, B: 0x36, A: 0xff}
	frame  = color.NRGBA{R: 0x93, G: 0xa1, B: 0xa1, A: 0xff}
	accent = color.NRGBA{R: 0x2a, G: 0xa1, B: 0x98, A: 0xff}
)

// at is the pixel at a box point of an image drawn at 4 pixels to a unit.
func at(img *image.NRGBA, x, y float64) color.NRGBA { return img.NRGBAAt(int(x*4), int(y*4)) }

// onC is the point at an angle (degrees, screen direction) on the C's centre line, out units further.
func onC(deg, out float64) [2]float64 {
	a := deg * math.Pi / 180
	return [2]float64{50 + (cR+out)*math.Cos(a), 50 + (cR+out)*math.Sin(a)}
}

func near(a, b color.NRGBA, tol int) bool {
	d := func(x, y uint8) int { return max(int(x)-int(y), int(y)-int(x)) }
	return d(a.R, b.R) <= tol && d(a.G, b.G) <= tol && d(a.B, b.B) <= tol && d(a.A, b.A) <= tol
}

func TestIconGeometry(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 400, 400))
	Draw(img, Icon(plate), Colors{Frame: frame, Accent: accent}, 4, 0, 0)
	for _, c := range []struct {
		what string
		x, y float64
		want color.NRGBA
	}{
		{"outside the plate's corner", 3, 3, color.NRGBA{}},
		{"the plate", 12, 12, plate},
		{"the C at 112.5 degrees", onC(112.5, 0)[0], onC(112.5, 0)[1], frame}, // between two spokes
		{"the C at 157.5 degrees", onC(157.5, 0)[0], onC(157.5, 0)[1], frame},
		{"the C at 247.5 degrees", onC(247.5, 0)[0], onC(247.5, 0)[1], frame},
		{"inside the C", 56, 33, plate}, // between the C and the inner ring, off every spoke
		{"the inner ring", 50, 38, accent},
		{"a spoke", 50, 30, accent}, // up
		{"a handle", 50, 13, frame}, // up, outside the C
		{"the gap between the C and a handle", 50, 18, plate},
		{"the needle in the opening", 80, 50, accent},
		{"the opening above the needle", 80, 45, plate},
		{"the hub", 50, 50, accent},
		{"no spoke to the right but the needle", 70, 56, plate},
	} {
		if got := at(img, c.x, c.y); !near(got, c.want, 3) {
			t.Errorf("%s at (%v, %v): %v, want %v", c.what, c.x, c.y, got, c.want)
		}
	}
	// The C's ends: round caps at ±37.5° where the opening begins; none past them.
	if got := at(img, 50+28*0.7934+1.5, 50-28*0.6088+1); near(got, plate, 3) {
		t.Errorf("the C's upper end is missing: %v", got)
	}
	if got := at(img, 50+28, 50+9); !near(got, plate, 3) { // right of the C, 18° below the axis: in the opening
		t.Errorf("the opening is covered: %v", got)
	}
}

func TestMarkHasNoPlate(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 400, 400))
	Draw(img, Mark(), Colors{Frame: frame, Accent: accent}, 4, 0, 0)
	if got := at(img, 5, 5); got.A != 0 {
		t.Errorf("a corner of the mark is %v, want transparent", got)
	}
	if got := at(img, onC(157.5, 0)[0], onC(157.5, 0)[1]); !near(got, frame, 3) {
		t.Errorf("the C is %v, want the frame ink", got)
	}
	if got := at(img, 50, 50); !near(got, accent, 3) {
		t.Errorf("the hub is %v, want the accent", got)
	}
	// Heavier lines: a point 6 units from the C's centre line is still on the heavy C (stroke 14).
	if got := at(img, onC(157.5, 6)[0], onC(157.5, 6)[1]); !near(got, frame, 3) {
		t.Errorf("the heavy C does not reach 6 units out: %v", got)
	}
}

// TestStylesMatchTheReferenceSVGs: the stroke widths of the reference SVGs are the styles' (the
// geometry itself is in the constants; a changed reference without a changed style fails here).
func TestStylesMatchTheReferenceSVGs(t *testing.T) {
	widths := func(file string) map[float64]bool {
		b, err := os.ReadFile("../../../docs/developer/assets/" + file)
		if err != nil {
			t.Fatal(err)
		}
		set := map[float64]bool{}
		for _, m := range regexp.MustCompile(`stroke-width="([0-9.]+)"`).FindAllStringSubmatch(string(b), -1) {
			v, _ := strconv.ParseFloat(m[1], 64)
			set[v] = true
		}
		return set
	}
	want := func(s Style) map[float64]bool {
		return map[float64]bool{s.OuterRing: true, s.C: true, s.InnerRing: true, s.Spoke: true, s.Handle: true, s.Needle: true}
	}
	for file, st := range map[string]Style{"cogpit-icon.svg": Icon(plate), "cogpit-mark-dark.svg": Mark(), "cogpit-mark-light.svg": Mark()} {
		got, exp := widths(file), want(st)
		for w := range exp {
			if !got[w] {
				t.Errorf("%s: no stroke of width %v", file, w)
			}
		}
		for w := range got {
			if w != 1 && !exp[w] { // 1 is the plate's edge line
				t.Errorf("%s: stroke width %v is in no style", file, w)
			}
		}
	}
}
