// Command androidtoolexec is the go -toolexec wrapper for a cgo-free
// Android library. It adds x/entry/_cgo_android_export.go when compiling
// that package, so the linker writes JNI_OnLoad into .dynsym.
// The arm64 cgo-free build passes -tags androidnocgo and
// -ldflags=-checklinkname=0 with this wrapper.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		os.Exit(2)
	}
	if toolName(args[0]) == "compile" && androidArm64Nocgo() {
		flat := flatten(args[1:])
		if pkgOf(flat) == "github.com/lewtec/lewkit/x/entry" {
			export := exportFile(flat)
			if export == "" {
				fmt.Fprintln(os.Stderr, "androidtoolexec: _cgo_android_export.go not found next to x/entry")
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
