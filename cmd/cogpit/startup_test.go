package main

import (
	"bytes"
	"os"
	"os/exec"
	"testing"
)

// TestStartUpLogsNoFyneError: nothing cogpit links asks Fyne for the current app before Run makes
// one. Package initialisation did: the sidebar's icons logged "Fyne error: Attempt to access
// current Fyne app when none is started" ten times at every start (fyneui, navIcons). A fresh
// process of this test binary initialises every package cogpit does and runs no test; this
// package must not import Fyne's test package, whose init makes an app and would hide the lines.
func TestStartUpLogsNoFyneError(t *testing.T) {
	if testing.Short() {
		t.Skip("starts a process")
	}
	out, err := exec.Command(os.Args[0], "-test.run=^$", "-test.count=1").CombinedOutput()
	if err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if bytes.Contains(out, []byte("Fyne error")) || bytes.Contains(out, []byte("current Fyne app")) {
		t.Errorf("start-up logged:\n%s", out)
	}
}
