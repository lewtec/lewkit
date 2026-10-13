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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	entryPkg       = "github.com/lewtec/lewkit/x/entry/android"
	exportCacheGen = "1"
)

func main() {
	args := os.Args[1:]
	args, export := peelExport(args)
	if len(args) == 0 {
		os.Exit(2)
	}
	if versionProbe(args) {
		printToolVersion(args, export)
		return
	}
	if toolName(args[0]) == "compile" && androidArm64Nocgo() {
		flat := flatten(args[1:])
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

// peelExport splits a leading _cgo_android_export.go path from the tool args.
// The go command splits -toolexec on spaces and then appends the tool.
func peelExport(args []string) (rest []string, export string) {
	if len(args) > 0 && filepath.Base(args[0]) == "_cgo_android_export.go" {
		return args[1:], args[0]
	}
	return args, ""
}

func versionProbe(args []string) bool {
	return len(args) == 2 && args[1] == "-V=full"
}

func printToolVersion(args []string, export string) {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = os.Environ()
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			os.Stderr.Write(ee.Stderr)
			os.Exit(ee.ExitCode())
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	text := strings.TrimRight(string(out), "\n")
	if export != "" {
		stamp, err := exportStamp(export)
		if err != nil {
			fmt.Fprintln(os.Stderr, "androidtoolexec:", err)
			os.Exit(1)
		}
		text += " " + stamp
	}
	fmt.Println(text)
}

// exportStamp is the cache token for this export file. Bump exportCacheGen
// when the wrapper's compile behavior changes and the file bytes do not.
func exportStamp(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return "lewkitexport=" + exportCacheGen + "-" + hex.EncodeToString(h.Sum(nil)[:8]), nil
}

func androidArm64Nocgo() bool {
	return os.Getenv("GOOS") == "android" &&
		os.Getenv("GOARCH") == "arm64" &&
		os.Getenv("CGO_ENABLED") == "0"
}

func toolName(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, ".exe")
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

// flatten expands @response files enough to find -p and the package directory.
// The compiler still receives the original arguments.
func flatten(args []string) []string {
	var out []string
	for _, arg := range args {
		if !strings.HasPrefix(arg, "@") || len(arg) == 1 {
			out = append(out, arg)
			continue
		}
		body, err := os.ReadFile(arg[1:])
		if err != nil {
			out = append(out, arg)
			continue
		}
		out = append(out, flatten(responseArgs(string(body)))...)
	}
	return out
}

func responseArgs(body string) []string {
	var out []string
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if len(line) >= 2 && line[0] == '"' && line[len(line)-1] == '"' {
			line = line[1 : len(line)-1]
		}
		out = append(out, line)
	}
	return out
}
