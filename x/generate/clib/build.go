package clib

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	gotool "github.com/lewtec/lewkit/x/toolchain/go"
)

// Build links dir as a cgo-free shared library for this process.
// The package must already contain the files Run wrote.
func Build(ctx context.Context, dir, out string) error {
	return BuildFor(ctx, dir, out, runtime.GOOS, runtime.GOARCH)
}

// BuildFor links dir for goos/goarch. The host table decides the pair.
// A pair with no internal PIE, and a generated export whose build
// constraint omits the pair, return ErrHost before the linker. amd64 and
// arm64 link through the embedded compiler as an internal PIE. windows/386
// uses the host go command, which has the 386 backend. The clibexport tag
// is set. Linux and android then clear DF_1_PIE and drop PT_INTERP.
// Windows sets the DLL bit and replaces the entry with a DllMain that
// returns TRUE. Darwin rewrites the image as a dylib and installs an
// export trie. The patched image must contain every export name.
// arm uses GOARM=7.
func BuildFor(ctx context.Context, dir, out, goos, goarch string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	host, err := hostFor(goos, goarch)
	if err != nil {
		return err
	}
	if !host.link {
		return fmt.Errorf("%w: %s/%s has no internal PIE shared library", ErrHost, goos, goarch)
	}
	if dir == "" || out == "" {
		return ErrClibDirectoryAndOutputAreRequired
	}
	dir, err = filepath.Abs(dir)
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
	if err := exportCovers(export, goos, goarch); err != nil {
		return err
	}
	names, err := readExports(dir)
	if err != nil {
		return err
	}
	if strings.Contains(export, " ") || strings.Contains(out, " ") {
		return ErrClibPathsMustNotContainSpaces
	}
	tool, cleanup, err := buildTool(ctx, dir)
	if err != nil {
		return err
	}
	defer cleanup()
	if goarch == "386" {
		err = linkHost(ctx, dir, out, goos, goarch, tool, export)
	} else {
		err = linkCompiler(ctx, dir, out, goos, goarch, tool, export)
	}
	if err != nil {
		return err
	}
	switch goos {
	case "windows":
		if err := MakeDLL(out); err != nil {
			return err
		}
	case "darwin":
		if err := MakeDylib(out, names); err != nil {
			return err
		}
	default:
		if err := MakeShared(out); err != nil {
			return err
		}
	}
	return requireImageExports(out, goos, names)
}

func buildTool(ctx context.Context, dir string) (string, func(), error) {
	toolDir, err := os.MkdirTemp("", "clibtoolexec")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { os.RemoveAll(toolDir) }
	name := "clibtoolexec"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	tool := filepath.Join(toolDir, name)
	if err := runGo(ctx, dir, hostEnv(), "build", "-buildvcs=false", "-o", tool, "github.com/lewtec/lewkit/x/build/clibtoolexec"); err != nil {
		cleanup()
		return "", nil, err
	}
	if strings.Contains(tool, " ") {
		cleanup()
		return "", nil, ErrClibToolPathMustNotContainSpaces
	}
	return tool, cleanup, nil
}

func linkHost(ctx context.Context, dir, out, goos, goarch, tool, export string) error {
	return runGo(ctx, dir, linkEnv(goos, goarch), "build",
		"-buildmode=pie",
		"-tags=clibexport",
		"-buildvcs=false",
		"-ldflags=-linkmode=internal -checklinkname=0",
		"-toolexec="+tool+" "+export,
		"-o", out,
		".")
}

func linkCompiler(ctx context.Context, dir, out, goos, goarch, tool, export string) error {
	return gotool.BuildModule(ctx, gotool.Module{
		Dir:      dir,
		Out:      out,
		GOOS:     goos,
		GOARCH:   goarch,
		Tags:     []string{"clibexport"},
		Ldflags:  []string{"-checklinkname=0"},
		Toolexec: tool + " " + export,
		Library:  true,
	})
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
		return nil, ErrClibNoExports
	}
	return names, nil
}

func runGo(ctx context.Context, dir string, env []string, args ...string) error {
	cmd := execdriver.MustCommand(ctx, "go", args...)
	cmd.Dir = dir
	cmd.Env = env
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := execdriver.Run(ctx, cmd)
	if err != nil {
		text := strings.TrimSpace(out.String())
		if text == "" {
			return fmt.Errorf("%w %s: %w", ErrRunGoClibGo2, args[0], err)
		}
		return fmt.Errorf("%w %s: %w\n%s", ErrRunGoClibGo, args[0], err, text)
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
