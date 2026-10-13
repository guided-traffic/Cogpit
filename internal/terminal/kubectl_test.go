//go:build !windows

package terminal

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeKubectl writes a kubectl that prints its name, so that a test sees which one ran.
func fakeKubectl(t *testing.T, dir, says string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "kubectl")
	if err := os.WriteFile(path, []byte("#!/bin/sh\necho "+says+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestKubectlFirstEnv: the session's own directory holds a link to the chosen kubectl and leads
// PATH; for zsh, ZDOTDIR points at Cogpit's .zshenv and remembers the user's.
func TestKubectlFirstEnv(t *testing.T) {
	target := fakeKubectl(t, filepath.Join(t.TempDir(), "1.36.1"), "cogpit")
	env := []string{"PATH=/old:/usr/bin", "HOME=/h", "PATH=/usr/local/bin:/usr/bin", "ZDOTDIR=/h/.config/zsh"}

	dir, extra, err := kubectlFirst(target, "/bin/bash", env)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	bin := filepath.Join(dir, "bin")
	if len(extra) != 1 || extra[0] != "PATH="+bin+":/usr/local/bin:/usr/bin" {
		t.Errorf("bash: %q", extra)
	}
	if link, err := os.Readlink(filepath.Join(bin, "kubectl")); err != nil || link != target {
		t.Errorf("link %q, %v", link, err)
	}
	if fi, _ := os.Stat(dir); fi.Mode().Perm() != 0o700 {
		t.Errorf("mode %o", fi.Mode().Perm())
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 1 {
		t.Errorf("bash gets a zsh folder: %v", entries)
	}

	zdir, zextra, err := kubectlFirst(target, "/bin/zsh", env)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(zdir)
	want := []string{"PATH=" + filepath.Join(zdir, "bin") + ":/usr/local/bin:/usr/bin", "ZDOTDIR=" + filepath.Join(zdir, "zsh"),
		"COGPIT_ZDOTDIR=/h/.config/zsh", "COGPIT_ZDOTDIR_SET=1", "COGPIT_KUBECTL_DIR=" + filepath.Join(zdir, "bin")}
	if strings.Join(zextra, "\n") != strings.Join(want, "\n") {
		t.Errorf("zsh:\n%s\nwant\n%s", strings.Join(zextra, "\n"), strings.Join(want, "\n"))
	}
	if data, err := os.ReadFile(filepath.Join(zdir, "zsh", ".zshenv")); err != nil || string(data) != zshenv {
		t.Errorf(".zshenv: %v", err)
	}
	_, unset, _ := kubectlFirst(target, "/usr/local/bin/zsh", []string{"PATH=/usr/bin"})
	if !strings.Contains(strings.Join(unset, " "), "COGPIT_ZDOTDIR_SET=0") {
		t.Errorf("no ZDOTDIR: %q", unset)
	}

	for _, bad := range []string{"kubectl", filepath.Join(t.TempDir(), "missing"), t.TempDir()} {
		if d, _, err := kubectlFirst(bad, "/bin/sh", env); err == nil {
			os.RemoveAll(d)
			t.Errorf("kubectl %q accepted", bad)
		}
	}

	// localEnv: the kubectl entries come after opts.Env and before KUBECONFIG.
	full := localEnv([]string{"PATH=/from-opts"}, "/own:/user", "PATH=/k/bin:/from-opts")
	if p, _ := lastEnv(full, "PATH"); p != "/k/bin:/from-opts" {
		t.Errorf("PATH %q", p)
	}
	if k, _ := lastEnv(full, "KUBECONFIG"); k != "/own:/user" {
		t.Errorf("KUBECONFIG %q", k)
	}
}

// TestStartLocalKubectlFirst: in a login sh, the chosen kubectl leads the environment's PATH, and
// the session's directory is gone after close.
func TestStartLocalKubectlFirst(t *testing.T) {
	target := fakeKubectl(t, filepath.Join(t.TempDir(), "1.36.1"), "cogpit-kubectl")
	sess, out, home := startTestShell(t, LocalOptions{
		KubeconfigPaths: []string{writeUserKubeconfig(t)}, Context: "ctx-a", Kubectl: target,
	})
	script := `printf '%s\n' "$PATH" > path.txt; echo "__DONE$((40+2))__"` + "\n"
	if _, err := sess.Write([]byte(script)); err != nil {
		t.Fatal(err)
	}
	out.waitFor(t, "__DONE42__", 20*time.Second)
	path := strings.Split(readTrimmed(t, filepath.Join(home, "path.txt")), ":")
	var own string
	for _, p := range path {
		if strings.HasSuffix(p, "/bin") && strings.Contains(p, "cogpit-terminal-") {
			own = p
		}
	}
	if own == "" {
		t.Fatalf("PATH %q has no session directory", path)
	}
	// /bin/sh's profile may reorder PATH (macOS's path_helper): the entry must be there, and the
	// link must lead to the chosen kubectl.
	if link, err := os.Readlink(filepath.Join(own, "kubectl")); err != nil || link != target {
		t.Errorf("link %q, %v", link, err)
	}
	_ = sess.Close()
	waitClosed(t, sess.Done(), 10*time.Second, "session")
	if _, err := os.Stat(filepath.Dir(own)); !os.IsNotExist(err) {
		t.Errorf("the session's directory is still there: %v", err)
	}
}

// TestZshKubectlFirstAfterTheProfile starts a login zsh whose profile puts another kubectl first,
// as Homebrew's shellenv and macOS's path_helper do, and checks that kubectl is still Cogpit's
// choice at the prompt, that the user's own startup files ran from the user's ZDOTDIR, and that
// zsh's ZDOTDIR is the user's again.
func TestZshKubectlFirstAfterTheProfile(t *testing.T) {
	zsh, err := exec.LookPath("zsh")
	if err != nil {
		t.Skip("no zsh")
	}
	home := t.TempDir()
	userZdot := filepath.Join(home, ".config", "zsh")
	if err := os.MkdirAll(userZdot, 0o700); err != nil {
		t.Fatal(err)
	}
	shadow := filepath.Join(home, "shadow")
	fakeKubectl(t, shadow, "shadow-kubectl")
	files := map[string]string{
		".zshenv":   "export COGPIT_TEST_ZSHENV=ran\n",
		".zprofile": "path=(" + shadow + " $path)\n", // the profile puts its kubectl first
		".zshrc":    "PS1='$ '\nexport COGPIT_TEST_ZSHRC=ran\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(userZdot, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	target := fakeKubectl(t, filepath.Join(t.TempDir(), "1.36.1"), "cogpit-kubectl")
	sess, err := StartLocal(LocalOptions{
		Shell: zsh, Dir: home, KubeconfigPaths: []string{writeUserKubeconfig(t)}, Context: "ctx-a", Kubectl: target,
		Env: []string{"HOME=" + home, "ZDOTDIR=" + userZdot},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sess.Close() })
	out := record(sess)
	script := strings.Join([]string{
		`kubectl > which.txt 2>&1`,
		`printf '%s\n' "$ZDOTDIR" > zdotdir.txt`,
		`printf '%s %s\n' "$COGPIT_TEST_ZSHENV" "$COGPIT_TEST_ZSHRC" > ran.txt`,
		`printf '%s\n' "${COGPIT_KUBECTL_DIR-unset}" "${COGPIT_ZDOTDIR-unset}" > leftovers.txt`,
		`echo "__DONE$((40+2))__"`,
	}, "\n") + "\n"
	if _, err := sess.Write([]byte(script)); err != nil {
		t.Fatal(err)
	}
	out.waitFor(t, "__DONE42__", 20*time.Second)
	if got := readTrimmed(t, filepath.Join(home, "which.txt")); got != "cogpit-kubectl" {
		t.Errorf("kubectl at the prompt is %q, want Cogpit's choice", got)
	}
	if got := readTrimmed(t, filepath.Join(home, "zdotdir.txt")); got != userZdot {
		t.Errorf("ZDOTDIR %q, want the user's %q", got, userZdot)
	}
	if got := readTrimmed(t, filepath.Join(home, "ran.txt")); got != "ran ran" {
		t.Errorf("the user's .zshenv and .zshrc: %q", got)
	}
	if got := readTrimmed(t, filepath.Join(home, "leftovers.txt")); got != "unset\nunset" {
		t.Errorf("Cogpit's variables left in the shell: %q", got)
	}
}
