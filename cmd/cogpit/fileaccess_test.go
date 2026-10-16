package main

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/ui/nativedialog"
)

// At start a kubeconfig source's bookmark stays while the path is a source, a folder's and a CA
// file's stay as they are; without sources (no configuration directory) every bookmark stays.
func TestKeptSource(t *testing.T) {
	dir := t.TempDir()
	folder := filepath.Join(dir, "team")
	if err := os.Mkdir(folder, 0o700); err != nil {
		t.Fatal(err)
	}
	sources := engine.NewKubeconfigSources(filepath.Join(dir, "Cogpit"))
	if _, err := sources.AddFolder(folder); err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	keep := keptSource(sources, logger)
	for _, tc := range []struct {
		e    nativedialog.Entry
		want bool
	}{
		{nativedialog.Entry{Use: nativedialog.UseKubeconfig, Path: folder}, true},
		{nativedialog.Entry{Use: nativedialog.UseKubeconfig, Path: folder + "/"}, true},
		{nativedialog.Entry{Use: nativedialog.UseKubeconfig, Path: filepath.Join(dir, "removed.yaml")}, false},
		{nativedialog.Entry{Use: nativedialog.UseFolder, Path: filepath.Join(dir, "work")}, true},
		{nativedialog.Entry{Use: nativedialog.UseCAFile, Path: filepath.Join(dir, "ca.pem")}, true},
	} {
		if got := keep(tc.e); got != tc.want {
			t.Errorf("keep(%+v) = %v, want %v", tc.e, got, tc.want)
		}
	}
	if keptSource(nil, logger) != nil {
		t.Error("without sources a bookmark was judged")
	}
}
