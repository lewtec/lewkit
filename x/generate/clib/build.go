package clib

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Build links dir as a cgo-free shared library for this process.
// The package must already contain the files Run wrote.
func Build(ctx context.Context, dir, out string) error {
	return BuildFor(ctx, dir, out, runtime.GOOS, runtime.GOARCH)
}

// BuildFor links dir for goos/goarch. The link is an internal PIE with the
// clibexport tag. Linux and android then clear DF_1_PIE and drop PT_INTERP.
// Windows sets the DLL bit and replaces the entry with a DllMain that
// returns TRUE. Darwin rewrites the image as a dylib and installs an
// export trie. arm uses GOARM=7.
func BuildFor(ctx context.Context, dir, out, goos, goarch string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := targetsFor(goos + "/" + goarch); err != nil {
		return err
	}
	if dir == "" || out == "" {
		return fmt.Errorf("clib: directory and output are required")
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	out, err = filepath.Abs(out)
	if err != nil {
		return err
	}
	export := filepath.Join(dir, exportFileName)
	if _, err := os.Stat(export); err != nil {
		return err
	}
	if strings.Contains(export, " ") || strings.Contains(out, " ") {
		return fmt.Errorf("clib: paths must not contain spaces")
	}
	toolDir, err := os.MkdirTemp("", "clibtoolexec")
	if err != nil {
		return err
	}
	defer os.RemoveAll(toolDir)
	tool := filepath.Join(toolDir, "clibtoolexec")
	if err := runGo(ctx, dir, hostEnv(), "build", "-buildvcs=false", "-o", tool, "github.com/lewtec/lewkit/x/build/clibtoolexec"); err != nil {
		return err
	}
	if strings.Contains(tool, " ") {
		return fmt.Errorf("clib: tool path must not contain spaces")
	}
	if err := runGo(ctx, dir, linkEnv(goos, goarch), "build",
		"-buildmode=pie",
		"-tags=clibexport",
		"-buildvcs=false",
		"-ldflags=-linkmode=internal -checklinkname=0",
		"-toolexec="+tool+" "+export,
		"-o", out,
		"."); err != nil {
		return err
	}
	switch goos {
	case "windows":
		return MakeDLL(out)
	case "darwin":
		names, err := readExports(dir)
		if err != nil {
			return err
		}
		return MakeDylib(out, names)
	default:
		return MakeShared(out)
	}
}

func readExports(dir string) ([]string, error) {
	buf, err := os.ReadFile(filepath.Join(dir, exportFileName))
	if err != nil {
		return nil, err
	}
	var names []string
	for _, line := range strings.Split(string(buf), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "//go:cgo_export_dynamic" {
			names = append(names, fields[1])
		}
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("clib: no exports")
	}
	return names, nil
}

func runGo(ctx context.Context, dir string, env []string, args ...string) error {
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = dir
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		text := strings.TrimSpace(string(out))
		if text == "" {
			return fmt.Errorf("clib: go %s: %w", args[0], err)
		}
		return fmt.Errorf("clib: go %s: %w\n%s", args[0], err, text)
	}
	return nil
}

func hostEnv() []string {
	return filterEnv(nil, false)
}

func linkEnv(goos, goarch string) []string {
	extra := []string{"CGO_ENABLED=0", "GOOS=" + goos, "GOARCH=" + goarch}
	if goarch == "arm" {
		extra = append(extra, "GOARM=7")
	}
	return filterEnv(extra, true)
}

func filterEnv(extra []string, dropTune bool) []string {
	out := make([]string, 0, len(os.Environ())+len(extra))
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "CGO_ENABLED=") || strings.HasPrefix(e, "GOOS=") || strings.HasPrefix(e, "GOARCH=") || strings.HasPrefix(e, "CC=") {
			continue
		}
		if dropTune && (strings.HasPrefix(e, "GOARM=") || strings.HasPrefix(e, "GO386=")) {
			continue
		}
		out = append(out, e)
	}
	return append(out, extra...)
}
