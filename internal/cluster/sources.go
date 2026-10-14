package cluster

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/guided-traffic/cogpit/internal/kubeconfig"
	"github.com/guided-traffic/cogpit/internal/model"
)

// MaxSyncFolderFiles bounds how many files of one sync folder are read: a folder with more is
// not a folder of kubeconfigs (a Downloads folder), and reading it on every poll would cost.
const MaxSyncFolderFiles = 500

// InspectedFile is one kubeconfig file read on its own, as Connect reads it when it is the only
// file of KubeconfigPaths: its contexts and the exec credential plugins their users run.
type InspectedFile struct {
	// Path is the file's absolute path.
	Path string
	// Contexts are the file's contexts, sorted by name.
	Contexts []kubeconfig.Context
	// Exec holds the exec plugin of each context whose user runs one, by context name, described
	// as client-go will run it (a relative command resolved against the file's directory).
	Exec map[string]*kubeconfig.ExecPlugin
}

// InspectFile reads and parses one kubeconfig file without merging it with any other and without
// running anything. A file larger than kubeconfig.MaxFileSize, not a regular file, or not a
// kubeconfig is an error; its text never echoes a credential.
func InspectFile(path string) (*InspectedFile, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("kubeconfig file %s: %w", path, err)
	}
	data, err := readBounded(abs)
	if err != nil {
		return nil, err
	}
	cfg, err := kubeconfig.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("kubeconfig file %s: %w", abs, err)
	}
	// As client-go's loader does for a file: every entry remembers its file, and relative paths
	// (certificates, token files, an exec command with a separator) resolve against its directory.
	for _, ai := range cfg.AuthInfos {
		if ai != nil {
			ai.LocationOfOrigin = abs
		}
	}
	for _, cl := range cfg.Clusters {
		if cl != nil {
			cl.LocationOfOrigin = abs
		}
	}
	for _, kc := range cfg.Contexts {
		if kc != nil {
			kc.LocationOfOrigin = abs
		}
	}
	if err := clientcmd.ResolveLocalPaths(cfg); err != nil {
		return nil, fmt.Errorf("kubeconfig file %s: %w", abs, kubeconfig.SafeError(err))
	}
	out := &InspectedFile{Path: abs, Contexts: kubeconfig.FromConfig(cfg), Exec: map[string]*kubeconfig.ExecPlugin{}}
	for _, c := range out.Contexts {
		if p := BindPluginFiles(kubeconfig.ExecPluginOf(cfg, c.Name)); p != nil {
			out.Exec[c.Name] = p
		}
	}
	return out, nil
}

// readBounded reads a regular file of at most kubeconfig.MaxFileSize bytes.
func readBounded(path string) ([]byte, error) {
	// Looked at before it is opened: opening a FIFO for reading blocks until something writes to
	// it. The check after the open (below) still guards against a swap in between, which the
	// folder listing and a user-chosen path make unlikely.
	if pre, err := os.Stat(path); err == nil && !pre.Mode().IsRegular() {
		return nil, fmt.Errorf("kubeconfig file %s is not a regular file", path)
	}
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("kubeconfig file %s does not exist", path)
		}
		return nil, fmt.Errorf("kubeconfig file %s cannot be read: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	fi, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("kubeconfig file %s cannot be read: %w", path, err)
	}
	if !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("kubeconfig file %s is not a regular file", path)
	}
	if fi.Size() > kubeconfig.MaxFileSize {
		return nil, fmt.Errorf("kubeconfig file %s is larger than a kubeconfig (at most %s)", path, model.FormatBytes(kubeconfig.MaxFileSize))
	}
	data, err := io.ReadAll(io.LimitReader(f, kubeconfig.MaxFileSize+1))
	if err != nil {
		return nil, fmt.Errorf("kubeconfig file %s cannot be read: %w", path, err)
	}
	if len(data) > kubeconfig.MaxFileSize {
		return nil, fmt.Errorf("kubeconfig file %s grew past %s while being read", path, model.FormatBytes(kubeconfig.MaxFileSize))
	}
	return data, nil
}

// FolderFile is a file in a sync folder that may be a kubeconfig, with what its change detection
// compares.
type FolderFile struct {
	Path    string
	Size    int64
	ModTime int64 // UnixNano
}

// ListSyncFolder lists the files of dir, not its subdirectories, that may be kubeconfigs: regular
// files (a symlink counts as the file it points to) whose names pass kubeconfig.SyncCandidate and
// whose size is at most kubeconfig.MaxFileSize, sorted by name, at most MaxSyncFolderFiles. It
// reads no file's content.
func ListSyncFolder(dir string) ([]FolderFile, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("sync folder %s: %w", dir, err)
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("sync folder %s does not exist", abs)
		}
		return nil, fmt.Errorf("sync folder %s cannot be read: %w", abs, err)
	}
	var out []FolderFile
	for _, e := range entries {
		if !kubeconfig.SyncCandidate(e.Name()) {
			continue
		}
		p := filepath.Join(abs, e.Name())
		fi, err := os.Stat(p)
		if err != nil || !fi.Mode().IsRegular() || fi.Size() > kubeconfig.MaxFileSize {
			continue
		}
		out = append(out, FolderFile{Path: p, Size: fi.Size(), ModTime: fi.ModTime().UnixNano()})
		if len(out) == MaxSyncFolderFiles {
			break
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

// checkExecPolicy asks policy about the login cfg will use for the context name (kctx in raw) when
// it needs approval: the exec plugin cfg will run, and a connection that does not check the
// server's certificate (A28). The plugin is described from cfg.ExecProvider, the section the
// client's transport runs, the check from cfg's own TLS settings, and raw, loaded together with it,
// names its file and cluster.
func checkExecPolicy(policy func(*kubeconfig.ExecPlugin) error, raw clientcmdapi.Config, name string, kctx *clientcmdapi.Context, cfg *rest.Config) error {
	if policy == nil || cfg == nil || kctx == nil || (cfg.ExecProvider == nil && !cfg.Insecure) {
		return nil
	}
	if ai := raw.AuthInfos[kctx.AuthInfo]; ai != nil && ai.TokenFile != "" {
		return nil // checkTokenFilePolicy decided, with the exec section and the cluster, before the file was read
	}
	if cfg.Insecure { // the cluster as the client was built from it: its check off, whatever raw says
		raw = withInsecureCluster(raw, kctx)
	}
	return askPolicy(policy, raw, name, kctx, cfg.ExecProvider, "")
}

// withInsecureCluster returns raw with the cluster of kctx marked insecure-skip-tls-verify, a copy
// where it was not: the policy then judges the connection as it will be made.
func withInsecureCluster(raw clientcmdapi.Config, kctx *clientcmdapi.Context) clientcmdapi.Config {
	cl := raw.Clusters[kctx.Cluster]
	if cl != nil && cl.InsecureSkipTLSVerify {
		return raw
	}
	marked := clientcmdapi.NewCluster()
	if cl != nil {
		marked = cl.DeepCopy()
	}
	marked.InsecureSkipTLSVerify = true
	clusters := make(map[string]*clientcmdapi.Cluster, len(raw.Clusters)+1)
	for k, v := range raw.Clusters {
		clusters[k] = v
	}
	clusters[kctx.Cluster] = marked
	raw.Clusters = clusters
	return raw
}

// checkTokenFilePolicy asks policy about the token file the user entry of context name in raw
// names, if it has one, together with its exec section. It runs before client-go builds the
// rest.Config, which reads the token file: nothing is read from a file outside the kubeconfig's
// folder before the user allowed it (D-17).
func checkTokenFilePolicy(policy func(*kubeconfig.ExecPlugin) error, raw clientcmdapi.Config, name string, kctx *clientcmdapi.Context) error {
	if policy == nil || kctx == nil {
		return nil
	}
	ai := raw.AuthInfos[kctx.AuthInfo]
	if ai == nil || ai.TokenFile == "" {
		return nil
	}
	return askPolicy(policy, raw, name, kctx, ai.Exec, ai.TokenFile)
}

// askPolicy describes the login of context name, looks at the files it names right now, right
// before the policy decides (a program or CA file swapped since the user allowed it is caught,
// R-5, and so is a token file moved outside the folder), and asks policy unless nothing needs
// approval.
func askPolicy(policy func(*kubeconfig.ExecPlugin) error, raw clientcmdapi.Config, name string, kctx *clientcmdapi.Context, ec *clientcmdapi.ExecConfig, tokenFile string) error {
	file := kctx.LocationOfOrigin // a context without a user entry still has its file (A28)
	if ai := raw.AuthInfos[kctx.AuthInfo]; ai != nil {
		file = ai.LocationOfOrigin
	}
	p := BindPluginFiles(kubeconfig.NewLogin(file, name, kctx.AuthInfo, raw.Clusters[kctx.Cluster], ec, tokenFile))
	if p == nil {
		return nil
	}
	return policy(p)
}

// LookPathNow returns where an exec plugin's command would be found now: the command itself when
// it is a path, else the first match on PATH; empty when there is none. It is for showing the
// user; client-go looks the command up again when the plugin runs.
func LookPathNow(command string) string {
	if command == "" {
		return ""
	}
	p, err := exec.LookPath(command)
	if err != nil {
		return ""
	}
	if abs, err := filepath.Abs(p); err == nil {
		return abs
	}
	return p
}
