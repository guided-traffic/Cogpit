package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/kubeconfig"
)

// Kubeconfig sources (G2, docs/developer/architecture.md §6): kubeconfig files the user adds,
// and sync folders whose kubeconfig files Cogpit picks up as they appear. Each file is read on its own, never merged with another, so
// a context in one file can never borrow the user or cluster entry of another (a dropped file
// naming the user's real user entry with its own server). An exec credential plugin of a source
// runs only after the user allowed it in the GUI (trust-boundaries H-1; ExecApproval).
//
// Cogpit reads the files the user adds and the files in sync folders; it never writes them. It
// writes only its own files under its configuration directory: the list of sources and the
// approvals, each owner-only.

// Files and directories under Cogpit's configuration directory.
const (
	// KubeconfigSourcesFile lists the sources the user added.
	KubeconfigSourcesFile = "kubeconfigs.json"
	// ExecApprovalsFile holds the credential plugins the user allowed.
	ExecApprovalsFile = "exec-approvals.json"
	// DefaultKubeconfigPoll is how often WatchKubeconfigs looks at the sources for changes.
	DefaultKubeconfigPoll = 2 * time.Second
)

// legacyPastedKind is the kind of the sources earlier builds wrote for pasted kubeconfigs.
const legacyPastedKind SourceKind = "pasted"

// SourceKind is what a kubeconfig source is.
type SourceKind string

// Source kinds.
const (
	SourceFile   SourceKind = "file"   // a kubeconfig file the user added; read, never written
	SourceFolder SourceKind = "folder" // a sync folder: every kubeconfig file in it, as files come and go
)

// Approval states of an exec credential plugin (ContextInfo.Exec, ExecPluginInfo.State).
const (
	ExecApproved = "approved" // the user allowed exactly this plugin for this context and file
	ExecRequired = "required" // the user has not allowed it yet
	ExecChanged  = "changed"  // the user allowed it, and it changed since (command, arguments, environment or server)
)

// ErrExecNotApproved marks a refused connection: the context's credential plugin may run only
// after the user allowed it in Cogpit's window. Match it with errors.Is.
var ErrExecNotApproved = errors.New("the credential plugin is not allowed")

// ErrGUIOnly refuses the kubeconfig commands that only the user's own action in Cogpit's window may
// run: adding a source, removing a source, allowing a credential plugin.
var ErrGUIOnly = errors.New("only the user, in Cogpit's own window, can do this; agents and the CLI are refused " +
	"(a kubeconfig can name a program to run on this computer)")

// KubeconfigSource is one source the user added.
type KubeconfigSource struct {
	// ID names the source in context ids, unique among the sources: a file's name ("dev.yaml"),
	// or a folder's name with a slash ("kubeconfigs/").
	ID   string     `json:"id"`
	Kind SourceKind `json:"kind"`
	// Path is the file or folder, absolute.
	Path  string    `json:"path"`
	Added time.Time `json:"added"`
}

// ExecApproval is the user's allowance of one credential plugin for one context of one file.
type ExecApproval struct {
	File    string `json:"file"`
	Context string `json:"context"`
	// Fingerprint binds the plugin exactly (kubeconfig.ExecPlugin.Fingerprint): a change asks again.
	Fingerprint string    `json:"fingerprint"`
	Command     string    `json:"command"`
	Approved    time.Time `json:"approved"`
}

// ExecApprovalError refuses a connection whose credential plugin the user has not allowed.
type ExecApprovalError struct {
	Plugin *kubeconfig.ExecPlugin
	State  string // ExecRequired or ExecChanged
}

func (e *ExecApprovalError) Error() string {
	cmd := kubeconfig.DisplayText(e.Plugin.Command, 200)
	file := kubeconfig.DisplayText(e.Plugin.File, 400)
	if e.State == ExecChanged {
		return fmt.Sprintf("the credential plugin %s of %s changed since you allowed it; Cogpit runs it only after "+
			"you review and allow it again: connect from the context picker in Cogpit's window", cmd, file)
	}
	return fmt.Sprintf("the kubeconfig %s runs the program %s to log in, which you have not allowed yet; Cogpit runs it "+
		"only after you review and allow it: connect from the context picker in Cogpit's window", file, cmd)
}

// Is makes errors.Is(err, ErrExecNotApproved) true.
func (e *ExecApprovalError) Is(target error) bool { return target == ErrExecNotApproved }

// SourceContextID is the id of a context from a source file: "<context> [<label>]". Context ids
// of the default kubeconfig files are the bare context names.
func SourceContextID(context, label string) string { return context + " [" + label + "]" }

// KubeconfigFile is one kubeconfig file of a source, read on its own.
type KubeconfigFile struct {
	Source KubeconfigSource
	// Label names the file in its contexts' ids: the source's id, or the folder's id followed by
	// the file's name ("kubeconfigs/team.yaml").
	Label string
	Path  string
	// Contexts and Exec are as cluster.InspectFile returned them; nil when Err is set.
	Contexts []kubeconfig.Context
	Exec     map[string]*kubeconfig.ExecPlugin
	Err      error
}

// SourceStatus is a source with what reading it found, for kubeconfig.sources.
type SourceStatus struct {
	KubeconfigSource
	// Files are a folder's kubeconfig files (their labels), or the source's one file.
	Files []string `json:"files,omitempty"`
	// Contexts counts the contexts the source offers.
	Contexts int `json:"contexts"`
	// Skipped counts a folder's files that are not kubeconfigs or define no context.
	Skipped int `json:"skipped,omitempty"`
	// Error says why the source (or its one file) cannot be read.
	Error string `json:"error,omitempty"`
}

// maxPrivateDoc bounds the sources and approvals files: they hold a few entries.
const maxPrivateDoc = 1 << 20

const sourcesVersion, approvalsVersion = 1, 1

type sourcesDoc struct {
	Version int                `json:"version"`
	Sources []KubeconfigSource `json:"sources"`
}

type approvalsDoc struct {
	Version   int            `json:"version"`
	Approvals []ExecApproval `json:"approvals"`
}

// KubeconfigSources keeps the sources the user added and the credential plugins the user allowed,
// in files under one directory (the app's: <user config dir>/Cogpit). It is safe for concurrent
// use; two Cogpit processes changing the files at once may lose one change (the files are not
// locked).
type KubeconfigSources struct {
	dir string
	now func() time.Time
	mu  sync.Mutex // serialises the read-modify-write of the files
}

// NewKubeconfigSources returns the sources kept in dir. Nothing is read or written until used.
func NewKubeconfigSources(dir string) *KubeconfigSources {
	return &KubeconfigSources{dir: dir, now: time.Now}
}

// Dir returns the directory the sources live in.
func (k *KubeconfigSources) Dir() string { return k.dir }

func (k *KubeconfigSources) sourcesPath() string   { return filepath.Join(k.dir, KubeconfigSourcesFile) }
func (k *KubeconfigSources) approvalsPath() string { return filepath.Join(k.dir, ExecApprovalsFile) }

// List returns the sources in the order they were added.
func (k *KubeconfigSources) List() ([]KubeconfigSource, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.loadSources()
}

func (k *KubeconfigSources) loadSources() ([]KubeconfigSource, error) {
	var doc sourcesDoc
	if err := readPrivateJSON(k.sourcesPath(), maxPrivateDoc, &doc); err != nil {
		return nil, fmt.Errorf("the kubeconfig sources file %s is unreadable (%v); Cogpit leaves it as it is", k.sourcesPath(), err)
	}
	if doc.Version != 0 && doc.Version != sourcesVersion {
		return nil, fmt.Errorf("the kubeconfig sources file %s has format version %d, this Cogpit reads %d", k.sourcesPath(), doc.Version, sourcesVersion)
	}
	list := make([]KubeconfigSource, 0, len(doc.Sources))
	for _, s := range doc.Sources {
		if s.Kind == legacyPastedKind {
			// Pasting was removed (D-21). An entry written by an earlier build is no longer offered; its
			// file under kubeconfigs/ is the user's and stays on disk, and the next save of the list
			// drops the entry.
			continue
		}
		if s.ID == "" || !filepath.IsAbs(s.Path) || (s.Kind != SourceFile && s.Kind != SourceFolder) {
			return nil, fmt.Errorf("the kubeconfig sources file %s has an entry without an id, an absolute path or a known kind", k.sourcesPath())
		}
		list = append(list, s)
	}
	return list, nil
}

func (k *KubeconfigSources) saveSources(list []KubeconfigSource) error {
	if list == nil {
		list = []KubeconfigSource{}
	}
	data, err := json.MarshalIndent(sourcesDoc{Version: sourcesVersion, Sources: list}, "", "  ")
	if err != nil {
		return err
	}
	if err := ensurePrivateDir(k.dir); err != nil {
		return fmt.Errorf("kubeconfig sources: %w", err)
	}
	if err := writePrivateFile(k.sourcesPath(), append(data, '\n')); err != nil {
		return fmt.Errorf("kubeconfig sources: %w", err)
	}
	return nil
}

// AddFile adds a kubeconfig file as a source: Cogpit reads it, follows its changes and never
// writes it. The file must be a kubeconfig that defines a context. It returns the source and the
// file's contexts.
func (k *KubeconfigSources) AddFile(path string) (KubeconfigSource, []kubeconfig.Context, error) {
	abs, err := absClean(path)
	if err != nil {
		return KubeconfigSource{}, nil, err
	}
	f, err := cluster.InspectFile(abs)
	if err != nil {
		return KubeconfigSource{}, nil, err
	}
	if len(f.Contexts) == 0 {
		return KubeconfigSource{}, nil, fmt.Errorf("kubeconfig file %s: %w", abs, kubeconfig.ErrNoContexts)
	}
	src, err := k.add(SourceFile, abs, filepath.Base(abs))
	return src, f.Contexts, err
}

// AddFolder adds a sync folder: every kubeconfig file in it (not in its subfolders) becomes
// available, and files that appear or disappear later are picked up or dropped.
func (k *KubeconfigSources) AddFolder(path string) (KubeconfigSource, error) {
	abs, err := absClean(path)
	if err != nil {
		return KubeconfigSource{}, err
	}
	fi, err := os.Stat(abs)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return KubeconfigSource{}, fmt.Errorf("folder %s does not exist", abs)
		}
		return KubeconfigSource{}, fmt.Errorf("folder %s: %w", abs, err)
	}
	if !fi.IsDir() {
		return KubeconfigSource{}, fmt.Errorf("%s is not a folder; add a single file with Add Kubeconfig File", abs)
	}
	return k.add(SourceFolder, abs, filepath.Base(abs)+"/")
}

// add appends a file or folder source, unless one of the same kind and path exists.
func (k *KubeconfigSources) add(kind SourceKind, path, base string) (KubeconfigSource, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	list, err := k.loadSources()
	if err != nil {
		return KubeconfigSource{}, err
	}
	for _, s := range list {
		if s.Kind == kind && s.Path == path {
			return KubeconfigSource{}, fmt.Errorf("%s is added already (as %q)", path, s.ID)
		}
	}
	id := uniqueID(list, base)
	src := KubeconfigSource{ID: id, Kind: kind, Path: path, Added: k.now().UTC()}
	if err := k.saveSources(append(list, src)); err != nil {
		return KubeconfigSource{}, err
	}
	return src, nil
}

// uniqueID returns base, or base with " (2)", " (3)", … before a trailing slash, whichever no
// source has.
func uniqueID(list []KubeconfigSource, base string) string {
	taken := func(id string) bool {
		return slices.ContainsFunc(list, func(s KubeconfigSource) bool { return s.ID == id })
	}
	if !taken(base) {
		return base
	}
	stem, slash := strings.CutSuffix(base, "/")
	for n := 2; ; n++ {
		id := stem + " (" + strconv.Itoa(n) + ")"
		if slash {
			id += "/"
		}
		if !taken(id) {
			return id
		}
	}
}

// Remove removes a source. The file or folder the user added stays as it is. The approvals of the source's files are forgotten.
func (k *KubeconfigSources) Remove(id string) (KubeconfigSource, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	list, err := k.loadSources()
	if err != nil {
		return KubeconfigSource{}, err
	}
	i := slices.IndexFunc(list, func(s KubeconfigSource) bool { return s.ID == id })
	if i < 0 {
		return KubeconfigSource{}, fmt.Errorf("no kubeconfig source %q", id)
	}
	src := list[i]
	if err := k.saveSources(slices.Delete(list, i, i+1)); err != nil {
		return KubeconfigSource{}, err
	}
	if err := k.forget(func(a ExecApproval) bool { return sourceHas(src, a.File) }); err != nil {
		return src, err
	}
	return src, nil
}

// sourceHas reports whether file is the source's file or a file of its folder.
func sourceHas(src KubeconfigSource, file string) bool {
	if src.Kind == SourceFolder {
		return filepath.Dir(file) == src.Path
	}
	return file == src.Path
}

// Files reads every source: one KubeconfigFile per file source, one per
// kubeconfig file in a sync folder (files that are not kubeconfigs or define no context are left
// out and counted in the statuses). The error reports a sources file that cannot be read.
func (k *KubeconfigSources) Files() ([]KubeconfigFile, []SourceStatus, error) {
	list, err := k.List()
	if err != nil {
		return nil, nil, err
	}
	var files []KubeconfigFile
	statuses := make([]SourceStatus, 0, len(list))
	for _, src := range list {
		st := SourceStatus{KubeconfigSource: src}
		switch src.Kind {
		case SourceFolder:
			entries, err := cluster.ListSyncFolder(src.Path)
			if err != nil {
				st.Error = err.Error()
				break
			}
			for _, e := range entries {
				f := inspect(src, strings.TrimSuffix(src.ID, "/")+"/"+filepath.Base(e.Path), e.Path)
				if f.Err != nil || len(f.Contexts) == 0 {
					st.Skipped++
					continue
				}
				files = append(files, f)
				st.Files = append(st.Files, f.Label)
				st.Contexts += len(f.Contexts)
			}
		default:
			f := inspect(src, src.ID, src.Path)
			files = append(files, f)
			st.Files = []string{f.Label}
			if f.Err != nil {
				st.Error = f.Err.Error()
			}
			st.Contexts = len(f.Contexts)
		}
		statuses = append(statuses, st)
	}
	return files, statuses, nil
}

func inspect(src KubeconfigSource, label, path string) KubeconfigFile {
	f := KubeconfigFile{Source: src, Label: label, Path: path}
	in, err := cluster.InspectFile(path)
	if err != nil {
		f.Err = err
		return f
	}
	f.Contexts, f.Exec = in.Contexts, in.Exec
	return f
}

// Find returns the source file and the context name of a source context id.
func (k *KubeconfigSources) Find(id string) (KubeconfigFile, string, bool, error) {
	if !strings.HasSuffix(id, "]") {
		return KubeconfigFile{}, "", false, nil
	}
	files, _, err := k.Files()
	if err != nil {
		return KubeconfigFile{}, "", false, err
	}
	for _, f := range files {
		for _, c := range f.Contexts {
			if SourceContextID(c.Name, f.Label) == id {
				return f, c.Name, true, nil
			}
		}
	}
	return KubeconfigFile{}, "", false, nil
}

// Contexts returns the contexts of every source file as context infos, sorted by id, and an error
// naming each file that could not be read (a sync folder's files that are not kubeconfigs are not
// errors).
func (k *KubeconfigSources) Contexts() ([]ContextInfo, error) {
	files, _, err := k.Files()
	if err != nil {
		return nil, err
	}
	approvals, aErr := k.approvals()
	var out []ContextInfo
	var problems []error
	if aErr != nil {
		problems = append(problems, aErr)
	}
	for _, f := range files {
		if f.Err != nil {
			problems = append(problems, fmt.Errorf("kubeconfig source %q: %w", f.Source.ID, f.Err))
			continue
		}
		for _, c := range f.Contexts {
			info := ContextInfo{
				Name: SourceContextID(c.Name, f.Label), Cluster: c.Cluster, User: c.User, Namespace: c.Namespace,
				Server: c.Server, Source: f.Label, KubeContext: c.Name, File: f.Path,
			}
			if p := f.Exec[c.Name]; p != nil {
				info.Exec = approvalState(approvals, p)
			}
			out = append(out, info)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, errors.Join(problems...)
}

// approvals reads the approvals file.
func (k *KubeconfigSources) approvals() ([]ExecApproval, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.loadApprovals()
}

func (k *KubeconfigSources) loadApprovals() ([]ExecApproval, error) {
	var doc approvalsDoc
	if err := readPrivateJSON(k.approvalsPath(), maxPrivateDoc, &doc); err != nil {
		return nil, fmt.Errorf("the credential plugin approvals file %s is unreadable (%v); Cogpit allows no plugin "+
			"until it is fixed or moved aside", k.approvalsPath(), err)
	}
	if doc.Version != 0 && doc.Version != approvalsVersion {
		return nil, fmt.Errorf("the credential plugin approvals file %s has format version %d, this Cogpit reads %d",
			k.approvalsPath(), doc.Version, approvalsVersion)
	}
	return doc.Approvals, nil
}

func (k *KubeconfigSources) saveApprovals(list []ExecApproval) error {
	if list == nil {
		list = []ExecApproval{}
	}
	data, err := json.MarshalIndent(approvalsDoc{Version: approvalsVersion, Approvals: list}, "", "  ")
	if err != nil {
		return err
	}
	if err := writePrivateFile(k.approvalsPath(), append(data, '\n')); err != nil {
		return fmt.Errorf("credential plugin approvals: %w", err)
	}
	return nil
}

// approvalState is the state of plugin p against the approvals.
func approvalState(approvals []ExecApproval, p *kubeconfig.ExecPlugin) string {
	state := ExecRequired
	fp := p.Fingerprint()
	for _, a := range approvals {
		if a.File != p.File || a.Context != p.Context {
			continue
		}
		if fp != "" && a.Fingerprint == fp {
			return ExecApproved
		}
		state = ExecChanged
	}
	return state
}

// ExecState returns whether plugin p may run: ExecApproved, ExecRequired or ExecChanged. An
// unreadable approvals file allows nothing.
func (k *KubeconfigSources) ExecState(p *kubeconfig.ExecPlugin) (string, error) {
	approvals, err := k.approvals()
	if err != nil {
		return ExecRequired, err
	}
	return approvalState(approvals, p), nil
}

// Approve records the user's allowance of plugin p for its file and context, replacing an earlier
// one for them.
func (k *KubeconfigSources) Approve(p *kubeconfig.ExecPlugin) error {
	fp := p.Fingerprint()
	if fp == "" {
		return errors.New("credential plugin approvals: the plugin has no fingerprint")
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	list, err := k.loadApprovals()
	if err != nil {
		return err
	}
	list = slices.DeleteFunc(list, func(a ExecApproval) bool { return a.File == p.File && a.Context == p.Context })
	list = append(list, ExecApproval{File: p.File, Context: p.Context, Fingerprint: fp, Command: p.Command, Approved: k.now().UTC()})
	return k.saveApprovals(list)
}

// forget drops the approvals drop matches.
func (k *KubeconfigSources) forget(drop func(ExecApproval) bool) error {
	list, err := k.loadApprovals()
	if err != nil {
		return err
	}
	kept := slices.DeleteFunc(slices.Clone(list), drop)
	if len(kept) == len(list) {
		return nil
	}
	return k.saveApprovals(kept)
}

// Policy is the cluster.Options.ExecPolicy of the sources' contexts: it refuses a plugin the user
// has not allowed, or allowed in another form, with an *ExecApprovalError.
func (k *KubeconfigSources) Policy(p *kubeconfig.ExecPlugin) error {
	state, err := k.ExecState(p)
	if err != nil {
		return err
	}
	if state != ExecApproved {
		return &ExecApprovalError{Plugin: p, State: state}
	}
	return nil
}

// signature describes the sources' files as stat sees them (size, modification time), without
// reading them: WatchKubeconfigs compares it between polls.
func (k *KubeconfigSources) signature() string {
	list, err := k.List()
	if err != nil {
		return "error " + err.Error()
	}
	var b strings.Builder
	stat := func(p string) {
		fi, err := os.Stat(p)
		if err != nil {
			fmt.Fprintf(&b, "%s missing\n", p)
			return
		}
		fmt.Fprintf(&b, "%s %d %d\n", p, fi.Size(), fi.ModTime().UnixNano())
	}
	for _, s := range list {
		fmt.Fprintf(&b, "%s %s %s\n", s.ID, s.Kind, s.Path)
		if s.Kind != SourceFolder {
			stat(s.Path)
			continue
		}
		files, err := cluster.ListSyncFolder(s.Path)
		if err != nil {
			fmt.Fprintf(&b, "%s unreadable\n", s.Path)
			continue
		}
		for _, f := range files {
			fmt.Fprintf(&b, "%s %d %d\n", f.Path, f.Size, f.ModTime)
		}
	}
	return b.String()
}

// absClean returns path made absolute and clean; an empty path is an error.
func absClean(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errors.New("no path given")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}
