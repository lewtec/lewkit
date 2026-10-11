//go:build linux && amd64

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

// Build links dir as a cgo-free shared library at out.
// The package must already contain the files Run wrote. The link is an
// internal PIE with the clibexport tag. MakeShared then clears DF_1_PIE
// and drops PT_INTERP so glibc will dlopen the file.
func Build(ctx context.Context, dir, out string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return ErrHost
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
	if err := runGo(ctx, dir, linkEnv(), "build",
		"-buildmode=pie",
		"-tags=clibexport",
		"-buildvcs=false",
		"-ldflags=-linkmode=internal -checklinkname=0",
		"-toolexec="+tool+" "+export,
		"-o", out,
		"."); err != nil {
		return err
	}
	return MakeShared(out)
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
	return filterEnv(nil)
}

func linkEnv() []string {
	return filterEnv([]string{"CGO_ENABLED=0", "GOOS=linux", "GOARCH=amd64"})
}

func filterEnv(extra []string) []string {
	out := make([]string, 0, len(os.Environ())+len(extra))
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "CGO_ENABLED=") || strings.HasPrefix(e, "GOOS=") || strings.HasPrefix(e, "GOARCH=") || strings.HasPrefix(e, "CC=") {
			continue
		}
		out = append(out, e)
	}
	return append(out, extra...)
}
