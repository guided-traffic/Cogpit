package engine

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/guided-traffic/cogpit/internal/kubectl"
)

// kubectl in terminals (D-3, decided by the owner; architecture §10). A terminal bound to a
// cluster is where the user types kubectl, and kubectl is supported within one minor version of
// the API server. When a terminal of a connected cluster opens, the engine compares the kubectl the
// terminal would find (the Preferences' path to kubectl, else the first on Cogpit's PATH; its
// client version read once per file) with the version the API server reported at connect (no
// request). Within one minor version nothing changes. Otherwise a release Cogpit downloaded
// earlier into the directory for binaries that suits the cluster goes first on that terminal's
// PATH, and without one the terminal's tab offers the matching release (UIState: DockTab.Kubectl).
// The download runs only when the user clicks it in Cogpit's window (kubectl.download refuses
// agents and the CLI: it puts a program on the user's PATH), is verified before it is put in place
// (internal/kubectl), and is used by the terminals opened afterwards. Nothing is written outside
// the directory for binaries, and no shell profile or global location is touched.

// ErrKubectlGUIOnly refuses a kubectl download, and a change of the path to kubectl, the directory
// for binaries or the download mirror, from anywhere but the user's own action in Cogpit's window:
// each decides which program a terminal runs as kubectl.
var ErrKubectlGUIOnly = errors.New("only the user, in Cogpit's own window, can download kubectl or set the path to " +
	"kubectl, the directory for binaries or the download mirror; agents and the CLI are refused (they decide which " +
	"program Cogpit's terminals run as kubectl)")

// kubectlDownloadTimeout bounds one download (Cogpit's default): about 57 MB.
const kubectlDownloadTimeout = 15 * time.Minute

// KubectlDirName is the directory for binaries under Cogpit's configuration directory.
const KubectlDirName = "kubectl"

// Kubectl keeps what the engine knows about the kubectl its terminals run: the client versions it
// read (once per file) and the downloads running into the directory for binaries. Safe for
// concurrent use.
type Kubectl struct {
	dir    string // the default directory for binaries; "" when there is none
	prober *kubectl.Prober
	goos   string

	// lookPath finds kubectl on Cogpit's PATH; downloader makes the downloader for a mirror. Tests
	// replace both.
	lookPath   func(file string) (string, error)
	downloader func(mirror string) *kubectl.Downloader

	mu      sync.Mutex
	running map[string]*kubectlRun // by directory and version
}

// kubectlRun is one download; done is closed when it ended.
type kubectlRun struct {
	done chan struct{}
	path string
	err  error
}

// KubectlHooks replace how the engine finds kubectl on PATH and makes the downloader for a
// mirror; tests point them at fakes and a local release host. Nil fields: exec.LookPath, and the
// real release host or the mirror.
type KubectlHooks struct {
	LookPath   func(file string) (string, error)
	Downloader func(mirror string) *kubectl.Downloader
}

// NewKubectl returns the kubectl state with dir as the default directory for binaries (the app's:
// <user config dir>/Cogpit/kubectl); an empty dir has none, and downloads need the Preferences'.
func NewKubectl(dir string, hooks KubectlHooks) *Kubectl {
	k := &Kubectl{
		dir: dir, prober: kubectl.NewProber(), goos: runtime.GOOS, lookPath: hooks.LookPath,
		downloader: hooks.Downloader, running: map[string]*kubectlRun{},
	}
	if k.lookPath == nil {
		k.lookPath = exec.LookPath
	}
	if k.downloader == nil {
		k.downloader = func(mirror string) *kubectl.Downloader { return &kubectl.Downloader{Mirror: mirror} }
	}
	return k
}

// KubectlCheck is what a terminal's tab knows about its kubectl: the check made when it opened,
// the offer, and the offer's download.
type KubectlCheck struct {
	// Server is the cluster's version, as its API server reported it at connect ("1.36.1").
	Server string `json:"server"`
	// Found is the kubectl the terminal finds without Cogpit: the Preferences' path to kubectl, else
	// the first on Cogpit's PATH; empty when there is none. FromPreferences says which.
	Found           string `json:"found,omitempty"`
	FromPreferences bool   `json:"fromPreferences,omitempty"`
	// FoundVersion is its client version; empty when it could not be read (FoundError says why).
	FoundVersion string `json:"foundVersion,omitempty"`
	FoundError   string `json:"foundError,omitempty"`
	// Supported: the found kubectl is within one minor version of the cluster; nothing changes.
	Supported bool `json:"supported"`
	// First is the kubectl Cogpit put first on this terminal's PATH (a release it downloaded, or
	// the Preferences' path); empty: the terminal's PATH as it is.
	First        string `json:"first,omitempty"`
	FirstVersion string `json:"firstVersion,omitempty"`
	// Offer is the release Cogpit offers to download for this cluster ("1.36.1"); empty: none.
	Offer string `json:"offer,omitempty"`
	// DownloadOff: downloading is off in Preferences › Kubernetes, so the offer is a note.
	DownloadOff bool `json:"downloadOff,omitempty"`
	// Download is the offer's download, while it runs and after it ended.
	Download *KubectlDownload `json:"download,omitempty"`
	// Message is the line the terminal's tab shows; empty: none.
	Message string `json:"message,omitempty"`
}

// KubectlDownload is the state of a download.
type KubectlDownload struct {
	State string `json:"state"` // KubectlRunning, KubectlDone or KubectlFailed
	Done  int64  `json:"done,omitempty"`
	Total int64  `json:"total,omitempty"` // -1 while unknown
	Path  string `json:"path,omitempty"`  // the verified binary, once done
	Error string `json:"error,omitempty"`
}

// Download states.
const (
	KubectlRunning = "running"
	KubectlDone    = "done"
	KubectlFailed  = "failed"
)

func (c KubectlCheck) clone() KubectlCheck {
	if c.Download != nil {
		d := *c.Download
		c.Download = &d
	}
	return c
}

// Kubectl returns the session's kubectl state; nil when it has none.
func (s *Session) Kubectl() *Kubectl { return s.opts.Kubectl }

// KubectlBinDir is the directory for binaries in effect: the Preferences', else the default; empty
// without kubectl state or a directory.
func (s *Session) KubectlBinDir() string {
	if s.opts.Kubectl == nil {
		return ""
	}
	p, _ := s.Preferences()
	return s.opts.Kubectl.binDir(p)
}

// binDir is the directory for binaries: the Preferences', else k's default.
func (k *Kubectl) binDir(p Preferences) string {
	if p.KubectlDir != "" {
		return p.KubectlDir
	}
	return k.dir
}

// checkKubectl decides which kubectl a new terminal finds first against an API server that
// reported server, and what its tab says. It returns nil without a check (no kubectl state, or a
// version that does not parse) and the kubectl to put first ("" for PATH as it is).
func (s *Session) checkKubectl(ctx context.Context, server string) (*KubectlCheck, string) {
	k := s.opts.Kubectl
	if k == nil {
		return nil, ""
	}
	p, _ := s.Preferences() // the defaults without a store
	sv, err := kubectl.ParseVersion(server)
	if err != nil {
		if p.KubectlPath != "" && regularFile(p.KubectlPath) {
			return nil, p.KubectlPath // the user's choice, without a version to compare
		}
		return nil, ""
	}
	c := &KubectlCheck{Server: sv.String(), DownloadOff: !p.KubectlDownload}
	if p.KubectlPath != "" {
		c.Found, c.FromPreferences = p.KubectlPath, true
	} else if found, err := k.lookPath("kubectl"); err == nil {
		if abs, err := filepath.Abs(found); err == nil {
			c.Found = abs
		}
	}
	if c.Found != "" {
		v, err := k.prober.ClientVersion(ctx, c.Found)
		if err != nil {
			c.FoundError = safeLine(err.Error())
		} else {
			c.FoundVersion, c.Supported = v.String(), kubectl.Supported(v, sv)
		}
	}
	if c.FromPreferences && c.FoundError == "" {
		c.First, c.FirstVersion = c.Found, c.FoundVersion // the user's choice leads PATH
	}
	if !c.Supported {
		if best, ok := kubectl.Best(kubectl.Installed(k.binDir(p), k.goos), sv); ok {
			c.First, c.FirstVersion = best.Path, best.Version.String()
		} else {
			c.Offer = kubectl.Matching(sv).String()
			if run := k.current(k.binDir(p), c.Offer); run != nil {
				c.Download = &KubectlDownload{State: KubectlRunning, Total: -1}
			}
		}
	}
	c.Message = c.message()
	return c, c.First
}

func regularFile(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.Mode().IsRegular()
}

// found describes the kubectl the terminal finds without Cogpit, for the tab's line.
func (c *KubectlCheck) found() string {
	where := "on PATH"
	if c.FromPreferences {
		where = "set in Preferences"
	}
	switch {
	case c.Found == "":
		return "there is no kubectl on PATH"
	case c.FoundError != "":
		return "the version of the kubectl " + where + " could not be read"
	}
	return "the kubectl " + where + " is " + c.FoundVersion + ", more than one minor version away"
}

// message is the line the terminal's tab shows (Cogpit's wording, inferred: Lens downloads
// without asking and shows no such line).
func (c *KubectlCheck) message() string {
	switch {
	case c.Offer == "" && (c.First == "" || c.First == c.Found):
		return "" // the kubectl the terminal finds suits the cluster
	case c.Offer == "":
		return "kubectl " + c.FirstVersion + ", downloaded by Cogpit for this cluster's " + c.Server +
			", comes first on this terminal's PATH; " + c.found() + "."
	case c.Download != nil && c.Download.State == KubectlRunning:
		if c.Download.Total > 0 {
			return fmt.Sprintf("Downloading kubectl %s… %s of %s", c.Offer, megabytes(c.Download.Done), megabytes(c.Download.Total))
		}
		return "Downloading kubectl " + c.Offer + "…"
	case c.Download != nil && c.Download.State == KubectlDone:
		return "kubectl " + c.Offer + " is downloaded and verified: terminals opened from now on find it first."
	case c.Download != nil && c.Download.State == KubectlFailed:
		return "Downloading kubectl " + c.Offer + " failed: " + c.Download.Error
	case c.DownloadOff:
		return "kubectl " + c.Offer + " would match this cluster; " + c.found() + ". Downloading is off in Preferences › Kubernetes."
	}
	return "kubectl " + c.Offer + " matches this cluster; " + c.found() + "."
}

func megabytes(n int64) string { return fmt.Sprintf("%.1f MB", float64(n)/1e6) }

// safeLine makes text from a program, a file or a server one line of at most 300 bytes without
// control characters, for a tab's line.
func safeLine(s string) string {
	s = oneLine(strings.TrimSpace(s))
	if len(s) > 300 {
		cut := 300
		for cut > 0 && !utf8.RuneStart(s[cut]) {
			cut--
		}
		s = s[:cut] + "…"
	}
	return s
}

// current returns the running download of version into dir, nil when there is none.
func (k *Kubectl) current(dir, version string) *kubectlRun {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.running[dir+"\x00"+version]
}

// KubectlDownloadable says why a terminal tab cannot download kubectl now: no kubectl state, not
// a terminal, no offer, downloading off, or no directory for binaries.
func (s *Session) KubectlDownloadable(tabID string) error {
	_, _, _, err := s.kubectlOffer(tabID)
	return err
}

func (s *Session) kubectlOffer(tabID string) (*Kubectl, kubectl.Version, Preferences, error) {
	k := s.opts.Kubectl
	if k == nil {
		return nil, kubectl.Version{}, Preferences{}, errors.New("this Cogpit downloads no kubectl")
	}
	tab, ok := s.UIState().DockTab(tabID)
	if !ok {
		return nil, kubectl.Version{}, Preferences{}, fmt.Errorf("no dock tab %q", tabID)
	}
	if tab.Type != DockTerminal || tab.Kubectl == nil || tab.Kubectl.Offer == "" {
		return nil, kubectl.Version{}, Preferences{}, fmt.Errorf("the dock tab %q offers no kubectl download", tabID)
	}
	p, _ := s.Preferences()
	if !p.KubectlDownload {
		return nil, kubectl.Version{}, Preferences{}, errors.New("downloading kubectl is off in Preferences › Kubernetes")
	}
	if k.binDir(p) == "" {
		return nil, kubectl.Version{}, Preferences{}, errors.New("there is no directory for binaries: set one in Preferences › Kubernetes")
	}
	v, err := kubectl.ParseVersion(tab.Kubectl.Offer)
	if err != nil {
		return nil, kubectl.Version{}, Preferences{}, err
	}
	return k, v, p, nil
}

// DownloadKubectl downloads the release a terminal tab offers into the directory for binaries,
// verified (internal/kubectl), and returns its path; it waits for a download of the same release
// that already runs. Every terminal tab that offers the release shows the progress and the
// outcome. The caller (kubectl.download) has made sure that the user asked for it in the window.
func (s *Session) DownloadKubectl(ctx context.Context, tabID string) (string, error) {
	k, v, p, err := s.kubectlOffer(tabID)
	if err != nil {
		return "", err
	}
	dir, version := k.binDir(p), v.String()
	key := dir + "\x00" + version
	k.mu.Lock()
	if run := k.running[key]; run != nil {
		k.mu.Unlock()
		select {
		case <-run.done:
			return run.path, run.err
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	run := &kubectlRun{done: make(chan struct{})}
	k.running[key] = run
	k.mu.Unlock()
	defer func() {
		k.mu.Lock()
		delete(k.running, key)
		k.mu.Unlock()
		close(run.done)
	}()

	s.setKubectlDownload(version, KubectlDownload{State: KubectlRunning, Total: -1})
	dctx, cancel := context.WithTimeout(ctx, kubectlDownloadTimeout)
	defer cancel()
	var last time.Time
	progress := func(done, total int64) {
		if now := time.Now(); now.Sub(last) >= 200*time.Millisecond || done == total {
			last = now
			s.setKubectlDownload(version, KubectlDownload{State: KubectlRunning, Done: done, Total: total})
		}
	}
	run.path, run.err = k.downloader(p.KubectlMirror).Download(dctx, v, dir, progress)
	if run.err != nil {
		s.log.Warn("kubectl download failed", "version", version, "err", run.err)
		s.setKubectlDownload(version, KubectlDownload{State: KubectlFailed, Error: safeLine(run.err.Error())})
		return "", run.err
	}
	s.log.Info("kubectl downloaded and verified", "version", version, "path", run.path)
	s.setKubectlDownload(version, KubectlDownload{State: KubectlDone, Path: run.path})
	return run.path, nil
}

// setKubectlDownload shows a download's state in every terminal tab that offers the release.
func (s *Session) setKubectlDownload(version string, d KubectlDownload) {
	s.UpdateUI(func(u *UIState) {
		for i, t := range u.Dock {
			if t.Type != DockTerminal || t.Kubectl == nil || t.Kubectl.Offer != version {
				continue
			}
			c := t.Kubectl.clone()
			dl := d
			c.Download = &dl
			c.Message = c.message()
			u.Dock[i].Kubectl = &c
		}
	})
}

// setKubectlCheck records the check of a terminal tab that is opening; nil records nothing.
func (s *Session) setKubectlCheck(tabID string, c *KubectlCheck) {
	if c == nil {
		return
	}
	s.UpdateUI(func(u *UIState) {
		for i, t := range u.Dock {
			if t.ID == tabID {
				u.Dock[i].Kubectl = c
			}
		}
	})
}
