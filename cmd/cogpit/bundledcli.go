package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// bundledCLIName is the command-line tool `make app` puts beside the app's own executable, in
// Cogpit.app/Contents/MacOS, where Apple's "Placing Content in a Bundle" puts helper tools (D-16).
const bundledCLIName = "cogpit-cli"

// bundledCLI returns the path of the cogpit-cli in the running app's bundle, "" when this Cogpit
// is not an installed macOS bundle that carries one (bundledCLIBeside): a dev build (bin/cogpit,
// go run) and every other system offer the HTTP registration instead.
func bundledCLI() string {
	if runtime.GOOS != "darwin" {
		return ""
	}
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return bundledCLIBeside(exe, readOnlyVolume)
}

// bundledCLIBeside returns the path of the cogpit-cli in the app bundle whose main executable is
// exe, or "" when there is none to register: exe is not <name>.app/Contents/MacOS/<exe>, the bundle
// runs translocated (a quarantined copy macOS moved to a random path for this run) or from a
// read-only volume (its disk image), whose paths are gone once the app quits or the image is
// ejected, or the bundle carries no executable cogpit-cli (a bundle built before it did). The path
// is the running executable's own, its symbolic links resolved; it is never guessed.
func bundledCLIBeside(exe string, readOnly func(string) bool) string {
	exe, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return ""
	}
	macOS := filepath.Dir(exe)
	contents := filepath.Dir(macOS)
	bundle := filepath.Dir(contents)
	if filepath.Base(macOS) != "MacOS" || filepath.Base(contents) != "Contents" || filepath.Ext(bundle) != ".app" {
		return ""
	}
	if strings.Contains(bundle, "/AppTranslocation/") || readOnly(bundle) {
		return ""
	}
	cli := filepath.Join(macOS, bundledCLIName)
	fi, err := os.Lstat(cli)
	if err != nil || !fi.Mode().IsRegular() || fi.Mode().Perm()&0o111 == 0 {
		return ""
	}
	return cli
}
