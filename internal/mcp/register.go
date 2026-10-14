package mcp

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"
)

// ClaudeAddStdioCommand returns a ready-to-paste shell command that registers the stdio bridge at
// cli, the cogpit-cli an app bundle carries (Cogpit.app/Contents/MacOS/cogpit-cli), with Claude
// Code for every directory (user scope), replacing an earlier registration such as
// ClaudeAddCommand's:
//
//	claude mcp remove --scope user cogpit >/dev/null 2>&1; claude mcp add --scope user cogpit -- /Applications/Cogpit.app/Contents/MacOS/cogpit-cli mcp
//
// Unlike ClaudeAddCommand it carries no port and no token: Claude Code starts `cogpit-cli mcp`,
// which reads the running app's endpoint and token files itself (architecture §7), so the command
// is run once and the registration follows every restart of Cogpit. The path is quoted for a POSIX
// shell when it needs it (a space, a quote, a $). It returns "" for a path that is not absolute or
// holds a control character, which no pasted command should carry; the caller then offers
// ClaudeAddCommand.
func ClaudeAddStdioCommand(cli string) string {
	if !filepath.IsAbs(cli) || strings.ContainsFunc(cli, unicode.IsControl) {
		return ""
	}
	return fmt.Sprintf(`claude mcp remove --scope user %[1]s >/dev/null 2>&1; `+
		`claude mcp add --scope user %[1]s -- %[2]s mcp`, serverName, shellQuote(cli))
}

// shellQuote returns s as one word of a POSIX shell (sh, bash, zsh): unchanged when every
// character is one no shell treats specially, else in single quotes, inside which nothing is
// expanded (no $, no backquote, no zsh history !), each single quote written as a closing quote, a
// backslash-escaped quote and an opening quote.
func shellQuote(s string) string {
	if s != "" && !strings.ContainsFunc(s, func(r rune) bool { return !shellSafe(r) }) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// shellSafe reports whether r needs no quoting anywhere in a word (Python's shlex.quote keeps the
// same set; = and % only matter at a word's start, and a path starts with /).
func shellSafe(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		return true
	}
	return strings.ContainsRune("@%+=:,./-_", r)
}
