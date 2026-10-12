package fyneui

import (
	"strings"

	"github.com/guided-traffic/cogpit/internal/kubeconfig"
)

// What the approval dialog shows about how the credential travels and which files are bound
// (R-11, R-5, D-17): the facts the approval's fingerprint binds, which decide who can read the
// credential. All text comes from a kubeconfig and goes through kubeconfig.DisplayText.

// approvalFacts are the dialog's lines about the connection: warnings, shown prominently, and
// rows for the key/value list. Plain TLS to the server with the system's trust gives none.
type approvalFacts struct {
	warnings []string
	rows     [][2]string // name, value
}

// shortDigest is the front of a hex digest: enough to compare, short enough for a row.
func shortDigest(hex string) string {
	if len(hex) > 16 {
		return hex[:16] + "…"
	}
	return hex
}

// connectionFacts derives the lines from the plugin the engine described.
func connectionFacts(info execPluginView) approvalFacts {
	show := func(s string) string { return kubeconfig.DisplayText(s, maxShownPath) }
	pl := info.Plugin
	var f approvalFacts
	ownCA := pl.TLS.CAEmbedded || pl.TLS.CAFile != ""
	if pl.TLS.Insecure {
		f.warnings = append(f.warnings, "Certificate checks are off for this connection (insecure-skip-tls-verify): "+
			"anyone on the network path can read what is sent to the server, the login credential included.")
		f.rows = append(f.rows, [2]string{"Certificate check", "off"})
	}
	if pl.TLS.ProxyURL != "" {
		w := "The connection goes through the proxy " + show(pl.TLS.ProxyURL) + "."
		if pl.TLS.Insecure || ownCA {
			w += " With the settings here the proxy can read the login credential."
		}
		f.warnings = append(f.warnings, w)
		f.rows = append(f.rows, [2]string{"Proxy", show(pl.TLS.ProxyURL)})
	}
	switch {
	case pl.TLS.CAEmbedded && pl.TLS.CAFile != "":
		f.warnings = append(f.warnings, "The kubeconfig brings its own certificate authority (embedded and a file) "+
			"instead of your system's: it decides which server counts as genuine.")
	case pl.TLS.CAEmbedded:
		f.warnings = append(f.warnings, "The kubeconfig brings its own certificate authority (embedded) instead of "+
			"your system's: it decides which server counts as genuine.")
	case pl.TLS.CAFile != "":
		f.warnings = append(f.warnings, "The kubeconfig names its own certificate authority file instead of your "+
			"system's: it decides which server counts as genuine.")
	}
	if pl.TLS.CAEmbedded {
		f.rows = append(f.rows, [2]string{"Trusted CA", "embedded in the kubeconfig"})
	}
	if pl.TLS.CAFile != "" {
		v := show(pl.TLS.CAFile)
		switch {
		case pl.Files.CAFileSHA256 != "":
			v += " (inside its folder; content SHA-256 " + shortDigest(pl.Files.CAFileSHA256) + ", replacing it asks again)"
		case pl.Files.CAFileNote != "":
			v += " (inside its folder; content not hashed: " + show(pl.Files.CAFileNote) + ")"
		}
		f.rows = append(f.rows, [2]string{"CA file", v})
	}
	if pl.TLS.ServerName != "" {
		f.rows = append(f.rows, [2]string{"TLS server name", show(pl.TLS.ServerName)})
	}
	return f
}

// programFileText says how the program's file is bound, "" when it is bound by its path alone.
func programFileText(info execPluginView) string {
	f := info.Plugin.Files
	switch {
	case f.ProgramSHA256 != "":
		return "inside the kubeconfig's folder; content SHA-256 " + shortDigest(f.ProgramSHA256) + ", replacing the file asks again"
	case f.ProgramNote != "":
		return "inside the kubeconfig's folder; content not hashed: " + kubeconfig.DisplayText(f.ProgramNote, maxShownPath)
	case f.ProgramResolved != "":
		return "a link in the kubeconfig's folder, pointing to " + kubeconfig.DisplayText(f.ProgramResolved, maxShownPath) +
			"; repointing it asks again"
	}
	return ""
}

// approvalTitle and approvalLead are the dialog's title and first paragraph by kind and state.
func approvalTitle(info execPluginView) string {
	changed := info.State == "changed"
	switch info.Kind {
	case kubeconfig.LoginTokenFile:
		if changed {
			return "The token file changed"
		}
		return "Allow this file as the login token?"
	}
	if changed {
		return "The login program changed"
	}
	return "Allow this program to log in?"
}

func approvalLead(info execPluginView) string {
	ctx := kubeconfig.DisplayText(info.Plugin.Context, maxShownPath)
	switch info.Kind {
	case kubeconfig.LoginTokenFile:
		return "To log in to " + ctx + ", this kubeconfig sends the content of a file outside its own folder, on this " +
			"computer, to the server below as the login token. Cogpit reads it only if you allow it. Allow it only if " +
			"you trust where the kubeconfig came from."
	case kubeconfig.LoginBoth:
		return "To log in to " + ctx + ", this kubeconfig runs a program on this computer, with your permissions, and " +
			"sends the content of a file outside its own folder to the server below. Cogpit does so only if you allow " +
			"it. Allow it only if you trust where the kubeconfig came from."
	}
	return "To log in to " + ctx + ", this kubeconfig runs a program on this computer, with your " +
		"permissions. Cogpit runs it only if you allow it. Allow it only if you trust where the kubeconfig came from."
}

func approvalChangedNote(info execPluginView) string {
	if info.Kind == kubeconfig.LoginTokenFile {
		return "You allowed an earlier form of this login. Its token file or server changed since."
	}
	return "You allowed an earlier form of this plugin. Its program (or the file behind it), arguments, environment, " +
		"server or how the server is reached changed since."
}

// joinNames shows variable names, one line.
func joinNames(names []string) string {
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = kubeconfig.DisplayText(n, 100)
	}
	return strings.Join(out, ", ")
}
