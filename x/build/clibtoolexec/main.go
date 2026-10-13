// Command clibtoolexec is the go -toolexec wrapper for a cgo-free
// shared library. It adds _cgo_clib_export.go when compiling the package
// that contains that file, so the linker writes the C names into .dynsym.
//
// cmd/go ignores a file whose name starts with _, and it hashes the
// compiler's -V=full line into the compile cache key. A wrapper that
// forwards that line unchanged shares the cache with a build that never
// saw the export file, and the next link ships a library with an empty
// dynamic symbol table. The build passes the export file as the first
// argument. Its hash is appended to the tool version.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/lewtec/lewkit/x/build/internal/toolexec"
)

// exportCacheGen is appended to the compiler version. Bump it when the
// wrapper's compile behavior changes and the export file bytes do not.
const exportCacheGen = "1"

func main() {
	args := os.Args[1:]
	args, export := peelExport(args)
	if len(args) == 0 {
		os.Exit(2)
	}
	if toolexec.VersionProbe(args) {
		toolexec.PrintVersion("clibtoolexec", args, export, func(path string) (string, error) {
			return toolexec.HashStamp("clibexport=", exportCacheGen, path)
		})
		return
	}
	if toolexec.ToolName(args[0]) == "compile" {
		flat := toolexec.Flatten(args[1:])
		if export == "" {
			export = exportFile(flat)
		}
		if export != "" && sameDir(flat, export) {
			if _, err := os.Stat(export); err != nil {
				fmt.Fprintln(os.Stderr, "clibtoolexec:", err)
				os.Exit(1)
			}
			args = append(args, export)
		}
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			os.Exit(ee.ExitCode())
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// peelExport splits a leading _cgo_clib_export.go path from the tool args.
// The go command splits -toolexec on spaces and then appends the tool.
func peelExport(args []string) (rest []string, export string) {
	if len(args) > 0 && filepath.Base(args[0]) == "_cgo_clib_export.go" {
		return args[1:], args[0]
	}
	return args, ""
}

func sameDir(args []string, export string) bool {
	want, err := filepath.Abs(filepath.Dir(export))
	if err != nil {
		return false
	}
	for _, arg := range args {
		if !strings.HasSuffix(arg, ".go") || strings.HasPrefix(filepath.Base(arg), "_") {
			continue
		}
		dir, err := filepath.Abs(filepath.Dir(arg))
		if err == nil && dir == want {
			return true
		}
	}
	return false
}

func exportFile(args []string) string {
	for _, arg := range args {
		if !strings.HasSuffix(arg, ".go") || strings.HasPrefix(filepath.Base(arg), "_") {
			continue
		}
		candidate := filepath.Join(filepath.Dir(arg), "_cgo_clib_export.go")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}
