package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// childEnv marks a run of this test binary as cogpit-cli itself (TestMain).
const childEnv = "COGPIT_CLI_TEST_MAIN"

func TestMain(m *testing.M) {
	if os.Getenv(childEnv) == "1" {
		main()
	}
	os.Exit(m.Run())
}

// TestMCPOverRealStdio runs cogpit-cli mcp as its own process, as an agent host does, with a home
// of its own (so Cogpit's real directory and ~/.kube are never read): the agent speaks MCP over the
// process's stdin and stdout, nothing else reaches stdout (logs go to stderr), and closing stdin
// ends the process.
func TestMCPOverRealStdio(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, "mcp", "-v")
	cmd.Env = []string{
		childEnv + "=1", "HOME=" + home, "USERPROFILE=" + home, "XDG_CONFIG_HOME=" + filepath.Join(home, ".config"),
		"KUBECONFIG=" + os.DevNull, "PATH=" + os.Getenv("PATH"),
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	cs, err := sdk.NewClient(&sdk.Implementation{Name: "agent", Version: "1"}, nil).Connect(ctx, &sdk.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatalf("connect: %v; stderr: %s", err, stderr.String())
	}
	var ctxs []engine.ContextInfo
	if res := callTool(t, cs, "cluster_contexts", nil, &ctxs); res.IsError || len(ctxs) != 1 || ctxs[0].Name != "demo" {
		t.Errorf("cluster_contexts in an empty home: %s %+v", resultText(res), ctxs)
	}
	var st engine.UIState
	if res := callTool(t, cs, "ui_state", nil, &st); res.IsError || st.Connection.State != engine.Disconnected {
		t.Errorf("ui_state: %s", resultText(res))
	}
	if err := cs.Close(); err != nil {
		t.Errorf("close: %v; stderr: %s", err, stderr.String())
	}
	// Closing stdin is the normal end: exit code 0, no error.
	if cmd.ProcessState == nil || cmd.ProcessState.ExitCode() != exitOK || strings.Contains(stderr.String(), "cogpit-cli: ") {
		t.Errorf("process state %v; stderr: %s", cmd.ProcessState, stderr.String())
	}
	// -v logs at debug level, to stderr: the bridge found no running Cogpit and served its own engine.
	if !strings.Contains(stderr.String(), "no running Cogpit serves MCP") {
		t.Errorf("stderr: %s", stderr.String())
	}
}
