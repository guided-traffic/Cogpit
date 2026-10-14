package engine

// What Cogpit does for a context whose kubeconfig switches the check of the API server's
// certificate off (insecure-skip-tls-verify, H-37 in docs/security/trust-boundaries.md): client-go
// connects without checking, as kubectl does, and sends the context's credential to whoever answers
// at the server's address. Lens's documentation says nothing about such contexts.
//
//   - The picker marks the context (ContextInfo.Insecure, from the kubeconfig; Cogpit's default of
//     2026-10-13).
//   - A source context connects only once the user allowed it in the window, asked once per
//     context before its first connect and kept like a credential plugin's approval, in the same
//     store and bound the same way: the file, context, user, server and how it is reached (the
//     kubeconfig's proxy-url and Cogpit's own proxy), so a changed server or proxy asks again (the
//     owner's A28 of 2026-10-14 evening; kubeconfig.LoginInsecure, KubeconfigSources.Policy). A
//     context that runs a credential plugin is asked in that plugin's dialog, which warns about the
//     check. Agents, the CLI and links are refused until then, as for a plugin.
//   - The connect log gets a WARN line once the connection stands, from the connection's own
//     configuration, so it is what client-go actually does.

// insecureLogLine is the connect log's warning for a connection that does not check the server's
// certificate; "" for any other.
func insecureLogLine(b *Backend) string {
	if b == nil || b.Conn == nil || !b.Conn.Insecure() {
		return ""
	}
	return "Certificate checks are off for this context (insecure-skip-tls-verify in its kubeconfig): " +
		"Cogpit cannot tell the cluster at " + b.Conn.Server() + " from anyone else who answers there, " +
		"and sends the context's credential to whoever does"
}
