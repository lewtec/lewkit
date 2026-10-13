package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	if handled, err := childOrShare(); handled {
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	ensureToolchainEnv()
	os.Exit(m.Run())
}

func TestShare(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("shared object is an ELF file")
	}
	out, err := share()
	if err != nil {
		t.Fatal(err)
	}
	if out != "7\n" {
		t.Fatalf("output %q", out)
	}
}

func TestJIT(t *testing.T) {
	out, err := jit()
	if err != nil {
		t.Fatal(err)
	}
	if out != "jit-ok\n" {
		t.Fatalf("output %q", out)
	}
}

func ensureToolchainEnv() {
	goBin, err := exec.LookPath("go")
	if err != nil {
		goBin = "/home/grokbox/.local/share/mise/installs/go/1.27.0/bin/go"
	}
	path := os.Getenv("PATH")
	dir := filepath.Dir(goBin)
	if !strings.Contains(path, dir) {
		os.Setenv("PATH", dir+string(os.PathListSeparator)+path)
	}
	for _, pair := range [][2]string{
		{"GOCACHE", filepath.Join(os.TempDir(), "lewkit-toolchain-gocache")},
		{"GOMODCACHE", filepath.Join(os.TempDir(), "lewkit-toolchain-gomod")},
		{"GOPATH", filepath.Join(os.TempDir(), "lewkit-toolchain-gopath")},
		{"HOME", filepath.Join(os.TempDir(), "lewkit-toolchain-home")},
	} {
		if err := os.MkdirAll(pair[1], 0o755); err == nil {
			os.Setenv(pair[0], pair[1])
		}
	}
	os.Setenv("GOTOOLCHAIN", "local")
	os.Setenv("CGO_ENABLED", "0")
	os.Setenv("GO111MODULE", "on")
	os.Setenv("GOFLAGS", "")
}
