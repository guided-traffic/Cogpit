package cluster

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	restclient "k8s.io/client-go/rest"
)

// The discovery cache on disk (G3, D-65; architecture §5). Connecting asks the API server which
// groups and resources it serves (ServedResources), and the REST mapper of the object writes asks
// again; on a cluster with a few hundred custom resource definitions that is a large document (two
// requests with aggregated discovery, one per group version without it). A connection with a
// DiscoveryCacheDir keeps the answer in one file per server and identity, which the next connection
// (the window's after a restart, every run of cogpit-cli) reuses instead of asking:
//
//   - The file is named by a SHA-256 over the server URL (without user information), its TLS server
//     name, the kubeconfig files, the context's cluster and user entry names and the user it
//     impersonates: an identity, never a credential. It holds the server URL, the server's version,
//     when the answer was fetched, a digest of the custom resource definitions it was checked
//     against, and the groups and resources as the API server named them. No token, certificate,
//     key or object. Owner-only: 0600 in a 0700 directory, written whole (a temporary file renamed).
//   - It is used only for the same server version (Connect's GET /version, which it makes anyway)
//     and for DefaultDiscoveryCacheTTL (6 h, kubectl's own value) once the cluster's definitions
//     confirmed it, ServedResourcesTTL (10 min) before that or for a user who may not list them.
//     A partial answer (an aggregated API that did not answer) is never written and is asked again
//     after a minute.
//   - The engine's watch of the custom resource definitions reports their digest (DefinitionsSeen)
//     whenever it rebuilds the custom kinds: an answer checked against another set of definitions,
//     or read from a file that was never checked, is dropped from memory and from disk, so the
//     next read asks the server; an answer fetched in this connection takes the digest as its own.
//     A kind the REST mapper does not know resets it and asks the server again (objects.go).
//
// What a stale answer can cause, until one of the above drops it, is named in D-65 (decisions.md): a
// built-in kind's view whose API the server stopped serving without a new version shows the server's
// refusal; a kind the server began serving without a new version or definition (an aggregated API)
// is missing until the TTL; the custom kinds come from the definitions' watch, not from discovery.

const (
	// DefaultDiscoveryCacheTTL is how long a discovery answer the cluster's custom resource
	// definitions confirmed is reused, from the file or in memory (kubectl reuses its discovery
	// cache for six hours too, without that check).
	DefaultDiscoveryCacheTTL = 6 * time.Hour
	// discoveryFileFormat is the file's format; another one is not read.
	discoveryFileFormat = 1
	// discoveryClockSkew is how far in the future a file's fetch time may lie before the file is
	// not trusted (a clock that was set back).
	discoveryClockSkew = time.Minute
)

// discoveryConfig says where and how long a connection keeps its discovery answer.
type discoveryConfig struct {
	file    string // the cache file; empty: memory only
	key     string // the identity's digest, stored in the file and checked when it is read
	server  string // the server URL without user information
	version string // the server's version as Connect saw it
	context string // for log lines only
	ttl     time.Duration
}

// discoveryKey is the digest that names a connection's cache file: the server and the identity
// that asks it, without a credential. A proxy that routes one URL to clusters by the client's
// identity (Teleport's) gives each identity its own file.
func discoveryKey(server, tlsServerName string, files []string, clusterEntry, userEntry string, imp restclient.ImpersonationConfig) string {
	groups := slices.Clone(imp.Groups)
	slices.Sort(groups)
	parts := []string{"cogpit-discovery-v1", server, tlsServerName, strings.Join(files, "\x1f"), clusterEntry, userEntry,
		imp.UserName, imp.UID, strings.Join(groups, "\x1f")}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:])
}

// discoveryFile is the file's content.
type discoveryFile struct {
	Format      int                       `json:"format"`
	Key         string                    `json:"key"`
	Server      string                    `json:"server"`
	Version     string                    `json:"serverVersion"`
	FetchedAt   time.Time                 `json:"fetchedAt"`
	Definitions string                    `json:"definitions,omitempty"` // the digest it was checked against; empty: never checked
	Groups      *metav1.APIGroupList      `json:"groups"`
	Resources   []*metav1.APIResourceList `json:"resources"`
}

// discoverySnapshot is one discovery answer.
type discoverySnapshot struct {
	groups    *metav1.APIGroupList
	resources map[schema.GroupVersion]*metav1.APIResourceList
	failed    map[schema.GroupVersion]error
	fetched   time.Time
	defs      string // the definitions' digest it was checked against (or fetched after); "" unknown
	fromFile  bool
	noticed   bool // its partial answer was logged
}

// firstPartialNotice reports, once per answer, that ServedResources should log the groups a
// partial answer left out.
func (d *discoveryCache) firstPartialNotice() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.snap == nil || d.snap.noticed {
		return d.snap == nil
	}
	d.snap.noticed = true
	return true
}

// DiscoveryInfo says where a connection's discovery answer came from, for the Diagnostics view.
type DiscoveryInfo struct {
	// Source is "disk" (the cache file), "server" (asked in this connection) or "" (not read yet).
	Source    string    `json:"source,omitempty"`
	FetchedAt time.Time `json:"fetchedAt,omitzero"`
	// Checked: the cluster's custom resource definitions confirmed the answer.
	Checked   bool `json:"checked,omitempty"`
	Groups    int  `json:"groups"`
	Resources int  `json:"resources"`
	// Partial: groups that did not answer are left out (asked again after a minute).
	Partial bool `json:"partial,omitempty"`
	// Persisted: the connection keeps its answer on disk (a cache directory is set).
	Persisted bool `json:"persisted"`
}

// discoveryCache is the layer under client-go's memory cache (memory.NewMemCacheClientWithContext)
// that reads the answer from the file or asks the live discovery client, and writes what it asked.
// It implements the aggregated discovery interface, so the memory cache takes groups and resources
// from one call, and the cached interface, so the memory cache's Invalidate reaches it.
type discoveryCache struct {
	// The live discovery client: what the cache does not hold (the server's version, OpenAPI, the
	// REST client, one group version's resources) goes to the server.
	discovery.DiscoveryInterfaceWithContext
	live discovery.DiscoveryInterfaceWithContext
	cfg  discoveryConfig
	now  func() time.Time
	log  *slog.Logger

	mu       sync.Mutex
	snap     *discoverySnapshot // the answer the memory cache holds; nil before the first read and after Invalidate
	skipFile bool               // after Invalidate the file is not read again in this connection
	defs     string             // the definitions' digest the engine reported last: stamps the next answer
	warned   bool               // a failed write was logged
}

var (
	_ discovery.AggregatedDiscoveryInterfaceWithContext = (*discoveryCache)(nil)
	_ discovery.CachedDiscoveryInterfaceWithContext     = (*discoveryCache)(nil)
)

func newDiscoveryCache(live discovery.DiscoveryInterfaceWithContext, cfg discoveryConfig) *discoveryCache {
	if cfg.ttl <= 0 {
		cfg.ttl = DefaultDiscoveryCacheTTL
	}
	return &discoveryCache{DiscoveryInterfaceWithContext: live, live: live, cfg: cfg, now: time.Now, log: slog.Default()}
}

// ttlOf is how long an answer is reused: a partial one a minute, one the definitions did not
// check ServedResourcesTTL, a checked one the configured TTL.
func (d *discoveryCache) ttlOf(s *discoverySnapshot) time.Duration {
	switch {
	case len(s.failed) > 0:
		return partialServedResourcesTTL
	case s.defs == "":
		return min(ServedResourcesTTL, d.cfg.ttl)
	}
	return d.cfg.ttl
}

// expired reports whether the answer the memory cache holds is older than its TTL.
func (d *discoveryCache) expired() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.snap != nil && d.now().Sub(d.snap.fetched) >= d.ttlOf(d.snap)
}

// GroupsAndMaybeResourcesWithContext returns the groups and their resources: from the file when it
// is valid for this connection, else from the server (and then written to the file). The memory
// cache above calls it once per refresh.
func (d *discoveryCache) GroupsAndMaybeResourcesWithContext(ctx context.Context) (*metav1.APIGroupList, map[schema.GroupVersion]*metav1.APIResourceList, map[schema.GroupVersion]error, error) {
	d.mu.Lock()
	if !d.skipFile {
		if s := d.load(); s != nil {
			d.snap = s
			d.mu.Unlock()
			d.log.Info("cluster: discovery from the disk cache", "context", d.cfg.context,
				"age", d.now().Sub(s.fetched).Round(time.Second), "checked", s.defs != "")
			return s.groups, s.resources, nil, nil
		}
	}
	stamp := d.defs
	d.mu.Unlock()

	s, err := d.fetch(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	s.defs = stamp
	d.mu.Lock()
	d.snap = s
	d.store(s)
	d.mu.Unlock()
	return s.groups, s.resources, s.failed, nil
}

// fetch asks the live discovery client for every group and its resources: aggregated discovery
// where the server has it (two requests), else one request per group version. A partial answer
// (groups that did not answer) is returned with them as failed.
func (d *discoveryCache) fetch(ctx context.Context) (*discoverySnapshot, error) {
	groups, lists, err := discovery.ServerGroupsAndResourcesWithContext(ctx, d.live)
	var failed *discovery.ErrGroupDiscoveryFailed
	if err != nil && (!errors.As(err, &failed) || len(groups) == 0) {
		return nil, err
	}
	s := &discoverySnapshot{groups: &metav1.APIGroupList{}, resources: map[schema.GroupVersion]*metav1.APIResourceList{}, fetched: d.now()}
	for _, g := range groups {
		if g != nil {
			s.groups.Groups = append(s.groups.Groups, *g)
		}
	}
	for _, l := range lists {
		if l == nil {
			continue
		}
		if gv, perr := schema.ParseGroupVersion(l.GroupVersion); perr == nil {
			s.resources[gv] = l
		}
	}
	if failed != nil {
		s.failed = failed.Groups
	}
	return s, nil
}

// load reads the file and returns its answer when it belongs to this connection: the format, the
// identity and the server's version match, and it is younger than its TTL. Called with d.mu held.
func (d *discoveryCache) load() *discoverySnapshot {
	if d.cfg.file == "" {
		return nil
	}
	data, err := os.ReadFile(d.cfg.file)
	if err != nil {
		return nil
	}
	var f discoveryFile
	if err := json.Unmarshal(data, &f); err != nil || f.Format != discoveryFileFormat || f.Key != d.cfg.key ||
		f.Version != d.cfg.version || f.Version == "" || f.Groups == nil {
		return nil
	}
	s := &discoverySnapshot{groups: f.Groups, resources: map[schema.GroupVersion]*metav1.APIResourceList{},
		fetched: f.FetchedAt, defs: f.Definitions, fromFile: true}
	age := d.now().Sub(f.FetchedAt)
	if age < -discoveryClockSkew || age >= d.ttlOf(s) {
		return nil
	}
	for _, l := range f.Resources {
		if l == nil {
			continue
		}
		gv, err := schema.ParseGroupVersion(l.GroupVersion)
		if err != nil {
			return nil
		}
		s.resources[gv] = l
	}
	return s
}

// store writes a complete answer to the file, owner-only and whole. A partial answer is not
// written. Called with d.mu held.
func (d *discoveryCache) store(s *discoverySnapshot) {
	if d.cfg.file == "" || len(s.failed) > 0 || d.cfg.version == "" {
		return
	}
	f := discoveryFile{Format: discoveryFileFormat, Key: d.cfg.key, Server: d.cfg.server, Version: d.cfg.version,
		FetchedAt: s.fetched.UTC(), Definitions: s.defs, Groups: s.groups, Resources: make([]*metav1.APIResourceList, 0, len(s.resources))}
	for _, l := range s.resources {
		f.Resources = append(f.Resources, l)
	}
	slices.SortFunc(f.Resources, func(a, b *metav1.APIResourceList) int { return strings.Compare(a.GroupVersion, b.GroupVersion) })
	if err := writeOwnerOnly(d.cfg.file, f); err != nil && !d.warned {
		d.warned = true
		d.log.Warn("cluster: the discovery cache cannot be written; discovery stays in memory", "context", d.cfg.context, "err", err)
	}
}

// writeOwnerOnly writes v as JSON to file through a temporary file in its directory (0600; the
// directory 0700), renamed over the file.
func writeOwnerOnly(file string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	dir := filepath.Dir(file)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0o700); err != nil { // Cogpit's own directory: owner-only even if made wider before
		return err
	}
	tmp, err := os.CreateTemp(dir, ".discovery-*") // 0600
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }() // gone once renamed
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), file)
}

// definitionsSeen takes the digest of the cluster's custom resource definitions. It reports
// whether the answer the memory cache holds must be dropped: it was checked against other
// definitions, or read from a file that was never checked. An answer fetched in this connection and
// not checked yet takes the digest and is written again.
func (d *discoveryCache) definitionsSeen(digest string) (drop bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.defs = digest
	s := d.snap
	switch {
	case s == nil || s.defs == digest:
		return false
	case s.defs == "" && !s.fromFile:
		s.defs = digest
		d.store(s)
		return false
	}
	d.log.Info("cluster: the custom resource definitions changed; discovery is asked again at its next use", "context", d.cfg.context)
	return true
}

// info describes the answer the memory cache holds.
func (d *discoveryCache) info() DiscoveryInfo {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := DiscoveryInfo{Persisted: d.cfg.file != ""}
	s := d.snap
	if s == nil {
		return out
	}
	out.Source, out.FetchedAt, out.Checked, out.Partial = "server", s.fetched, s.defs != "", len(s.failed) > 0
	if s.fromFile {
		out.Source = "disk"
	}
	out.Groups = len(s.groups.Groups)
	for _, l := range s.resources {
		for _, r := range l.APIResources {
			if !strings.Contains(r.Name, "/") {
				out.Resources++
			}
		}
	}
	return out
}

// InvalidateWithContext drops the answer from memory and the file from disk; the rest of the
// connection asks the server. The memory cache's Invalidate calls it.
func (d *discoveryCache) InvalidateWithContext(context.Context) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.snap, d.skipFile = nil, true
	if d.cfg.file != "" {
		if err := os.Remove(d.cfg.file); err != nil && !errors.Is(err, os.ErrNotExist) {
			d.log.Warn("cluster: the discovery cache file cannot be removed", "context", d.cfg.context, "err", err)
		}
	}
}

// FreshWithContext reports whether the answer was asked of the server in this connection (or
// there is none yet).
func (d *discoveryCache) FreshWithContext(context.Context) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.snap == nil || !d.snap.fromFile
}

// ServerGroupsWithContext returns the groups (GroupsAndMaybeResourcesWithContext).
func (d *discoveryCache) ServerGroupsWithContext(ctx context.Context) (*metav1.APIGroupList, error) {
	g, _, _, err := d.GroupsAndMaybeResourcesWithContext(ctx)
	return g, err
}

// ServerGroupsAndResourcesWithContext returns every group and its resources through
// GroupsAndMaybeResourcesWithContext.
func (d *discoveryCache) ServerGroupsAndResourcesWithContext(ctx context.Context) ([]*metav1.APIGroup, []*metav1.APIResourceList, error) {
	return discovery.ServerGroupsAndResourcesWithContext(ctx, d)
}

// ServerPreferredResourcesWithContext returns the resources in their preferred versions.
func (d *discoveryCache) ServerPreferredResourcesWithContext(ctx context.Context) ([]*metav1.APIResourceList, error) {
	return discovery.ServerPreferredResourcesWithContext(ctx, d)
}

// ServerPreferredNamespacedResourcesWithContext returns the namespaced resources in their
// preferred versions.
func (d *discoveryCache) ServerPreferredNamespacedResourcesWithContext(ctx context.Context) ([]*metav1.APIResourceList, error) {
	return discovery.ServerPreferredNamespacedResourcesWithContext(ctx, d)
}

// WithLegacyWithContext returns the cache itself, as client-go's caches do.
func (d *discoveryCache) WithLegacyWithContext(context.Context) discovery.DiscoveryInterfaceWithContext {
	return d
}

// discoveryFileName is the cache file of a key in dir.
func discoveryFileName(dir, key string) string {
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, key[:32]+".json")
}

// errNoDiscovery is the error of a Conn without a discovery client.
func (c *Conn) errNoDiscovery() error {
	return fmt.Errorf("context %q: discovery is not available", c.contextName)
}
