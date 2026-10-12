package main

import (
	"context"
	"fmt"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/kubeconfig"
)

const contextsAbout = `Lists the contexts Cogpit offers, as the app's context picker and the MCP tool cluster_contexts
list them: every context of ~/.kube, of the files --kubeconfig (else $KUBECONFIG) names, and of the
files and sync folders added in Cogpit, each file read on its own and named "<context> [<file>]",
and the synthetic demo cluster "demo". LOGIN says whether a context logs in through a credential
plugin or a token file outside its folder, and whether the user allowed it in Cogpit's window:
approved, required or changed. Listing reads the kubeconfig files only; it runs no plugin and
prints no credential.`

func runContexts(ctx context.Context, args []string, e *env) int {
	fs := newFlags("contexts", "[flags]", contextsAbout)
	var c common
	c.register(fs)
	output := fs.String("o", outTable, "output format: table, json or yaml (json and yaml are cluster_contexts' result)")
	fs.StringVar(output, "output", outTable, "same as -o")
	if code, done := fs.parse(args, e); done {
		return code
	}
	if len(fs.positional) > 0 {
		return fs.usageError(e, "contexts takes no arguments")
	}
	if !validOutput(*output) {
		return fs.usageError(e, fmt.Sprintf("unknown output format %q: want table, json or yaml", *output))
	}
	c.apply(e)

	s, closeSession, err := e.newSession(sessionOptions{kubeconfig: c.kubeconfig})
	if err != nil {
		return fail(e, err)
	}
	defer closeSession()
	var list []engine.ContextInfo
	data, err := dispatchInto(ctx, s, "cluster.contexts", nil, &list)
	if err != nil {
		return fail(e, err)
	}
	if *output != outTable {
		if err := writeResult(e.stdout, *output, data); err != nil {
			return fail(e, err)
		}
		return exitOK
	}
	rows := make([][]string, 0, len(list))
	for _, c := range list {
		server := c.Server
		if c.Demo {
			server = "(synthetic demo cluster)"
		}
		rows = append(rows, []string{c.Name, c.Namespace, server, login(c)})
	}
	if err := writeTable(e.stdout, []string{"CONTEXT", "NAMESPACE", "SERVER", "LOGIN"}, rows); err != nil {
		return fail(e, err)
	}
	return exitOK
}

// login describes how a context logs in when the user must allow it: "plugin approved",
// "token file required", "plugin and token file changed"; empty otherwise.
func login(c engine.ContextInfo) string {
	if c.Exec == "" {
		return ""
	}
	what := "plugin"
	switch c.ExecKind {
	case kubeconfig.LoginTokenFile:
		what = "token file"
	case kubeconfig.LoginBoth:
		what = "plugin and token file"
	}
	return what + " " + c.Exec
}
