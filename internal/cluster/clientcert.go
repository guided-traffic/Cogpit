package cluster

import (
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/guided-traffic/cogpit/internal/kubeconfig"
)

// The validity of a context's client certificate file (D-75). client-certificate-data is read by
// package kubeconfig; a client-certificate file is read here, and only it: the user's key file
// (client-key) is never opened, and of the certificate only its two dates are kept.
//
// When the file is read is the point (D-72, docs/security/file-access.md): listing the contexts
// must not raise a macOS privacy prompt the user did not ask for. So a file that lies in a place
// macOS protects (protectedPlace), or that a symbolic link leads into one, is not read while the
// contexts are listed: it is read right after client-go read it to connect (noteClientCertFile),
// where a prompt, if any, was client-go's own, and remembered for the process; its date shows
// from the first connect on. Any other file is read at every listing, as the kubeconfig is.

// maxCertFile bounds a client certificate file: a certificate with its chain is a few KiB.
const maxCertFile = 256 << 10

// macPrivacy says whether macOS's Files and Folders privacy guards the user's places: on macOS.
// Tests set it.
var macPrivacy atomic.Bool

func init() { macPrivacy.Store(runtime.GOOS == "darwin") }

// certMemo holds the validity of the client certificate files read at a connect, by path.
var certMemo = struct {
	sync.Mutex
	m map[string]kubeconfig.CertValidity
}{m: map[string]kubeconfig.CertValidity{}}

// maxCertMemo bounds the remembered files; a user has a few.
const maxCertMemo = 256

// certLogged holds the problems logged once (logCertProblem), by context, file and problem.
var certLogged = struct {
	sync.Mutex
	m map[string]bool
}{m: map[string]bool{}}

// maxCertLogged bounds the problems remembered as logged; past it, no more are logged.
const maxCertLogged = 256

// fillClientCerts sets the client certificate of every context whose user names a
// client-certificate file, and logs once each certificate that cannot be read. src names the
// kubeconfig file (or files) for the log.
func fillClientCerts(ctxs []kubeconfig.Context, src string) {
	for i := range ctxs {
		c := &ctxs[i]
		if c.ClientCertProblem != "" {
			logCertProblem(c.Name, src, c.ClientCertProblem)
			continue
		}
		if c.ClientCert != nil || c.ClientCertFile == "" {
			continue
		}
		v, ok, err := clientCertOfFile(c.ClientCertFile)
		switch {
		case err != nil:
			logCertProblem(c.Name, src, "client-certificate "+c.ClientCertFile+": "+err.Error())
		case ok:
			c.ClientCert = &v
		}
	}
}

// clientCertOfFile is the validity of a client certificate file: read now unless macOS protects
// where it lies (then as read at the last connect, ok false before one).
func clientCertOfFile(path string) (v kubeconfig.CertValidity, ok bool, err error) {
	if !filepath.IsAbs(path) {
		return v, false, nil // only a path resolved against its kubeconfig's folder is read
	}
	if !readableUnasked(path) {
		certMemo.Lock()
		defer certMemo.Unlock()
		v, ok = certMemo.m[filepath.Clean(path)]
		return v, ok, nil
	}
	v, err = readCertValidity(path)
	return v, err == nil, err
}

// noteClientCertFile reads a client certificate file right after client-go read it to connect and
// remembers its validity for the listings (clientCertOfFile). A file that cannot be read is
// forgotten: client-go's own read failed then too, and its connect says why.
func noteClientCertFile(path string) {
	if path == "" || !filepath.IsAbs(path) {
		return
	}
	v, err := readCertValidity(path)
	key := filepath.Clean(path)
	certMemo.Lock()
	defer certMemo.Unlock()
	if err != nil {
		delete(certMemo.m, key)
		return
	}
	if _, known := certMemo.m[key]; !known && len(certMemo.m) >= maxCertMemo {
		return
	}
	certMemo.m[key] = v
}

// readCertValidity reads a client certificate file, a regular file of at most maxCertFile bytes
// (a FIFO is never opened: opening it would wait for a writer), and returns its validity. Its
// errors never quote the file.
func readCertValidity(path string) (kubeconfig.CertValidity, error) {
	if fi, err := os.Stat(path); err != nil {
		return kubeconfig.CertValidity{}, errors.New("cannot be read")
	} else if !fi.Mode().IsRegular() {
		return kubeconfig.CertValidity{}, errors.New("not a regular file")
	}
	f, err := os.Open(path) // #nosec G304 -- the kubeconfig's own client-certificate, read as client-go reads it
	if err != nil {
		return kubeconfig.CertValidity{}, errors.New("cannot be read")
	}
	defer func() { _ = f.Close() }()
	if fi, err := f.Stat(); err != nil || !fi.Mode().IsRegular() {
		return kubeconfig.CertValidity{}, errors.New("not a regular file")
	}
	data, err := io.ReadAll(io.LimitReader(f, maxCertFile+1))
	switch {
	case err != nil:
		return kubeconfig.CertValidity{}, errors.New("cannot be read")
	case len(data) > maxCertFile:
		return kubeconfig.CertValidity{}, errors.New("larger than a certificate")
	}
	return kubeconfig.ParseCertValidity(data)
}

// logCertProblem logs, once per context, source and problem, at debug level, that a context's
// client certificate cannot be read: the cluster list then shows nothing for it (D-75), and this
// is not an error the user must act on.
func logCertProblem(context, src, problem string) {
	key := context + "\x00" + src + "\x00" + problem
	certLogged.Lock()
	if certLogged.m[key] || len(certLogged.m) >= maxCertLogged {
		certLogged.Unlock()
		return
	}
	certLogged.m[key] = true
	certLogged.Unlock()
	slog.Debug("cluster: a context's client certificate cannot be read; its expiry is not shown",
		"context", kubeconfig.DisplayText(context, 200), "kubeconfig", src, "problem", problem)
}

// readableUnasked reports whether reading path cannot raise a macOS privacy prompt: always off
// macOS; on macOS when neither the path nor a symbolic link on the way to it leads into a place
// macOS protects (protectedPlace). Nothing inside such a place is looked at to find out: each
// component is checked before it is looked at, and a link's target before it is followed.
func readableUnasked(path string) bool {
	if !macPrivacy.Load() {
		return true
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	homes := []string{filepath.Clean(home)}
	if resolved, err := filepath.EvalSymlinks(home); err == nil && resolved != homes[0] { // the home folder is not protected
		homes = append(homes, resolved)
	}
	p := filepath.Clean(path)
	for hops := 0; hops < 40; hops++ {
		if protectedPlace(homes, p) {
			return false
		}
		next, linked := followFirstLink(homes, p)
		switch {
		case next == "":
			return false // the way leads into a protected place
		case !linked:
			return true
		}
		p = next
	}
	return false // a loop of links: client-go's read fails the same way
}

// followFirstLink walks path's components from the root and returns the path with its first
// symbolic link replaced by the link's target (linked true), or path itself when none of its
// components is a link (linked false). It returns "" when a component lies in a protected place
// before a link was found. A component that cannot be looked at ends the walk: reading the file
// then fails the same way, without a prompt.
func followFirstLink(homes []string, path string) (next string, linked bool) {
	parts := strings.Split(strings.TrimPrefix(path, string(filepath.Separator)), string(filepath.Separator))
	cur := string(filepath.Separator)
	for i, part := range parts {
		comp := filepath.Join(cur, part)
		if protectedPlace(homes, comp) {
			return "", false
		}
		fi, err := os.Lstat(comp)
		if err != nil {
			return path, false
		}
		if fi.Mode()&os.ModeSymlink == 0 {
			cur = comp
			continue
		}
		target, err := os.Readlink(comp)
		if err != nil {
			return path, false
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(cur, target)
		}
		return filepath.Join(append([]string{target}, parts[i+1:]...)...), true
	}
	return path, false
}

// protectedPlace reports whether path is, or lies in, a place macOS's privacy settings protect
// (docs/security/file-access.md): the Desktop, Documents and Downloads folders, iCloud Drive, the
// cloud storage folders of file providers, other apps' containers, and the volumes under /Volumes
// (removable and network ones; the start-up disk's link there counts too), in each of homes (the
// home folder as named and as its links resolve). macOS's file system is case-insensitive by
// default: the names are compared so.
func protectedPlace(homes []string, path string) bool {
	if inPlace(path, "/Volumes") {
		return true
	}
	for _, home := range homes {
		for _, sub := range protectedInHome {
			if inPlace(path, filepath.Join(home, sub)) {
				return true
			}
		}
	}
	return false
}

// protectedInHome are the protected places in the home folder (protectedPlace).
var protectedInHome = []string{"Desktop", "Documents", "Downloads", filepath.Join("Library", "Mobile Documents"),
	filepath.Join("Library", "CloudStorage"), filepath.Join("Library", "Containers"), filepath.Join("Library", "Group Containers")}

// inPlace reports whether path is place or lies in it, ignoring case.
func inPlace(path, place string) bool {
	return strings.EqualFold(path, place) ||
		(len(path) > len(place) && strings.EqualFold(path[:len(place)], place) && path[len(place)] == filepath.Separator)
}
