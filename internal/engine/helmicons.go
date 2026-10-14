package engine

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/guided-traffic/cogpit/internal/helm"
)

// Chart icons (D-46, a night decision for the owner to confirm; HELM2, G11): the Charts view and a
// chart's drawer show the icon a repository's index names for the chart (inferred: Lens shows
// icons). Fetching one tells the host that serves it, which may be anyone's, that this computer
// looked at the chart; so the window asks for an icon only for what it shows, the Preferences'
// "Load chart icons" (Preferences.LoadChartIcons, on by default) switches it off, and agents never
// fetch one. An icon is fetched as helm.IconFetcher does it (https only, no credentials, at most
// 256 KiB, public addresses or a repository's own host, through the Preferences' proxy), checked
// as an uploaded cluster icon is (CheckIcon: a PNG or JPEG decoded under bounds and kept as a PNG
// of at most 256 pixels a side, an SVG kept only when it is one <svg> document that refers to
// nothing outside itself; anything else is refused), and kept in Cogpit's own cache folder
// (<cache>/helm/icons/<hash of the URL>.png or .svg, owner-only) for DefaultIconTTL. At most
// maxIconFetches download at once, one per URL; a failure is remembered for iconRetryAfter.
// Missing, refused or failed, the window shows Cogpit's own helm-wheel glyph.

// Defaults of the chart icons (Cogpit's, D-46).
const (
	// DefaultIconTTL is how long a fetched icon is used before it is fetched again (when shown).
	DefaultIconTTL = 7 * 24 * time.Hour
	// iconRetryAfter is how long a failed icon is not asked for again.
	iconRetryAfter = time.Hour
	// maxIconFetches bounds the icon downloads at once.
	maxIconFetches = 4
	// maxIconsKept bounds the icons in memory and maxIconFiles those in the cache folder.
	maxIconsKept = 512
	maxIconFiles = 2048
	// helmIconDir holds the icons under Cogpit's cache folder.
	helmIconDir = "helm/icons"
)

// ErrChartIconsOff refuses a chart icon while the Preferences' "Load chart icons" is off.
var ErrChartIconsOff = errors.New("loading chart icons is off (Preferences › Kubernetes › Load chart icons)")

// ErrChartIconsGUIOnly refuses turning "Load chart icons" on from anywhere but the user's own
// action in Cogpit's window: it decides whether Cogpit contacts the hosts that serve the icons.
var ErrChartIconsGUIOnly = errors.New("only the user, in Cogpit's own window, can turn on loading chart icons; agents and " +
	"the CLI may only turn it off (it decides whether Cogpit contacts the hosts that serve the icons)")

// RefuseChartIcons refuses a preferences change that turns chart icons on from another origin
// than the GUI.
func RefuseChartIcons(ctx context.Context, on *bool) error {
	if on != nil && *on && OriginOf(ctx) != OriginGUI {
		return ErrChartIconsGUIOnly
	}
	return nil
}

// chartIcons keeps a session's chart icons. Its zero value with dir set is ready.
type chartIcons struct {
	dir     string // the cache folder; empty: memory only
	fetcher *helm.IconFetcher
	sem     chan struct{}
	mu      sync.Mutex
	entries map[string]*iconEntry // by URL
	clock   uint64
}

// iconEntry is an icon held in memory: kept, failed, or downloading.
type iconEntry struct {
	data   []byte
	format string // "png", "svg"
	err    error
	at     time.Time     // when it was fetched or failed
	done   chan struct{} // closed when its download ends
	used   uint64
}

// iconFile is the cache folder's file of an icon URL, without its extension.
func iconFile(iconURL string) string {
	sum := sha256.Sum256([]byte(iconURL))
	return hex.EncodeToString(sum[:16])
}

// ChartIcon returns a chart icon as Cogpit keeps it, for the window: its bytes and format ("png",
// "svg"). It fetches the icon when it is not kept, or kept longer than DefaultIconTTL, waiting for
// the download (bounded by ctx), unless the Preferences' "Load chart icons" is off
// (ErrChartIconsOff). Only the window calls it: agents read the icon's URL from the chart.
func (s *Session) ChartIcon(ctx context.Context, iconURL string) (data []byte, format string, err error) {
	h := s.HelmRepos()
	if h == nil {
		return nil, "", ErrNoHelmRepos
	}
	if p, _ := s.Preferences(); !p.LoadChartIcons {
		return nil, "", ErrChartIconsOff
	}
	return h.icon(ctx, iconURL)
}

// icon returns the icon at iconURL: from memory, the cache folder, or a download.
func (h *HelmRepos) icon(ctx context.Context, iconURL string) ([]byte, string, error) {
	ic := &h.icons
	now := h.now()
	ic.mu.Lock()
	e := ic.entries[iconURL]
	if e != nil && !iconLoading(e) && (e.err != nil && now.Sub(e.at) >= iconRetryAfter || e.err == nil && now.Sub(e.at) >= DefaultIconTTL) {
		e = nil // stale: fetched again
	}
	if e == nil {
		if data, format, at, ok := ic.read(iconURL); ok && now.Sub(at) < DefaultIconTTL {
			e = &iconEntry{data: data, format: format, at: at, done: closedDone}
		} else {
			e = &iconEntry{done: make(chan struct{})}
			go h.fetchIcon(iconURL, e)
		}
		ic.put(iconURL, e)
	}
	ic.clock++
	e.used = ic.clock
	ic.mu.Unlock()
	select {
	case <-e.done:
	case <-ctx.Done():
		return nil, "", ctx.Err()
	}
	if e.err != nil {
		return nil, "", e.err
	}
	return e.data, e.format, nil
}

// iconLoading reports whether e's download runs.
func iconLoading(e *iconEntry) bool {
	select {
	case <-e.done:
		return false
	default:
		return true
	}
}

// fetchIcon downloads, checks and keeps an icon, then ends e.
func (h *HelmRepos) fetchIcon(iconURL string, e *iconEntry) {
	ic := &h.icons
	var (
		data   []byte
		format string
		err    = errors.New("internal error")
	)
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("internal error: %v", p)
		}
		ic.mu.Lock()
		e.data, e.format, e.err, e.at = data, format, err, h.now()
		ic.mu.Unlock()
		close(e.done)
	}()
	ic.sem <- struct{}{}
	defer func() { <-ic.sem }()
	_ = h.ensureLoaded() // the repositories' hosts, whose icons may be at a private address
	raw, ferr := ic.fetcher.FetchIcon(context.Background(), iconURL, h.ownHost)
	if ferr != nil {
		err = ferr
		return
	}
	data, format, err = CheckIcon(raw)
	if err == nil && ic.dir != "" {
		ic.write(iconURL, data, format)
	}
}

// ownHost reports whether a host:port is one of the repositories' Cogpit uses: their icons may be
// at a private address.
func (h *HelmRepos) ownHost(hostPort string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.ContainsFunc(h.entries, func(e repoEntry) bool {
		return e.repo != nil && helm.HostPort(e.repo.URL) == hostPort
	})
}

// put holds e, dropping the least recently used finished icon beyond maxIconsKept. The caller
// holds mu.
func (ic *chartIcons) put(iconURL string, e *iconEntry) {
	if ic.entries == nil {
		ic.entries = map[string]*iconEntry{}
	}
	ic.entries[iconURL] = e
	for len(ic.entries) > maxIconsKept {
		oldest := ""
		var at uint64
		for u, c := range ic.entries {
			if !iconLoading(c) && (oldest == "" || c.used < at) {
				oldest, at = u, c.used
			}
		}
		if oldest == "" {
			return
		}
		delete(ic.entries, oldest)
	}
}

// read returns an icon the cache folder keeps, checked again as Cogpit wrote it (a regular file
// of bounded size, a PNG of at most iconSide pixels a side or a checked SVG), and when it was
// fetched (its modification time).
func (ic *chartIcons) read(iconURL string) (data []byte, format string, at time.Time, ok bool) {
	if ic.dir == "" {
		return nil, "", time.Time{}, false
	}
	for _, f := range []string{iconPNG, iconSVG} {
		path := filepath.Join(ic.dir, iconFile(iconURL)+"."+f)
		fi, err := os.Lstat(path)
		if err != nil || !fi.Mode().IsRegular() || fi.Size() > maxIconUpload {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil || checkKeptIcon(data, f) != nil {
			continue
		}
		return data, f, fi.ModTime(), true
	}
	return nil, "", time.Time{}, false
}

// checkKeptIcon checks an icon file Cogpit wrote before it is drawn: someone else's file is no
// reason to decode a huge image or render a foreign document.
func checkKeptIcon(data []byte, format string) error {
	if format == iconSVG {
		return checkSVG(data)
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width > iconSide || cfg.Height > iconSide {
		return errors.New("not a PNG Cogpit wrote")
	}
	return nil
}

// write keeps an icon in the cache folder (owner-only, atomically), removing the other format's
// file of the URL and the oldest icons beyond maxIconFiles. A failure costs a download later.
func (ic *chartIcons) write(iconURL string, data []byte, format string) {
	base := filepath.Join(ic.dir, iconFile(iconURL))
	if writePrivateFile(base+"."+format, data) != nil {
		return
	}
	for _, f := range []string{iconPNG, iconSVG} {
		if f != format {
			_ = os.Remove(base + "." + f)
		}
	}
	pruneIconFiles(ic.dir, maxIconFiles)
}

// pruneIconFiles removes the oldest icons of dir beyond most.
func pruneIconFiles(dir string, most int) {
	des, err := os.ReadDir(dir)
	if err != nil || len(des) <= most {
		return
	}
	type kept struct {
		name string
		at   time.Time
	}
	var files []kept
	for _, de := range des {
		base, ext, _ := strings.Cut(de.Name(), ".")
		if !de.Type().IsRegular() || (ext != iconPNG && ext != iconSVG) || len(base) != 32 {
			continue
		}
		if fi, err := de.Info(); err == nil {
			files = append(files, kept{de.Name(), fi.ModTime()})
		}
	}
	if len(files) <= most {
		return
	}
	slices.SortFunc(files, func(a, b kept) int { return a.at.Compare(b.at) })
	for _, f := range files[:len(files)-most] {
		_ = os.Remove(filepath.Join(dir, f.name))
	}
}
