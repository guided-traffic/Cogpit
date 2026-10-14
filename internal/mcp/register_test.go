package mcp

import (
	"os/exec"
	"testing"
)

// TestClaudeAddStdioCommand: the registration names the bundled cogpit-cli by its absolute path,
// after replacing an earlier registration; a path a pasted command should not carry gives none.
func TestClaudeAddStdioCommand(t *testing.T) {
	for _, tc := range []struct{ cli, want string }{
		{"/Applications/Cogpit.app/Contents/MacOS/cogpit-cli",
			"claude mcp remove --scope user cogpit >/dev/null 2>&1; " +
				"claude mcp add --scope user cogpit -- /Applications/Cogpit.app/Contents/MacOS/cogpit-cli mcp"},
		{"/Users/a b/Apps/Cogpit.app/Contents/MacOS/cogpit-cli",
			"claude mcp remove --scope user cogpit >/dev/null 2>&1; " +
				"claude mcp add --scope user cogpit -- '/Users/a b/Apps/Cogpit.app/Contents/MacOS/cogpit-cli' mcp"},
		{"Cogpit.app/Contents/MacOS/cogpit-cli", ""},
		{"", ""},
		{"/Applications/Co\ngpit.app/Contents/MacOS/cogpit-cli", ""},
		{"/Applications/Co\x1b]0;x\agpit.app/Contents/MacOS/cogpit-cli", ""},
	} {
		if got := ClaudeAddStdioCommand(tc.cli); got != tc.want {
			t.Errorf("ClaudeAddStdioCommand(%q):\n got %s\nwant %s", tc.cli, got, tc.want)
		}
	}
}

// TestShellQuoteRoundTrips: a POSIX shell reads every quoted path back as the same one word,
// whatever it holds, and expands nothing in it.
func TestShellQuoteRoundTrips(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no sh")
	}
	for _, s := range []string{
		"/Applications/Cogpit.app/Contents/MacOS/cogpit-cli",
		"/Users/a b/My Apps/Cogpit.app/Contents/MacOS/cogpit-cli",
		"/Volumes/it's/Cogpit.app",
		`/tmp/$HOME/$(id)/` + "`id`" + `/"q"/\n/!x/*/?/[a]/~/;&|<>(){}#`,
		"/tmp/-n",
		"/tmp/ünïcode/Cögpit.app",
	} {
		out, err := exec.Command(sh, "-c", "printf '%s|' "+shellQuote(s)).Output() // #nosec G204 -- the test's own words
		if err != nil {
			t.Fatalf("sh on %q: %v", s, err)
		}
		if got := string(out); got != s+"|" {
			t.Errorf("shellQuote(%q) = %s reads back as %q", s, shellQuote(s), got)
		}
	}
	if got := shellQuote("/Applications/Cogpit.app/Contents/MacOS/cogpit-cli"); got != "/Applications/Cogpit.app/Contents/MacOS/cogpit-cli" {
		t.Errorf("a plain path is quoted: %s", got)
	}
}
