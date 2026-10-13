package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/guided-traffic/cogpit/internal/app"
	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/demo"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
)

type writer = io.Writer

// env is what a command runs with: its streams, its logger and Cogpit's directories. main fills
// it from the process; tests give every field their own (temporary directories, buffers), so that
// no test reads the developer's configuration or kubeconfig.
type env struct {
	stdin  io.ReadCloser
	stdout io.Writer
	stderr io.Writer
	logger *slog.Logger
	level  *slog.LevelVar // -v sets it to debug; nil leaves the logger as it is

	// dirs are Cogpit's directories (app.UserDirs); dirsErr says why there are none.
	dirs    app.Dirs
	dirsErr error

	// demo sizes the synthetic demo cluster (the context "demo"); the zero value is the app's
	// default size.
	demo demo.Options

	// mcpTransport is what `mcp` serves on; nil means stdin and stdout.
	mcpTransport sdk.Transport
}

// flags is a subcommand's flag set. Flags and arguments may be mixed (`get pods -n x`), as kubectl
// allows.
type flags struct {
	*flag.FlagSet
	name, synopsis, about string
	positional            []string
}

func newFlags(name, synopsis, about string) *flags {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard) // errors and help are printed by parse
	fs.Usage = func() {}
	return &flags{FlagSet: fs, name: name, synopsis: synopsis, about: about}
}

// parse parses args, flags and arguments in any order. done is true when the command must not
// run: --help (exit 0, the help on stdout) or a usage error (exit 2, on stderr).
func (f *flags) parse(args []string, e *env) (code int, done bool) {
	for {
		if err := f.Parse(args); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				f.printHelp(e.stdout)
				return exitOK, true
			}
			return f.usageError(e, err.Error()), true
		}
		args = f.Args()
		if len(args) == 0 {
			return exitOK, false
		}
		f.positional = append(f.positional, args[0])
		args = args[1:]
	}
}

func (f *flags) usageError(e *env, msg string) int {
	fmt.Fprintf(e.stderr, "cogpit-cli %s: %s\n", f.name, msg)
	fmt.Fprintf(e.stderr, "Run 'cogpit-cli %s --help' for usage.\n", f.name)
	return exitUsage
}

func (f *flags) printHelp(w io.Writer) {
	fmt.Fprintf(w, "Usage:\n  cogpit-cli %s", f.name)
	if f.synopsis != "" {
		fmt.Fprint(w, " "+f.synopsis)
	}
	fmt.Fprintf(w, "\n\n%s\n", f.about)
	n := 0
	f.VisitAll(func(*flag.Flag) { n++ })
	if n > 0 {
		fmt.Fprint(w, "\nFlags:\n")
		f.SetOutput(w)
		f.PrintDefaults()
		f.SetOutput(io.Discard)
	}
}

// common are the flags every command that reads a kubeconfig takes.
type common struct {
	kubeconfig string
	verbose    bool
}

func (c *common) register(f *flags) {
	f.StringVar(&c.kubeconfig, "kubeconfig", "", "kubeconfig files, separated by the OS path list separator, offered as sources of their own for this run (as the app's --kubeconfig); default: the files $KUBECONFIG lists. ~/.kube and the sources added in Cogpit are offered either way")
	f.BoolVar(&c.verbose, "v", false, "debug logging on stderr")
}

// apply sets the log level.
func (c *common) apply(e *env) {
	if c.verbose && e.level != nil {
		e.level.Set(slog.LevelDebug)
	}
}

// sessionOptions configure a headless session.
type sessionOptions struct {
	// kubeconfig is the --kubeconfig flag's value.
	kubeconfig string
	// audit writes the agent-initiated commands to Cogpit's audit log (the MCP bridge).
	audit bool
}

// newSession builds a headless engine: no window, so no confirmer (a command that needs the
// user's confirmation is refused), the same kubeconfig sources and credential plugin approvals
// as the app (app.Dirs.KubeconfigSources), the demo cluster as the context "demo", every command
// registered. No drain or Helm journal and no hotbar: their commands are refused or disabled. The
// returned function closes the session.
func (e *env) newSession(opts sessionOptions) (*engine.Session, func(), error) {
	if e.dirsErr != nil || e.dirs.Config == "" {
		// The app falls back to merging the kubeconfig files as kubectl does, which runs their
		// credential plugins unasked. The CLI does not: without the approvals there is no
		// connection.
		return nil, nil, fmt.Errorf("the configuration directory is unknown (%v): without it there are no kubeconfig "+
			"sources and no credential plugin approvals, so cogpit-cli does not connect", e.dirsErr)
	}
	named := cluster.NamedKubeconfigFiles(filepath.SplitList(opts.kubeconfig))
	connector := &engine.ClusterConnector{
		IncludeDemo: true,
		DemoOptions: e.demo,
		Logger:      e.logger,
		Sources:     e.dirs.KubeconfigSources(named),
		// The app's cluster settings, read only: a context's proxy applies to the CLI's connection too.
		ClusterSettings: engine.NewClusterSettingsStore(e.dirs.Config),
	}
	so := engine.Options{Connector: connector, Logger: e.logger}
	closers := []func(){}
	if opts.audit {
		f, err := e.dirs.OpenAuditLog()
		if err != nil {
			e.logger.Warn("audit log disabled", "err", err)
		} else {
			so.Audit = f
			closers = append(closers, func() { _ = f.Close() })
		}
	}
	s := engine.New(so)
	if err := commands.RegisterAll(s.Registry()); err != nil {
		s.Close()
		return nil, nil, err
	}
	return s, func() {
		s.Close()
		for _, c := range closers {
			c()
		}
	}, nil
}

// dispatch runs one engine command with origin CLI and returns its result as JSON, the very
// bytes an MCP client receives for the tool.
func dispatch(ctx context.Context, s *engine.Session, id string, params any) (json.RawMessage, error) {
	var raw json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}
		raw = b
	}
	out, err := s.Dispatch(ctx, engine.Invocation{ID: id, Params: raw, Origin: engine.OriginCLI})
	if err != nil {
		return nil, err
	}
	return json.Marshal(out)
}

// dispatchInto runs a command like dispatch and decodes its result into out.
func dispatchInto(ctx context.Context, s *engine.Session, id string, params, out any) (json.RawMessage, error) {
	data, err := dispatch(ctx, s, id, params)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, out); err != nil {
		return nil, fmt.Errorf("%s: unexpected result: %w", id, err)
	}
	return data, nil
}

// fail prints an error and returns exitError. Engine errors carry no credential
// (kubeconfig.SafeError); the text is made safe for a terminal all the same.
func fail(e *env, err error) int {
	fmt.Fprintf(e.stderr, "cogpit-cli: %s\n", safeText(err.Error()))
	return exitError
}
