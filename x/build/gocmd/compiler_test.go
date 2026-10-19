package gocmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/lewkit/x/test"
	gotool "github.com/lewtec/lewkit/x/toolchain/go"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	gotool.TestExit(m)
}

func TestLewkitCompilerRejects(t *testing.T) {
	dir := t.TempDir()
	base := Command{Dir: dir, Compiler: CompilerLewkit, Args: []string{"-o", "out", "."}}
	cases := []struct {
		name string
		cmd  Command
	}{
		{name: "cgo", cmd: withEnv(base, "CGO_ENABLED=1")},
		{name: "386", cmd: withEnv(base, "GOARCH=386")},
		{name: "c-shared", cmd: Command{Dir: dir, Compiler: CompilerLewkit, Args: []string{"-buildmode=c-shared", "-o", "out", "."}}},
		{name: "toolexec", cmd: Command{Dir: dir, Compiler: CompilerLewkit, Args: []string{"-toolexec", "wrap", "-o", "out", "."}}},
		{name: "overlay", cmd: Command{Dir: dir, Compiler: CompilerLewkit, Args: []string{"-overlay", "o.json", "-o", "out", "."}}},
		{name: "run", cmd: Command{Verb: "run", Dir: dir, Compiler: CompilerLewkit, Args: []string{"."}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cmd.Run(t.Context())
			require.ErrorIs(t, err, ErrCompiler)
		})
	}
}

func TestLewkitCompilerBuild(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		test.Skip(t, "lewkit compiler is amd64")
	}
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/hello\n\ngo 1.27\n"), 0o644))
	src := "package main\nfunc main() { println(\"from-lewkit\") }\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte(src), 0o644))
	t.Run("linux", func(t *testing.T) {
		out := filepath.Join(dir, "hello")
		err := Command{
			Dir:      dir,
			Compiler: CompilerLewkit,
			Env:      append(os.Environ(), "CGO_ENABLED=0", "GOOS=linux", "GOARCH=amd64"),
			Args:     []string{"-trimpath", "-ldflags", "-s -w", "-o", out, "."},
		}.Run(t.Context())
		require.NoError(t, err)
		if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" {
			got, err := exec.Command(out).CombinedOutput()
			require.NoErrorf(t, err, "%v\n%s", err, got)
			if string(got) != "from-lewkit\n" {
				require.FailNowf(t, "", "output %q", got)
			}
			return
		}
		typ, machine := elfHeader(t, out)
		if typ != 2 || machine != 62 {
			require.FailNowf(t, "", "ELF type %d machine %d", typ, machine)
		}
	})
	t.Run("windows", func(t *testing.T) {
		out := filepath.Join(dir, "hello.exe")
		err := Command{
			Dir:      dir,
			Compiler: CompilerLewkit,
			Env:      append(os.Environ(), "CGO_ENABLED=0", "GOOS=windows", "GOARCH=amd64"),
			Args:     []string{"-trimpath", "-ldflags", "-H windowsgui -s -w", "-o", out, "."},
		}.Run(t.Context())
		require.NoError(t, err)
		body, err := os.ReadFile(out)
		require.NoError(t, err)
		if len(body) < 2 || body[0] != 'M' || body[1] != 'Z' {
			require.FailNowf(t, "", "not a PE: %q", body[:min(16, len(body))])
		}
	})
	t.Run("linux-arm64", func(t *testing.T) {
		out := filepath.Join(dir, "hello-arm64")
		err := Command{
			Dir:      dir,
			Compiler: CompilerLewkit,
			Env:      append(os.Environ(), "CGO_ENABLED=0", "GOOS=linux", "GOARCH=arm64"),
			Args:     []string{"-trimpath", "-o", out, "."},
		}.Run(t.Context())
		require.NoError(t, err)
		typ, machine := elfHeader(t, out)
		if typ != 2 || machine != 183 {
			require.FailNowf(t, "", "ELF type %d machine %d", typ, machine)
		}
	})
	t.Run("android-arm64", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			test.Skip(t, "toolexec wrapper is a shell script")
		}
		out := filepath.Join(dir, "libhello.so")
		tool := writeToolexec(t, dir)
		export := filepath.Join(dir, "_cgo_android_export.go")
		require.NoError(t, os.WriteFile(export, []byte("package unused\n"), 0o644))
		err := Command{
			Dir:      dir,
			Compiler: CompilerLewkit,
			Env:      append(os.Environ(), "CGO_ENABLED=0", "GOOS=android", "GOARCH=arm64"),
			Args: []string{
				"-trimpath",
				"-tags", "appmode",
				"-toolexec", tool + " " + export,
				"-ldflags", "-checklinkname=0",
				"-o", out, ".",
			},
		}.Run(t.Context())
		require.NoError(t, err)
		typ, machine := elfHeader(t, out)
		if typ != 3 || machine != 183 {
			require.FailNowf(t, "", "ELF type %d machine %d", typ, machine)
		}
	})
}

func TestLewkitCompilerProgressTree(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		test.Skip(t, "lewkit compiler is amd64")
	}
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/hello\n\ngo 1.27\n"), 0o644))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "p"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "p", "p.go"), []byte("package p\n\nfunc N() int { return 7 }\n"), 0o644))
	src := "package main\n\nimport \"example.com/hello/p\"\n\nfunc main() { println(p.N()) }\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte(src), 0o644))
	out := filepath.Join(dir, "hello")
	base, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	sess, ctx := taskgroup.New(base, taskgroup.Limits{CPU: 1})
	sawControl := false
	sawCPU := false
	sawLink := false
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			nodes := sess.List(64)
			byID := make(map[taskgroup.ID]taskgroup.Node, len(nodes))
			for _, n := range nodes {
				byID[n.ID] = n
				if n.Name == "go build" && n.Pool == taskgroup.Control {
					sawControl = true
				}
			}
			for _, n := range nodes {
				parent, ok := byID[n.Parent]
				if ok && parent.Name == "go build" && n.Pool == taskgroup.CPU {
					sawCPU = true
					if n.Name == gotool.LinkStep {
						sawLink = true
					}
				}
			}
			if sawControl && sawCPU && sawLink {
				return
			}
			select {
			case <-time.After(15 * time.Millisecond):
			case <-stop:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
	var runErr error
	taskgroup.Go(ctx, "release", taskgroup.Control, func(ctx context.Context, _ *taskgroup.Status) error {
		runErr = Command{
			Dir:      dir,
			Compiler: CompilerLewkit,
			Env:      append(os.Environ(), "CGO_ENABLED=0", "GOOS=linux", "GOARCH=amd64"),
			Args:     []string{"-trimpath", "-o", out, "."},
		}.Run(ctx)
		return runErr
	})
	waitErr := sess.Wait()
	close(stop)
	<-done
	if runErr != nil {
		require.FailNow(t, fmt.Sprint(runErr))
	}
	if waitErr != nil {
		require.FailNow(t, fmt.Sprint(waitErr))
	}
	if !sawControl || !sawCPU || !sawLink {
		require.FailNowf(t, "", "progress tree control=%v cpu=%v link=%v", sawControl, sawCPU, sawLink)
	}
	if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" {
		got, err := exec.Command(out).CombinedOutput()
		require.NoErrorf(t, err, "%v\n%s", err, got)
		if string(got) != "7\n" {
			require.FailNowf(t, "", "output %q", got)
		}
		return
	}
	typ, machine := elfHeader(t, out)
	if typ != 2 || machine != 62 {
		require.FailNowf(t, "", "ELF type %d machine %d", typ, machine)
	}
}

func TestCgoExportArg(t *testing.T) {
	got, ok := cgoExportArg("'/tmp/wrap' '/tmp/_cgo_android_export.go'")
	if !ok || got != "/tmp/_cgo_android_export.go" {
		require.FailNowf(t, "", "%q %v", got, ok)
	}
	got, ok = cgoExportArg("'/tmp/wrap' '/tmp/_cgo_clib_export.go'")
	if !ok || got != "/tmp/_cgo_clib_export.go" {
		require.FailNowf(t, "", "clib %q %v", got, ok)
	}
	if _, ok := cgoExportArg("'/tmp/wrap' '/tmp/_cgo_other.go'"); ok {
		require.FailNow(t, fmt.Sprint("accepted _cgo_other.go"))
	}
	if _, ok := cgoExportArg("wrap"); ok {
		require.FailNow(t, fmt.Sprint("accepted wrap"))
	}
	dir := t.TempDir()
	missing := filepath.Join(dir, "_cgo_android_export.go")
	err := Command{
		Dir:      dir,
		Compiler: CompilerLewkit,
		Args:     []string{"-toolexec", "wrap " + missing, "-o", "out", "."},
	}.Run(t.Context())
	require.ErrorIs(t, err, ErrCompiler)
}

func writeToolexec(t *testing.T, dir string) string {
	t.Helper()
	tool := filepath.Join(dir, "toolexec")
	if runtime.GOOS == "windows" {
		tool += ".exe"
		wrap := filepath.Join(dir, "wrap")
		require.NoError(t, os.MkdirAll(wrap, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(wrap, "go.mod"), []byte("module wrap\n\ngo 1.27.0\n"), 0o644))
		body := "package main\n\nimport (\n\t\"os\"\n\t\"os/exec\"\n\t\"path/filepath\"\n)\n\nfunc main() {\n\targs := os.Args[1:]\n\tif len(args) > 0 && filepath.Base(args[0]) == \"_cgo_android_export.go\" {\n\t\targs = args[1:]\n\t}\n\tif len(args) == 0 {\n\t\tos.Exit(2)\n\t}\n\tcmd := exec.Command(args[0], args[1:]...)\n\tcmd.Stdout = os.Stdout\n\tcmd.Stderr = os.Stderr\n\tif err := cmd.Run(); err != nil {\n\t\tos.Exit(1)\n\t}\n}\n"
		require.NoError(t, os.WriteFile(filepath.Join(wrap, "main.go"), []byte(body), 0o644))
		cmd := exec.Command("go", "build", "-o", tool, ".")
		cmd.Dir = wrap
		out, err := cmd.CombinedOutput()
		require.NoErrorf(t, err, "%s", out)
		return tool
	}
	script := "#!/bin/sh\nif [ \"$(basename \"$1\")\" = \"_cgo_android_export.go\" ]; then\n\tshift\nfi\nexec \"$@\"\n"
	require.NoError(t, os.WriteFile(tool, []byte(script), 0o755))
	return tool
}

func elfHeader(t *testing.T, path string) (typ, machine uint16) {
	t.Helper()
	body, err := os.ReadFile(path)
	require.NoError(t, err)
	if len(body) < 20 || string(body[:4]) != "\x7fELF" || body[4] != 2 || body[5] != 1 {
		require.FailNowf(t, "", "not ELF64 LE")
	}
	return uint16(body[16]) | uint16(body[17])<<8, uint16(body[18]) | uint16(body[19])<<8
}

func TestLewkitCompilerFromContext(t *testing.T) {
	err := Command{Args: []string{"-buildmode=c-archive", "-o", "x", "."}}.Run(WithCompiler(t.Context(), CompilerLewkit))
	require.ErrorIs(t, err, ErrCompiler)
}

func withEnv(c Command, extra string) Command {
	c.Env = append(os.Environ(), extra)
	return c
}
