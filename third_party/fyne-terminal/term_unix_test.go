//go:build !windows
// +build !windows

package terminal

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunLocalShell_MissingStartDir(t *testing.T) {
	term := New()
	term.Resize(fyne.NewSize(45, 45))
	term.SetStartDir(filepath.Join(t.TempDir(), "missing"))

	keyTyped() // the working directory is only checked after recent key presses
	assert.Error(t, term.RunLocalShell())

	// a failed start must not leave the directory check running against a missing process
	time.Sleep(time.Millisecond * 300)
}

func TestProcessDir(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("working directory lookup is not supported on " + runtime.GOOS)
	}

	dir, err := filepath.EvalSymlinks(t.TempDir())
	assert.NoError(t, err)

	cmd := exec.Command("sleep", "10")
	cmd.Dir = dir
	assert.NoError(t, cmd.Start())
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	assert.Equal(t, dir, processDir(cmd.Process.Pid))
}

func TestProcessDir_MissingProcess(t *testing.T) {
	cmd := exec.Command("true")
	assert.NoError(t, cmd.Run())

	assert.Equal(t, "", processDir(cmd.Process.Pid))
}

func TestTerminal_Close(t *testing.T) {
	term := New()
	term.Resize(fyne.NewSize(45, 45))
	done := startLocalShell(t, term)

	// a program that will not exit by itself, or respond to input
	_, err := term.Write([]byte("sleep 600\n"))
	require.NoError(t, err)
	child := 0
	// Cogpit patch: require, with shellTimeout; without the child the test went on to signal process 0,
	// its own process group, and failed in confusing ways
	require.Eventually(t, func() bool {
		out, _ := exec.Command("pgrep", "-P", strconv.Itoa(term.cmd.Process.Pid), "sleep").Output()
		child, _ = strconv.Atoi(strings.TrimSpace(string(out)))
		return child != 0
	}, shellTimeout, time.Millisecond*20)
	running := func() bool {
		return syscall.Kill(child, 0) == nil
	}
	assert.True(t, running())

	term.Close()
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(time.Second * 5):
		t.Fatal("Terminal did not stop after Close")
	}
	assert.Eventually(t, func() bool { return !running() }, time.Second*2, time.Millisecond*20)
	assert.Nil(t, term.dirWatchDone)

	term.Close() // closing again is harmless
}

func TestTerminal_Close_BeforeOpen(t *testing.T) {
	term := New() // without a size the shell will not start
	done := make(chan error)
	go func() {
		done <- term.RunLocalShell()
	}()

	term.Close()
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(time.Second * 5):
		t.Fatal("Terminal did not stop after Close")
	}
	assert.Nil(t, term.cmd)
}

// TestRunLocalShell_MainLoop runs a local shell with fyne.Do on a stand-in for the desktop driver's
// main goroutine (mainLoop), where the terminal is sized, typed into and resized as in a window,
// so that the race detector sees RunLocalShell's goroutines apart from the main one. Cogpit patch:
// Fyne's test driver runs fyne.Do on the calling goroutine, so the tests above could not show the
// races between RunLocalShell, the directory check and the main goroutine.
func TestRunLocalShell_MainLoop(t *testing.T) {
	// a shell that does not report its directory itself (OSC 7, as macOS's zsh does), so that the
	// directory check has to, and that reads no profile of the user's
	t.Setenv("SHELL", "/bin/sh")
	loop := useMainLoop(t)
	dir, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	sub := filepath.Join(dir, "sub")
	require.NoError(t, os.Mkdir(sub, 0o700))

	term := New()
	term.SetStartDir(dir)
	configs := make(chan Config, 64)
	term.AddListener(configs)
	done := make(chan error, 1)
	go func() {
		done <- term.RunLocalShell() // before the terminal has a size, as in a window's new tab
	}()
	t.Cleanup(term.Close)

	loop.do(func() {
		term.Resize(fyne.NewSize(450, 450))
		_, _ = term.Write(nil) // as typing does, possibly while the shell starts
	})
	deadline := time.Now().Add(shellTimeout)
	for !isConnected(term) {
		require.True(t, time.Now().Before(deadline), "the local shell's terminal was not open after %s", shellTimeout)
		time.Sleep(10 * time.Millisecond)
	}

	// The exit code, read as an application may at any time, -1 until the shell has exited, on a
	// goroutine that synchronises with nothing else and lives on: the race detector can miss what a
	// goroutine that has finished read.
	codes, release := make(chan int, 1), make(chan struct{})
	go func() {
		code := term.ExitCode()
		<-release
		codes <- code
	}()

	typeLine := func(s string) func() {
		return func() {
			for _, r := range s {
				term.TypedRune(r)
			}
			term.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
		}
	}
	loop.do(typeLine("cd sub"))
	loop.do(func() { term.Resize(fyne.NewSize(600, 450)) })

	// The directory is checked only within half a second of a key press; a key that sends nothing
	// keeps the check going until the shell's new directory is reported, while the window is
	// resized, which reads the configuration the check writes.
	widths := []float32{450, 600}
	for i, pwd := 0, ""; pwd != sub; i++ {
		select {
		case c := <-configs:
			pwd = c.PWD
		case <-time.After(100 * time.Millisecond):
			require.True(t, time.Now().Before(deadline), "the directory %s was not reported after %s", sub, shellTimeout)
			width := widths[i%2]
			loop.do(func() {
				term.TypedKey(&fyne.KeyEvent{Name: desktop.KeyShiftLeft})
				term.Resize(fyne.NewSize(width, 450))
			})
		}
	}

	loop.do(typeLine("exit 3"))
	waitShellExit(t, term, done)
	assert.Equal(t, 3, term.ExitCode())
	close(release)
	assert.Contains(t, []int{-1, 3}, <-codes)
	loop.flush()
}
