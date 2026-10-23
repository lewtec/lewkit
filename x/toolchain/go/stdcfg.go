package gotool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lewtec/lewkit/x/toolchain/load"
)

// runtimeDeps builds the runtime archive closure with the embedded compiler.
// compileCfg is the importcfg for package main. linkDeps is the packagefile
// lines for the link. The build mode is the platform default. A shared
// library on Linux still receives the non-PIE runtime archives that go list
// produced for a default executable.
func runtimeDeps(ctx context.Context, dir string) (compileCfg, linkDeps string, err error) {
	if ctx.Err() != nil {
		return "", "", context.Cause(ctx)
	}
	if err = os.MkdirAll(dir, 0o755); err != nil {
		return "", "", err
	}
	depsPath := filepath.Join(dir, "deps")
	compileCfg = filepath.Join(dir, "importcfg")
	if body, readErr := os.ReadFile(depsPath); readErr == nil {
		return compileCfg, string(body), nil
	}
	work, err := os.MkdirTemp("", "lewkit-stdlib-")
	if err != nil {
		return "", "", err
	}
	defer os.RemoveAll(work)
	m := Module{GOOS: hostGOOS(), GOARCH: hostGOARCH()}
	archives, err := carriedArchives(ctx, work, "", m, &load.Graph{})
	if err != nil {
		return "", "", err
	}
	runtimeArchive := archives["runtime"]
	if runtimeArchive == "" {
		return "", "", ErrRuntimeHasNoExportArchive
	}
	keys := make([]string, 0, len(archives))
	for path := range archives {
		keys = append(keys, path)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, path := range keys {
		fmt.Fprintf(&b, "packagefile %s=%s\n", path, archives[path])
	}
	if err = os.WriteFile(compileCfg, []byte("# import config\npackagefile runtime="+runtimeArchive+"\n"), 0o644); err != nil {
		return "", "", err
	}
	if err = os.WriteFile(depsPath, []byte(b.String()), 0o644); err != nil {
		return "", "", err
	}
	return compileCfg, b.String(), nil
}
