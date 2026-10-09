//go:build linux

package native

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestHostLibDirsReadsNixProfiles(t *testing.T) {
	t.Setenv("NIX_PROFILES", "/tmp/nix-prof-a"+string(filepath.ListSeparator)+"/tmp/nix-prof-b")
	t.Setenv("USER", "lucasew")
	t.Setenv("CONDA_PREFIX", "/tmp/conda-webkit")
	dirs := hostLibDirs()
	want := []string{
		"/tmp/nix-prof-a/lib",
		"/tmp/nix-prof-b/lib",
		"/etc/profiles/per-user/lucasew/lib",
		"/tmp/conda-webkit/lib",
	}
	if dir := multiarchDir(); dir != "" {
		want = append(want, dir)
	}
	for _, want := range want {
		if !containsDir(dirs, want) {
			t.Fatalf("missing %s in %q", want, dirs)
		}
	}
}

func multiarchDir() string {
	switch runtime.GOARCH {
	case "amd64":
		return "/usr/lib/x86_64-linux-gnu"
	case "arm64":
		return "/usr/lib/aarch64-linux-gnu"
	case "386":
		return "/usr/lib/i386-linux-gnu"
	default:
		return ""
	}
}

func TestToolPathFindsExecutableOnPath(t *testing.T) {
	dir := t.TempDir()
	blocked := filepath.Join(dir, "blocked")
	if err := os.Mkdir(blocked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(blocked, "zenity"), []byte("nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "zenity"), 0o755); err != nil {
		t.Fatal(err)
	}
	binDir := filepath.Join(dir, "bin")
	if err := os.Mkdir(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(binDir, "zenity")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", strings.Join([]string{blocked, dir, binDir, "."}, string(filepath.ListSeparator)))
	if got := toolPath(t.Context(), "zenity"); got != bin {
		t.Fatalf("toolPath = %q, want %q", got, bin)
	}
	if got := toolPath(t.Context(), "gtk4-launch"); got != "" {
		t.Fatalf("missing toolPath = %q", got)
	}
}

func TestHostLibDirsReadsPathSiblingLib(t *testing.T) {
	t.Setenv("PATH", strings.Join([]string{
		"/tmp/mise-webkit/.mise-bins",
		"/tmp/conda-prefix/bin",
		"relative/bin",
	}, string(filepath.ListSeparator)))
	t.Setenv("CONDA_PREFIX", "")
	dirs := hostLibDirs()
	for _, want := range []string{"/tmp/mise-webkit/lib", "/tmp/conda-prefix/lib"} {
		if !containsDir(dirs, want) {
			t.Fatalf("missing %s in %q", want, dirs)
		}
	}
	if containsDir(dirs, "relative/lib") {
		t.Fatalf("relative PATH lib leaked into %q", dirs)
	}
}

func TestPrepareRejectsNilContext(t *testing.T) {
	if err := Prepare(nil); err != errNilContext {
		t.Fatalf("Prepare(nil) = %v", err)
	}
}

func containsDir(dirs []string, want string) bool {
	for _, dir := range dirs {
		if dir == want {
			return true
		}
	}
	return false
}
