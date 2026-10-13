package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
)

// Accessible namespaces are kept per context: trimmed, sorted and without duplicates, read back by
// the next start, refused when a name is no namespace's or there are too many, and dropped one by
// one from a file that holds bad names.
func TestAccessibleNamespacesKept(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Cogpit")
	s := New(Options{ClusterSettings: NewClusterSettingsStore(dir)})
	set := func(names ...string) error {
		_, err := s.ChangeClusterSettings(devCtx, func(c *ClusterSettings) error { c.AccessibleNamespaces = names; return nil })
		return err
	}
	if err := set(" team-b ", "team-a", "team-b", ""); err != nil {
		t.Fatal(err)
	}
	want := []string{"team-a", "team-b"}
	if got := s.UIState().ClusterSettings[devCtx].AccessibleNamespaces; !slices.Equal(got, want) {
		t.Errorf("UI state %q, want %q", got, want)
	}
	if v, _ := s.ClusterSettingsView(devCtx); !slices.Equal(v.AccessibleNamespaces, want) {
		t.Errorf("view %q", v.AccessibleNamespaces)
	}
	for name, names := range map[string][]string{
		"upper case":  {"Team-A"},
		"underscore":  {"team_a"},
		"too long":    {fmt.Sprintf("%064d", 0)},
		"a dot":       {"team.a"},
		"too many":    manyNamespaces(maxAccessibleNamespaces + 1),
		"a path":      {"../etc"},
		"a selector?": {"team-a,team-b"},
	} {
		if err := set(names...); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	if got := s.UIState().ClusterSettings[devCtx].AccessibleNamespaces; !slices.Equal(got, want) {
		t.Errorf("a refused change changed the list: %q", got)
	}
	if err := set(manyNamespaces(maxAccessibleNamespaces)...); err != nil {
		t.Errorf("%d namespaces refused: %v", maxAccessibleNamespaces, err)
	}
	if err := set(want...); err != nil {
		t.Fatal(err)
	}
	s.Close()

	s2 := New(Options{ClusterSettings: NewClusterSettingsStore(dir)})
	if cs, _ := s2.ClusterSettingsOf(devCtx); !slices.Equal(cs.AccessibleNamespaces, want) {
		t.Errorf("after a restart: %q", cs.AccessibleNamespaces)
	}
	if _, err := s2.ChangeClusterSettings(devCtx, func(c *ClusterSettings) error { c.AccessibleNamespaces = nil; return nil }); err != nil {
		t.Fatal(err)
	}
	if _, ok := s2.UIState().ClusterSettings[devCtx]; ok {
		t.Error("a context without accessible namespaces or any other setting kept an entry")
	}
	s2.Close()

	doc := `{"version":1,"clusters":{"dev":{"accessibleNamespaces":["team-b","Bad_Name","team-a","team-a",""]}}}`
	if err := os.WriteFile(filepath.Join(dir, ClusterSettingsFile), []byte(doc), 0o600); err != nil {
		t.Fatal(err)
	}
	s3 := New(Options{ClusterSettings: NewClusterSettingsStore(dir)})
	defer s3.Close()
	if got := s3.UIState().ClusterSettings["dev"].AccessibleNamespaces; !slices.Equal(got, want) {
		t.Errorf("normalized from the file: %q, want %q", got, want)
	}
}

func manyNamespaces(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("ns-%03d", i)
	}
	return out
}

// Equal compares every field: a field added later and forgotten there would make a change of it
// invisible to the window (maps.EqualFunc in the shell).
func TestClusterSettingsEqualComparesEveryField(t *testing.T) {
	typ := reflect.TypeFor[ClusterSettings]()
	for i := range typ.NumField() {
		var cs ClusterSettings
		f := reflect.ValueOf(&cs).Elem().Field(i)
		switch f.Kind() {
		case reflect.String:
			f.SetString("x")
		case reflect.Slice:
			f.Set(reflect.ValueOf([]string{"x"}))
		case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
			f.SetUint(1)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			f.SetInt(1)
		case reflect.Bool:
			f.SetBool(true)
		default:
			t.Fatalf("field %s: a kind the test does not set (%s)", typ.Field(i).Name, f.Kind())
		}
		if cs.Equal(ClusterSettings{}) || cs.IsZero() {
			t.Errorf("field %s: Equal does not compare it", typ.Field(i).Name)
		}
	}
	if !(ClusterSettings{AccessibleNamespaces: []string{}}).IsZero() {
		t.Error("an empty list is not the default")
	}
}
