package gocmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/taskgroup"
	gotool "github.com/lewtec/lewkit/x/toolchain/go"
)

func TestMain(m *testing.M) {
	if handled, err := gotool.Child(); handled {
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
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
			if !errors.Is(err, ErrCompiler) {
				t.Fatalf("got %v", err)
			}
		})
	}
}

func TestLewkitCompilerBuild(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("lewkit compiler is amd64")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/hello\n\ngo 1.27\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	src := "package main\nfunc main() { println(\"from-lewkit\") }\n"
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Run("linux", func(t *testing.T) {
		out := filepath.Join(dir, "hello")
		err := Command{
			Dir:      dir,
			Compiler: CompilerLewkit,
			Env:      append(os.Environ(), "CGO_ENABLED=0", "GOOS=linux", "GOARCH=amd64"),
			Args:     []string{"-trimpath", "-ldflags", "-s -w", "-o", out, "."},
		}.Run(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		got, err := exec.Command(out).CombinedOutput()
		if err != nil {
			t.Fatalf("%v\n%s", err, got)
		}
		if string(got) != "from-lewkit\n" {
			t.Fatalf("output %q", got)
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
		if err != nil {
			t.Fatal(err)
		}
		body, err := os.ReadFile(out)
		if err != nil {
			t.Fatal(err)
		}
		if len(body) < 2 || body[0] != 'M' || body[1] != 'Z' {
			t.Fatalf("not a PE: %q", body[:min(16, len(body))])
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
		if err != nil {
			t.Fatal(err)
		}
		typ, machine := elfHeader(t, out)
		if typ != 2 || machine != 183 {
			t.Fatalf("ELF type %d machine %d", typ, machine)
		}
	})
	t.Run("android-arm64", func(t *testing.T) {
		out := filepath.Join(dir, "libhello.so")
		tool := filepath.Join(dir, "toolexec")
		script := "#!/bin/sh\nif [ \"$(basename \"$1\")\" = \"_cgo_android_export.go\" ]; then\n\tshift\nfi\nexec \"$@\"\n"
		if err := os.WriteFile(tool, []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
		export := filepath.Join(dir, "_cgo_android_export.go")
		if err := os.WriteFile(export, []byte("package unused\n"), 0o644); err != nil {
			t.Fatal(err)
		}
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
		if err != nil {
			t.Fatal(err)
		}
		typ, machine := elfHeader(t, out)
		if typ != 3 || machine != 183 {
			t.Fatalf("ELF type %d machine %d", typ, machine)
		}
	})
}

func TestLewkitCompilerProgressTree(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("lewkit compiler is amd64")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/hello\n\ngo 1.27\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "p"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "p", "p.go"), []byte("package p\n\nfunc N() int { return 7 }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	src := "package main\n\nimport \"example.com/hello/p\"\n\nfunc main() { println(p.N()) }\n"
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
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
		t.Fatal(runErr)
	}
	if waitErr != nil {
		t.Fatal(waitErr)
	}
	if !sawControl || !sawCPU || !sawLink {
		t.Fatalf("progress tree control=%v cpu=%v link=%v", sawControl, sawCPU, sawLink)
	}
	got, err := exec.Command(out).CombinedOutput()
	if err != nil {
		t.Fatalf("%v\n%s", err, got)
	}
	if string(got) != "7\n" {
		t.Fatalf("output %q", got)
	}
}

func TestCgoExportArg(t *testing.T) {
	got, ok := cgoExportArg("'/tmp/wrap' '/tmp/_cgo_android_export.go'")
	if !ok || got != "/tmp/_cgo_android_export.go" {
		t.Fatalf("%q %v", got, ok)
	}
	got, ok = cgoExportArg("'/tmp/wrap' '/tmp/_cgo_clib_export.go'")
	if !ok || got != "/tmp/_cgo_clib_export.go" {
		t.Fatalf("clib %q %v", got, ok)
	}
	if _, ok := cgoExportArg("'/tmp/wrap' '/tmp/_cgo_other.go'"); ok {
		t.Fatal("accepted _cgo_other.go")
	}
	if _, ok := cgoExportArg("wrap"); ok {
		t.Fatal("accepted wrap")
	}
	dir := t.TempDir()
	missing := filepath.Join(dir, "_cgo_android_export.go")
	err := Command{
		Dir:      dir,
		Compiler: CompilerLewkit,
		Args:     []string{"-toolexec", "wrap " + missing, "-o", "out", "."},
	}.Run(t.Context())
	if !errors.Is(err, ErrCompiler) {
		t.Fatalf("got %v", err)
	}
}

func elfHeader(t *testing.T, path string) (typ, machine uint16) {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(body) < 20 || string(body[:4]) != "\x7fELF" || body[4] != 2 || body[5] != 1 {
		t.Fatalf("not ELF64 LE")
	}
	return uint16(body[16]) | uint16(body[17])<<8, uint16(body[18]) | uint16(body[19])<<8
}

func TestLewkitCompilerFromContext(t *testing.T) {
	err := Command{Args: []string{"-buildmode=c-archive", "-o", "x", "."}}.Run(WithCompiler(t.Context(), CompilerLewkit))
	if !errors.Is(err, ErrCompiler) {
		t.Fatalf("got %v", err)
	}
}

func withEnv(c Command, extra string) Command {
	c.Env = append(os.Environ(), extra)
	return c
}
