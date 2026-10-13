package terminal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// A local terminal can be told which kubectl to find first (LocalOptions.Kubectl, D-3): the
// version Cogpit downloaded for the cluster, or the one the Preferences name. The session gets a
// directory of its own, removed when it closes:
//
//	<dir>/bin/kubectl   a link to that kubectl, so that nothing else from its folder joins PATH
//	<dir>/zsh/.zshenv   for zsh only (below)
//
// and PATH=<dir>/bin:<PATH> in the shell's environment. A login shell's profile runs after that and
// can reorder PATH: macOS's /etc/zprofile and /etc/profile run path_helper, which moves every
// inherited entry behind /etc/paths and /etc/paths.d (a kubectl in /usr/local/bin, Docker
// Desktop's link, then wins), and a profile can prepend its own folders (Homebrew's shellenv).
// For zsh, the shell macOS starts by default, Cogpit sets ZDOTDIR to <dir>/zsh: zsh then reads
// Cogpit's .zshenv first, which gives zsh the user's own ZDOTDIR back, reads the user's .zshenv,
// and registers a hook that puts <dir>/bin first again before the first prompt, after every
// profile ran, and then removes itself. Every other file zsh reads is the user's, from the user's
// ZDOTDIR, as without Cogpit. Other shells keep the environment's order only
// (docs/security/trust-boundaries.md, H-36).

// zshenv is the .zshenv of a zsh terminal whose kubectl Cogpit chose.
const zshenv = `# Written by Cogpit for one terminal and removed when it closes. zsh reads it in place of your
# .zshenv because Cogpit set ZDOTDIR: it gives zsh your own ZDOTDIR back, reads your .zshenv, and
# puts the kubectl Cogpit chose for this terminal first on PATH before the first prompt, after
# your profile ran.
if [[ ${COGPIT_ZDOTDIR_SET-} == 1 ]]; then
  ZDOTDIR=$COGPIT_ZDOTDIR
else
  unset ZDOTDIR
fi
unset COGPIT_ZDOTDIR COGPIT_ZDOTDIR_SET
if [[ -f ${ZDOTDIR:-$HOME}/.zshenv ]]; then
  source ${ZDOTDIR:-$HOME}/.zshenv
fi
cogpit_kubectl_first() {
  if [[ -n ${COGPIT_KUBECTL_DIR-} && -d $COGPIT_KUBECTL_DIR ]]; then
    path=("$COGPIT_KUBECTL_DIR" ${path:#$COGPIT_KUBECTL_DIR})
  fi
  unset COGPIT_KUBECTL_DIR
  precmd_functions=(${precmd_functions:#cogpit_kubectl_first})
  unfunction cogpit_kubectl_first
}
typeset -ga precmd_functions
precmd_functions+=(cogpit_kubectl_first)
`

// kubectlFirst prepares the session's own directory for opts.Kubectl and returns it with the
// environment entries that put it first; env is the shell's environment before them (its last
// PATH and ZDOTDIR count). The caller removes the directory when the session ends.
func kubectlFirst(kubectl, shellPath string, env []string) (dir string, extra []string, err error) {
	if !filepath.IsAbs(kubectl) {
		return "", nil, fmt.Errorf("local terminal: kubectl %q is not an absolute path", kubectl)
	}
	if fi, err := os.Stat(kubectl); err != nil || !fi.Mode().IsRegular() {
		return "", nil, fmt.Errorf("local terminal: kubectl %s is not a file", kubectl)
	}
	dir, err = os.MkdirTemp("", "cogpit-terminal-*") // owner-only (0700)
	if err != nil {
		return "", nil, fmt.Errorf("local terminal: %w", err)
	}
	fail := func(err error) (string, []string, error) {
		_ = os.RemoveAll(dir)
		return "", nil, fmt.Errorf("local terminal: preparing kubectl: %w", err)
	}
	bin := filepath.Join(dir, "bin")
	if strings.ContainsRune(bin, os.PathListSeparator) {
		return fail(fmt.Errorf("the temporary directory %s contains %q and cannot be put on PATH", bin, os.PathListSeparator))
	}
	if err := os.Mkdir(bin, 0o700); err != nil {
		return fail(err)
	}
	if err := os.Symlink(kubectl, filepath.Join(bin, binaryName())); err != nil {
		return fail(err)
	}
	path, _ := lastEnv(env, "PATH")
	if path != "" {
		path = string(os.PathListSeparator) + path
	}
	extra = []string{"PATH=" + bin + path}
	if filepath.Base(shellPath) == "zsh" {
		zdot := filepath.Join(dir, "zsh")
		if err := os.Mkdir(zdot, 0o700); err != nil {
			return fail(err)
		}
		if err := os.WriteFile(filepath.Join(zdot, ".zshenv"), []byte(zshenv), 0o600); err != nil {
			return fail(err)
		}
		set := "0"
		user, ok := lastEnv(env, "ZDOTDIR")
		if ok {
			set = "1"
		}
		extra = append(extra, "ZDOTDIR="+zdot, "COGPIT_ZDOTDIR="+user, "COGPIT_ZDOTDIR_SET="+set, "COGPIT_KUBECTL_DIR="+bin)
	}
	return dir, extra, nil
}

// binaryName is kubectl's file name on this computer.
func binaryName() string {
	if os.PathSeparator == '\\' {
		return "kubectl.exe"
	}
	return "kubectl"
}

// lastEnv returns the last value of key in env, as os/exec passes a duplicated key.
func lastEnv(env []string, key string) (string, bool) {
	for i := len(env) - 1; i >= 0; i-- {
		if k, v, ok := strings.Cut(env[i], "="); ok && k == key {
			return v, true
		}
	}
	return "", false
}
