// Command generate builds glslang.wasm.
//
// glslang.wasm depends on libs.
// libs depends on configure.
// configure depends on source and tools.
// source fetches the glslang release. tools writes a wasm-opt stub that
// emcc will accept, and puts uv's Python on PATH.
// source and tools do not wait for each other.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/lewtec/lewkit/x/cmd"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/lewkit/x/tool"
	"github.com/lewtec/lewkit/x/workflow"
)

const glslangVersion = "15.1.0"

func main() {
	app, err := cmd.Parse[cmd.App[generateCmd]](os.Args[1:]...)
	if err != nil {
		entry.Main(context.Background(), func(context.Context) error { return err })
		return
	}
	parent := taskgroup.WithLimits(context.Background(), app.Args.Apply(taskgroup.DefaultLimits()))
	entry.MainFrom(parent, app.Run)
}

type generateCmd struct {
	taskgroup.Arg `flatten:"" ctx:"taskgroup"`
}

func (generateCmd) Description() string {
	return "build glslang.wasm"
}

func (generateCmd) Run(ctx context.Context) error {
	here, err := sourceDir()
	if err != nil {
		return err
	}
	work := filepath.Join(os.TempDir(), "lewkit-glslang-wasm")
	out := filepath.Join(here, "..", "wasm", "glslang.wasm")
	graph := wasmGraph(here, work, out)
	fut, err := workflow.Run(ctx, graph, nil)
	if err != nil {
		return err
	}
	return fut.Wait(ctx)
}

func sourceDir() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("glslang: source dir")
	}
	// compile.c sits next to this command's parent directory.
	return filepath.Dir(filepath.Dir(file)), nil
}

// wasmGraph is the glslang reactor build.
// glslang.wasm depends on libs.
// libs depends on configure.
// configure depends on source and tools.
func wasmGraph(here, work, out string) workflow.Graph {
	box := &wasmBox{here: here, work: work, out: out}
	return workflow.Graph{
		Dir:      here,
		Defaults: []string{"glslang.wasm"},
		Steps: []workflow.Step{
			{
				Name: "tools",
				Desc: "wasm-opt stub and python",
				Pool: taskgroup.IO,
				Tasks: []workflow.Task{
					workflow.Func(func(ctx context.Context, st *taskgroup.Status) error {
						return box.tools(ctx, st)
					}),
				},
			},
			{
				Name: "source",
				Desc: "glslang " + glslangVersion,
				Pool: taskgroup.Internet,
				Tasks: []workflow.Task{
					workflow.Func(func(ctx context.Context, st *taskgroup.Status) error {
						return box.source(ctx, st)
					}),
				},
			},
			{
				Name: "configure",
				Deps: []string{"source", "tools"},
				Desc: "configure after source and tools",
				Pool: taskgroup.IO,
				Tasks: []workflow.Task{
					workflow.Func(func(ctx context.Context, st *taskgroup.Status) error {
						return box.configure(ctx, st)
					}),
				},
			},
			{
				Name: "libs",
				Deps: []string{"configure"},
				Desc: "static libs after configure",
				Pool: taskgroup.IO,
				Tasks: []workflow.Task{
					workflow.Func(func(ctx context.Context, st *taskgroup.Status) error {
						return box.libs(ctx, st)
					}),
				},
			},
			{
				Name: "glslang.wasm",
				Deps: []string{"libs"},
				Desc: "glslang.wasm after libs",
				Pool: taskgroup.IO,
				Tasks: []workflow.Task{
					workflow.Func(func(ctx context.Context, st *taskgroup.Status) error {
						return box.link(ctx, st)
					}),
				},
			},
		},
	}
}

// wasmBox is the shared state of one glslang build.
// A step reads a field only after the step that writes it has finished.
type wasmBox struct {
	here       string
	work       string
	out        string
	env        []string
	srcroot    string
	build      string
	staticLibs []string
}

func (b *wasmBox) tools(ctx context.Context, st *taskgroup.Status) error {
	if err := os.MkdirAll(filepath.Join(b.work, "bin"), 0o755); err != nil {
		return err
	}
	stub := filepath.Join(b.work, "bin", "wasm-opt")
	if err := os.WriteFile(stub, []byte(wasmOptStub), 0o755); err != nil {
		return err
	}
	py, err := commandOutput(ctx, "", "uv", "python", "find", "3.13")
	if err != nil {
		return fmt.Errorf("uv python: %w", err)
	}
	py = strings.TrimSpace(py)
	if py == "" {
		return errors.New("uv python: empty path")
	}
	path := filepath.Join(b.work, "bin") + string(os.PathListSeparator) + filepath.Dir(py) + string(os.PathListSeparator) + os.Getenv("PATH")
	env := setEnv(os.Environ(), "PATH", path)
	env = setEnv(env, "EM_BINARYEN_ROOT", b.work)
	b.env = env
	st.Update(stub)
	return nil
}

func (b *wasmBox) source(ctx context.Context, st *taskgroup.Status) error {
	if err := os.MkdirAll(b.work, 0o755); err != nil {
		return err
	}
	b.srcroot = filepath.Join(b.work, "glslang-"+glslangVersion)
	if info, err := os.Stat(b.srcroot); err == nil && info.IsDir() {
		st.Update(b.srcroot)
		return nil
	}
	tarball := filepath.Join(b.work, "glslang-"+glslangVersion+".tar.gz")
	url := "https://github.com/KhronosGroup/glslang/archive/refs/tags/" + glslangVersion + ".tar.gz"
	if err := tool.DownloadFile(ctx, url, tarball, tool.DownloadOptions{}); err != nil {
		return err
	}
	if err := os.RemoveAll(b.srcroot); err != nil {
		return err
	}
	// Extract strips the top directory, so the tree lands in srcroot.
	if err := tool.Extract(ctx, tarball, b.srcroot); err != nil {
		return err
	}
	st.Update(b.srcroot)
	return nil
}

func (b *wasmBox) configure(ctx context.Context, st *taskgroup.Status) error {
	b.build = filepath.Join(b.work, "build")
	if err := os.MkdirAll(b.build, 0o755); err != nil {
		return err
	}
	cache := filepath.Join(b.build, "CMakeCache.txt")
	if _, err := os.Stat(cache); err == nil {
		st.Update(cache)
		return nil
	}
	st.Update(b.build)
	return mise(ctx, b.env, b.here, "emcmake", "cmake",
		"-S", b.srcroot,
		"-B", b.build,
		"-G", "Ninja",
		"-DCMAKE_BUILD_TYPE=Release",
		"-DENABLE_OPT=OFF",
		"-DENABLE_HLSL=OFF",
		"-DENABLE_GLSLANG_BINARIES=OFF",
		"-DENABLE_SPVREMAPPER=OFF",
		"-DBUILD_TESTING=OFF",
		"-DENABLE_GLSLANG_JS=OFF",
	)
}

func (b *wasmBox) libs(ctx context.Context, st *taskgroup.Status) error {
	st.Update(b.build)
	if err := mise(ctx, b.env, b.here, "emmake", "cmake", "--build", b.build, "--parallel"); err != nil {
		return err
	}
	var libs []string
	for _, name := range []string{
		"glslang", "SPIRV", "MachineIndependent", "GenericCodeGen",
		"glslang-default-resource-limits", "OSDependent", "SPIRV-Tools-static", "SPIRV-Tools",
	} {
		var found string
		err := filepath.WalkDir(b.build, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || found != "" {
				return err
			}
			if d.Name() == "lib"+name+".a" {
				found = path
			}
			return nil
		})
		if err != nil {
			return err
		}
		if found != "" {
			libs = append(libs, found)
		}
	}
	if len(libs) == 0 {
		return fmt.Errorf("no glslang static libs in %s", b.build)
	}
	b.staticLibs = libs
	st.Update(fmt.Sprintf("%d libs", len(libs)))
	return nil
}

func (b *wasmBox) link(ctx context.Context, st *taskgroup.Status) error {
	st.Update(b.out)
	args := []string{
		"emcc", filepath.Join(b.here, "compile.c"), "-O2",
		"-I", b.srcroot,
	}
	args = append(args, b.staticLibs...)
	args = append(args,
		"--no-entry",
		"--no-wasm-opt",
		"-sSTANDALONE_WASM=1",
		"-sALLOW_MEMORY_GROWTH=1",
		"-sINITIAL_MEMORY=67108864",
		"-sEXPORTED_FUNCTIONS=_compile_compute,_compile_stage,_last_error,_malloc,_free",
		"-o", b.out,
	)
	return mise(ctx, b.env, b.here, args...)
}

func mise(ctx context.Context, env []string, dir string, args ...string) error {
	cmdArgs := append([]string{"exec", "conda:emscripten@4.0.9", "conda:cmake@4.4.3", "conda:ninja@1.13.2", "--"}, args...)
	return commandRun(ctx, env, dir, "mise", cmdArgs...)
}

func commandRun(ctx context.Context, env []string, dir, name string, args ...string) error {
	cmd := execdriver.MustCommand(ctx, name, args...)
	cmd.Dir = dir
	if env != nil {
		cmd.Env = env
	}
	out := taskgroup.LineWriterFrom(ctx)
	cmd.Stdout = out
	cmd.Stderr = out
	return execdriver.Run(ctx, cmd)
}

func commandOutput(ctx context.Context, dir, name string, args ...string) (string, error) {
	cmd := execdriver.MustCommand(ctx, name, args...)
	cmd.Dir = dir
	out, err := execdriver.Output(ctx, cmd)
	return string(out), err
}

func setEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i, line := range env {
		if strings.HasPrefix(line, prefix) {
			next := append([]string(nil), env...)
			next[i] = prefix + value
			return next
		}
	}
	return append(env, prefix+value)
}

// conda emscripten's wasm-opt is too old for emcc 4.0.9, so tools plants
// a stub that reports a new version and otherwise copies the input.
const wasmOptStub = `#!/bin/sh
if [ "$1" = "--version" ]; then
	echo "wasm-opt version 123"
	exit 0
fi
out="" src=""
while [ $# -gt 0 ]; do
	if [ "$1" = "-o" ]; then out=$2; shift 2; continue; fi
	case $1 in
	--*) shift ;;
	*) src=$1; shift ;;
	esac
done
if [ -n "$out" ] && [ -n "$src" ] && [ "$out" != "$src" ]; then cp "$src" "$out"; fi
exit 0
`
