package kubectl

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// Bounds of a version probe (Cogpit's defaults): kubectl answers "version --client" in well under
// a second and prints about 300 bytes.
const (
	probeTimeout   = 10 * time.Second
	maxProbeOutput = 64 << 10
)

// Prober reads the client version of kubectl binaries, once per file: a binary is run again only
// when its size or modification time changed. Safe for concurrent use.
type Prober struct {
	mu    sync.Mutex
	cache map[string]probed

	// run runs path as "kubectl version --client -o json" and returns what it printed on standard
	// output; tests replace it.
	run func(ctx context.Context, path string) ([]byte, error)
}

type probed struct {
	size    int64
	modTime time.Time
	version Version
	err     error
}

// NewProber returns a Prober that runs the binaries it is asked about.
func NewProber() *Prober { return &Prober{cache: map[string]probed{}, run: runVersion} }

// ClientVersion returns the client version of the kubectl at path, an absolute file name: from
// the cache while the file is unchanged, else by running it once with "version --client -o json"
// (no API server is asked). A binary that does not answer within ten seconds, prints more than
// 64 KiB or no version is an error, cached like a version.
func (p *Prober) ClientVersion(ctx context.Context, path string) (Version, error) {
	path = filepath.Clean(path)
	fi, err := os.Stat(path)
	if err != nil {
		return Version{}, fmt.Errorf("kubectl %s: %w", path, err)
	}
	if !fi.Mode().IsRegular() {
		return Version{}, fmt.Errorf("kubectl %s is not a file", path)
	}
	p.mu.Lock()
	c, ok := p.cache[path]
	p.mu.Unlock()
	if ok && c.size == fi.Size() && c.modTime.Equal(fi.ModTime()) {
		return c.version, c.err
	}
	out, err := p.run(ctx, path)
	var v Version
	if err == nil {
		v, err = parseClientVersion(out)
	}
	if err != nil {
		err = fmt.Errorf("kubectl %s: reading its version: %w", path, err)
	}
	if ctx.Err() == nil { // a cancelled probe says nothing about the binary
		p.mu.Lock()
		p.cache[path] = probed{size: fi.Size(), modTime: fi.ModTime(), version: v, err: err}
		p.mu.Unlock()
	}
	return v, err
}

// parseClientVersion reads the output of "kubectl version --client -o json".
func parseClientVersion(out []byte) (Version, error) {
	var doc struct {
		ClientVersion *struct {
			GitVersion string `json:"gitVersion"`
		} `json:"clientVersion"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		return Version{}, fmt.Errorf("not the JSON of kubectl version: %w", err)
	}
	if doc.ClientVersion == nil {
		return Version{}, errors.New("no clientVersion in its output")
	}
	return ParseVersion(doc.ClientVersion.GitVersion)
}

// runVersion runs "kubectl version --client -o json" with Cogpit's environment, bounded in time
// and output.
func runVersion(ctx context.Context, path string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "version", "--client", "-o", "json")
	out := &boundedBuffer{max: maxProbeOutput}
	cmd.Stdout = out
	cmd.Stderr = io.Discard
	cmd.WaitDelay = time.Second
	err := cmd.Run()
	switch {
	case out.over: // whatever Run says: os/exec may drain the rest after a failed write
		return nil, fmt.Errorf("more than %d bytes of output", maxProbeOutput)
	case err != nil && ctx.Err() != nil:
		return nil, fmt.Errorf("no answer within %s", probeTimeout)
	case err != nil:
		return nil, err
	}
	return out.Bytes(), nil
}

// boundedBuffer keeps at most max bytes; a write beyond fails, which ends the copy from the
// program. It does not embed bytes.Buffer, whose ReadFrom would let io.Copy pass the bound.
type boundedBuffer struct {
	buf  bytes.Buffer
	max  int
	over bool
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	if b.buf.Len()+len(p) > b.max {
		b.over = true
		return 0, errors.New("output too long")
	}
	return b.buf.Write(p)
}

// Bytes returns what was kept.
func (b *boundedBuffer) Bytes() []byte { return b.buf.Bytes() }
