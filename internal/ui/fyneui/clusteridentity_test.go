package fyneui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// Every icon colour the engine offers has a token in both variants, and a cluster's initials in it
// read at 3:1 where they are drawn: the hotbar (the header's surface), the navigator (the
// sidebar's), the picker's list (the input background) and the settings page (the background).
func TestIdentityColorsContrast(t *testing.T) {
	for name, p := range map[string]*palette{"dark": &darkPalette, "light": &lightPalette} {
		for _, c := range engine.IconColors {
			ink, ok := p.fyne[identityColorName(c)]
			if !ok {
				t.Errorf("%s: no token for %s", name, c)
				continue
			}
			for _, bg := range []fyne.ThemeColorName{theme.ColorNameHeaderBackground, colorSidebar, theme.ColorNameInputBackground, theme.ColorNameBackground} {
				if got := contrastRatio(ink, p.fyne[bg]); got < 3.0 {
					t.Errorf("%s: %s on %s is %.2f:1, want 3:1", name, c, bg, got)
				}
			}
		}
	}
}

// A cluster's identity: its display name or context name, initials of either, and its colour.
func TestIdentityFromTheClusterSettings(t *testing.T) {
	for in, want := range map[string]string{"kind-cogpit": "KC", "Dev cluster": "DC", "prod": "PR", "  Two  Words ": "TW"} {
		if got := nameInitials(in); got != want {
			t.Errorf("nameInitials(%q) = %q, want %q", in, got, want)
		}
	}
	f := newFakeEngine(t)
	f.ui.ClusterSettings = map[string]engine.ClusterSettings{"kind-dev [.kube/config]": {Name: "Dev cluster", IconColor: "green"}}
	sh, _, q := newTestShell(t, f, Options{})
	q.drain()
	id := sh.identity("kind-dev [.kube/config]")
	if id.name != "Dev cluster" || id.initials != "DC" || id.ink != identityColorName("green") || id.image != nil {
		t.Errorf("set: %+v", id)
	}
	if id := sh.identity("prod-eu [.kube/prod.yaml]"); id.name != "prod-eu" || id.initials != "PE" || id.ink != colorAccent {
		t.Errorf("default: %+v", id)
	}
	// A fake without icons draws the initials even when a file is named.
	f.ui.ClusterSettings["kind-dev [.kube/config]"] = engine.ClusterSettings{Icon: "x.png"}
	sh.ui.ClusterSettings = f.ui.ClusterSettings
	if id := sh.identity("kind-dev [.kube/config]"); id.image != nil || id.initials != "KD" {
		t.Errorf("icon without a source: %+v", id)
	}
}
