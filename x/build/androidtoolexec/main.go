// Command androidtoolexec is the go -toolexec wrapper for a cgo-free
// Android library. It adds x/entry/android/_cgo_android_export.go when
// compiling that package, so the linker writes JNI_OnLoad into .dynsym.
// The arm64 cgo-free build passes -ldflags=-checklinkname=0 with this wrapper.
//
// cmd/go ignores a file whose name starts with _, and it hashes the
// compiler's -V=full line into the compile cache key. A wrapper that
// forwards that line unchanged shares the cache with a build that never
// saw the export file, and the next link ships a library with no
// JNI_OnLoad. The build passes the export file as the first argument.
// Its hash is appended to the tool version.
package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/lewtec/lewkit/x/build/internal/toolexec"
)

const (
	entryPkg = "github.com/lewtec/lewkit/x/entry/android"
	// exportCacheGen is appended to the compiler version. Bump it when the
	// wrapper's compile behavior changes and the export file bytes do not.
	exportCacheGen = "1"
)

func main() {
	args := os.Args[1:]
	args, export := peelExport(args)
	if len(args) == 0 {
		os.Exit(2)
	}
	if toolexec.VersionProbe(args) {
		toolexec.PrintVersion("androidtoolexec", args, export, func(path string) (string, error) {
			return toolexec.HashStamp("lewkitexport=", exportCacheGen, path)
		})
		return
	}
	if toolexec.ToolName(args[0]) == "compile" && androidArm64Nocgo() {
		flat := toolexec.Flatten(args[1:])
		if pkgOf(flat) == entryPkg {
			if export == "" {
				export = exportFile(flat)
			}
			if export == "" {
				fmt.Fprintln(os.Stderr, "androidtoolexec: _cgo_android_export.go not found next to x/entry/android")
				os.Exit(1)
			}
			if _, err := os.Stat(export); err != nil {
				fmt.Fprintln(os.Stderr, "androidtoolexec:", err)
				os.Exit(1)
			}
			args = append(args, export)
		}
	}
	os.Exit(runTool(args, os.Stderr))
}

// runTool runs the real compile or link. Combined output is copied to stderr.
// Inheriting the go command's merged stdout and stderr handles has exited 1
// on Windows with an empty buffer, so the child gets its own pipes.
func runTool(args []string, stderr io.Writer) int {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if len(out) > 0 {
		stderr.Write(out)
	}
	if err == nil {
		return 0
	}
	if ee, ok := err.(*exec.ExitError); ok {
		if len(out) == 0 {
			fmt.Fprintf(stderr, "androidtoolexec: %s: exit status %d\n", filepath.Base(args[0]), ee.ExitCode())
		}
		return ee.ExitCode()
	}
	fmt.Fprintln(stderr, err)
	return 1
}

// peelExport splits a leading _cgo_android_export.go path from the tool args.
// The go command splits -toolexec on spaces and then appends the tool.
func peelExport(args []string) (rest []string, export string) {
	if len(args) > 0 && filepath.Base(args[0]) == "_cgo_android_export.go" {
		return args[1:], args[0]
	}
	return args, ""
}

func androidArm64Nocgo() bool {
	return os.Getenv("GOOS") == "android" &&
		os.Getenv("GOARCH") == "arm64" &&
		os.Getenv("CGO_ENABLED") == "0"
}

func pkgOf(args []string) string {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-p" {
			return args[i+1]
		}
	}
	return ""
}

func exportFile(args []string) string {
	for _, arg := range args {
		if !strings.HasSuffix(arg, ".go") || strings.HasPrefix(filepath.Base(arg), "_") {
			continue
		}
		dir := filepath.Dir(arg)
		candidate := filepath.Join(dir, "_cgo_android_export.go")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}
