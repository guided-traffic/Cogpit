package engine

import (
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/util/validation"
)

// TestCheckDataKeyIsTheServers: a drawer's added key is refused exactly where the API server refuses
// it (validation.IsConfigMapKey), with a message that says why.
func TestCheckDataKeyIsTheServers(t *testing.T) {
	for _, key := range []string{
		"a", "A_b-c.d", "config.yaml", ".env", "..data", "..", ".", "a..b", "with space", "slash/x", "colon:x", "ümlaut", "",
		strings.Repeat("k", 253), strings.Repeat("k", 254), "tab\tkey", "new\nline", "-leading", "trailing-", "_",
	} {
		err := CheckDataKey(key)
		if server := len(validation.IsConfigMapKey(key)) == 0; server != (err == nil) {
			t.Errorf("%q: Cogpit says %v, the server's rule valid=%v", key, err, server)
		}
	}
	for key, want := range map[string]string{
		"":                       "needs a name",
		"a b":                    "only letters, digits",
		"..x":                    "does not start with \"..\"",
		strings.Repeat("x", 300): "at most 253 characters",
	} {
		if err := CheckDataKey(key); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%.20q: %v, want %q", key, err, want)
		}
	}
}

// TestDataKeysDetail: the confirmation's detail names the keys removed, added and changed, sorted,
// capped, never a value.
func TestDataKeysDetail(t *testing.T) {
	for _, c := range []struct {
		got, want string
	}{
		{DataKeysDetail(nil, nil), ""},
		{DataKeysDetail([]string{"b", "a"}, nil), "Adds keys a, b."},
		{DataKeysDetail([]string{"new"}, []string{"old"}), "Removes key old and adds key new."},
		{SecretSaveDetail(nil, nil, nil), "The values are not shown."},
		{SecretSaveDetail([]string{"password"}, nil, nil), "Changes key password. The values are not shown."},
		{SecretSaveDetail([]string{"c"}, []string{"b"}, []string{"a"}), "Removes key a, adds key b and changes key c. The values are not shown."},
		{DataKeysDetail(nil, strings.Split("k01 k02 k03 k04 k05 k06 k07 k08 k09 k10 k11 k12", " ")),
			"Removes keys k01, k02, k03, k04, k05, k06, k07, k08, k09, k10 and 2 more."},
	} {
		if c.got != c.want {
			t.Errorf("got %q, want %q", c.got, c.want)
		}
	}
}

// TestDataEditRebase: the rebase compares each changed and removed key's value at the base and an
// added key's absence; a removal without its value at the base never rebases.
func TestDataEditRebase(t *testing.T) {
	data := map[string]any{"a": "1", "b": "2"}
	plain := func(v any) (string, bool) { s, ok := v.(string); return s, ok }
	for _, c := range []struct {
		name string
		edit DataEdit
		from map[string]string
		want bool
	}{
		{"a change, unchanged", DataEdit{Values: map[string]string{"a": "9"}}, map[string]string{"a": "1"}, true},
		{"a change, changed", DataEdit{Values: map[string]string{"a": "9"}}, map[string]string{"a": "0"}, false},
		{"a change without from", DataEdit{Values: map[string]string{"a": "9"}}, nil, false},
		{"an add, absent", DataEdit{Add: map[string]string{"c": "3"}}, nil, true},
		{"an add that appeared", DataEdit{Add: map[string]string{"b": "3"}}, nil, false},
		{"an add that appeared in the binary data", DataEdit{Add: map[string]string{"bin": "3"}}, nil, false},
		{"a removal, unchanged", DataEdit{Remove: []string{"b"}}, map[string]string{"b": "2"}, true},
		{"a removal, changed", DataEdit{Remove: []string{"b"}}, map[string]string{"b": "1"}, false},
		{"a removal, gone", DataEdit{Remove: []string{"z"}}, map[string]string{"z": "1"}, false},
		{"a removal without from", DataEdit{Remove: []string{"b"}}, nil, false},
		{"nothing", DataEdit{}, nil, false},
	} {
		if got := c.edit.unchangedSince(data, map[string]any{"bin": "AA=="}, c.from, plain); got != c.want {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}
