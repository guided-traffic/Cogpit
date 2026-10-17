package engine

import (
	"os"
	"path/filepath"
	"testing"
)

// A cluster's accent colour (D-80): the icon colour's choices, in any case, empty for none;
// anything else refused, and dropped from a file that holds it; kept across a restart, shown in
// the UI state and the settings' view.
func TestAccentColorKept(t *testing.T) {
	for in, want := range map[string]string{"red": "red", " Green ": "green", "#FF8800": "#ff8800", "": "", "  ": ""} {
		if got, err := ParseAccentColor(in); err != nil || got != want {
			t.Errorf("ParseAccentColor(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"pink", "#ff88", "#gg0000", "accent", "red;"} {
		if _, err := ParseAccentColor(in); err == nil {
			t.Errorf("ParseAccentColor(%q) took it", in)
		}
	}
	if err := (ClusterSettings{AccentColor: "pink"}).Check(); err == nil {
		t.Error("Check took the accent colour pink")
	}

	dir := filepath.Join(t.TempDir(), "Cogpit")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	doc := `{"version":1,"clusters":{"prod":{"accentColor":"red"},"odd":{"name":"Odd","accentColor":"pink"}}}`
	if err := os.WriteFile(filepath.Join(dir, ClusterSettingsFile), []byte(doc), 0o600); err != nil {
		t.Fatal(err)
	}
	s := New(Options{ClusterSettings: NewClusterSettingsStore(dir)})
	defer s.Close()
	u := s.UIState()
	if u.ClusterSettings["prod"].AccentColor != "red" || u.ClusterSettings["odd"].AccentColor != "" || u.ClusterSettings["odd"].Name != "Odd" {
		t.Errorf("read from the file: %+v", u.ClusterSettings)
	}
	if v, _ := s.ClusterSettingsView("prod"); v.AccentColor != "red" {
		t.Errorf("prod's view: %+v", v)
	}
	if _, err := s.ChangeClusterSettings("prod", func(c *ClusterSettings) error { c.AccentColor = "#00ff00"; return nil }); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ChangeClusterSettings("prod", func(c *ClusterSettings) error { c.AccentColor = "teal"; return nil }); err == nil {
		t.Error("a change to an accent colour Check refuses was kept")
	}
	s2 := New(Options{ClusterSettings: NewClusterSettingsStore(dir)})
	defer s2.Close()
	if cs, _ := s2.ClusterSettingsOf("prod"); cs.AccentColor != "#00ff00" {
		t.Errorf("after a restart: %+v", cs)
	}
}
