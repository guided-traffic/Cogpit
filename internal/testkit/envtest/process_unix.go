//go:build unix

package envtest

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// apiServerCommand writes a shell wrapper around the kube-apiserver binary that records its own
// process id in pidFile and then execs the server, which keeps that id. envtest runs the binary
// once with --help first (it probes for a flag); that run records nothing.
func apiServerCommand(t testing.TB, server, pidFile, dir string) string {
	t.Helper()
	script := "#!/bin/sh\n" +
		"case \" $* \" in *\" --help \"*) exec " + shellQuote(server) + " \"$@\" ;; esac\n" +
		"echo $$ > " + shellQuote(pidFile) + "\n" +
		"exec " + shellQuote(server) + " \"$@\"\n"
	path := filepath.Join(dir, "kube-apiserver-wrapper")
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

// shellQuote quotes s for sh: in single quotes, each single quote closed, escaped and reopened.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func stopProcess(pid int) error     { return syscall.Kill(pid, syscall.SIGSTOP) }
func continueProcess(pid int) error { return syscall.Kill(pid, syscall.SIGCONT) }
