package fyneui

import (
	"fmt"
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

// connectionFacts derives the lines from how the engine described one context's connection.
func connectionFacts(c approvalContextView) approvalFacts {
	show := func(s string) string { return kubeconfig.DisplayText(s, maxShownPath) }
	pl := c
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
func programFileText(g approvalGroupView) string {
	f := g.Plugin.Files
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

// changedCount is how many of the group's contexts the user allowed in an earlier form.
func changedCount(g approvalGroupView) int {
	n := 0
	for _, c := range g.Contexts {
		if c.State == "changed" {
			n++
		}
	}
	return n
}

// approvalTitle and approvalLead are the dialog's title and first paragraph by kind and state.
func approvalTitle(g approvalGroupView) string {
	many := len(g.Contexts) > 1
	changed := changedCount(g) == len(g.Contexts)
	switch g.Kind {
	case kubeconfig.LoginTokenFile:
		switch {
		case changed:
			return "The token file changed"
		case many:
			return fmt.Sprintf("Allow this file as the login token for %d contexts?", len(g.Contexts))
		}
		return "Allow this file as the login token?"
	}
	switch {
	case changed:
		return "The login program changed"
	case many:
		return fmt.Sprintf("Allow this program to log in to %d contexts?", len(g.Contexts))
	}
	return "Allow this program to log in?"
}

func approvalLead(g approvalGroupView) string {
	to := "To log in to " + kubeconfig.DisplayText(g.Contexts[0].KubeContext, maxShownPath)
	all := ""
	if len(g.Contexts) > 1 {
		to = fmt.Sprintf("To log in to the %d contexts below", len(g.Contexts))
		all = " Allowing it allows it for every context listed."
	}
	switch g.Kind {
	case kubeconfig.LoginTokenFile:
		return to + ", this kubeconfig sends the content of a file outside its own folder, on this " +
			"computer, to the server of each as the login token. Cogpit reads it only if you allow it." + all +
			" Allow it only if you trust where the kubeconfig came from."
	case kubeconfig.LoginBoth:
		return to + ", this kubeconfig runs a program on this computer, with your permissions, and " +
			"sends the content of a file outside its own folder to the server of each. Cogpit does so only if you allow " +
			"it." + all + " Allow it only if you trust where the kubeconfig came from."
	}
	return to + ", this kubeconfig runs a program on this computer, with your " +
		"permissions. Cogpit runs it only if you allow it." + all + " Allow it only if you trust where the kubeconfig came from."
}

// approvalChangedNote says what changed; n of the group's contexts were allowed in an earlier form.
func approvalChangedNote(g approvalGroupView, n int) string {
	of := ""
	if len(g.Contexts) > 1 {
		of = fmt.Sprintf(" (%d of %d, marked changed)", n, len(g.Contexts))
	}
	if g.Kind == kubeconfig.LoginTokenFile {
		return "You allowed an earlier form of this login" + of + ". Its token file or server changed since."
	}
	return "You allowed an earlier form of this plugin" + of + ". Its program (or the file behind it), arguments, environment, " +
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
