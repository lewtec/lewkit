// Package toolexec is the shared -toolexec wrapper used by the Android and C-library builds.
package toolexec

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

// VersionProbe reports the compiler's -V=full request.
func VersionProbe(args []string) bool {
	return len(args) == 2 && args[1] == "-V=full"
}

// ToolName is the tool's file name without a trailing .exe.
func ToolName(path string) string {
	return strings.TrimSuffix(filepath.Base(path), ".exe")
}

// PrintVersion runs args as a version probe and prints the tool line.
// When export is set, stamp is appended. A failure writes to stderr and exits.
func PrintVersion(label string, args []string, export string, stamp func(string) (string, error)) {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = os.Environ()
	out, err := cmd.Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			os.Stderr.Write(exitError.Stderr)
			os.Exit(exitError.ExitCode())
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	text := strings.TrimRight(string(out), "\n")
	if export != "" {
		token, err := stamp(export)
		if err != nil {
			fmt.Fprintln(os.Stderr, label+":", err)
			os.Exit(1)
		}
		text += " " + token
	}
	fmt.Println(text)
}

// HashStamp is prefix + generation + "-" + the first 8 bytes of the file's SHA-256, hex encoded.
func HashStamp(prefix, generation, path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return prefix + generation + "-" + hex.EncodeToString(hash.Sum(nil)[:8]), nil
}

// Flatten expands @response files. A missing file is left as the original argument.
// The compiler still receives the caller's original arguments.
func Flatten(args []string) []string {
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
		out = append(out, Flatten(responseArgs(string(body)))...)
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
