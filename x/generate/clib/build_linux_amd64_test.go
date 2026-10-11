//go:build linux && amd64

package clib

import (
	"context"
	"debug/elf"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDlopen(t *testing.T) {
	repo := moduleRoot(t)
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/demo\n\ngo 1.27.0\n\nrequire github.com/lewtec/lewkit v0.0.0\n\nreplace github.com/lewtec/lewkit => "+repo+"\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "add.h"), []byte(`
#include <stdint.h>
int32_t add(int32_t a, int32_t b);
int32_t neg(int32_t v);
double scale(double x);
float half(float x);
int32_t deref(int32_t *p);
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte(`
package main

func init() {
	API.Add = func(a, b int32) int32 { return a + b }
	API.Neg = func(v int32) int32 { return -v }
	API.Scale = func(x float64) float64 { return x * 2 }
	API.Half = func(x float32) float32 { return x / 2 }
	API.Deref = func(p *int32) int32 {
		if p == nil {
			return 0
		}
		return *p
	}
}

func main() {}
`), 0o644))
	require.NoError(t, Run(context.Background(), Config{File: filepath.Join(dir, "add.h")}))
	lib := filepath.Join(dir, "libdemo.so")
	require.NoError(t, Build(context.Background(), dir, lib))

	host := filepath.Join(dir, "host.c")
	require.NoError(t, os.WriteFile(host, []byte(`
#include <dlfcn.h>
#include <stdint.h>
#include <stdio.h>
int main(void) {
	void *h = dlopen("`+lib+`", RTLD_NOW);
	if (!h) { fprintf(stderr, "dlopen: %s\n", dlerror()); return 1; }
	int32_t (*add)(int32_t, int32_t) = dlsym(h, "add");
	int32_t (*neg)(int32_t) = dlsym(h, "neg");
	double (*scale)(double) = dlsym(h, "scale");
	float (*half)(float) = dlsym(h, "half");
	int32_t (*deref)(int32_t *) = dlsym(h, "deref");
	if (!add || !neg || !scale || !half || !deref) {
		fprintf(stderr, "dlsym: %s\n", dlerror());
		return 1;
	}
	if (add(20, 22) != 42) { fprintf(stderr, "add\n"); return 1; }
	if (neg(-3) != 3) { fprintf(stderr, "neg\n"); return 1; }
	double y = scale(2.5);
	if (y < 4.9 || y > 5.1) { fprintf(stderr, "scale %f\n", y); return 1; }
	float z = half(3.0f);
	if (z < 1.4f || z > 1.6f) { fprintf(stderr, "half %f\n", z); return 1; }
	int32_t n = 7;
	if (deref(&n) != 7) { fprintf(stderr, "deref\n"); return 1; }
	if (add(1, 2) != 3) { fprintf(stderr, "add2\n"); return 1; }
	return 0;
}
`), 0o644))
	bin := filepath.Join(dir, "host")
	cmd := exec.Command("gcc", "-o", bin, host, "-ldl")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	cmd = exec.Command(bin)
	out, err = cmd.CombinedOutput()
	require.NoError(t, err, string(out))
}

func TestCrossLink(t *testing.T) {
	repo := moduleRoot(t)
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/demo\n\ngo 1.27.0\n\nrequire github.com/lewtec/lewkit v0.0.0\n\nreplace github.com/lewtec/lewkit => "+repo+"\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "add.h"), []byte("#include <stdint.h>\nint32_t add(int32_t a, int32_t b);\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644))
	require.NoError(t, Run(context.Background(), Config{File: filepath.Join(dir, "add.h")}))

	targets := []struct {
		goos, goarch string
		class        elf.Class
		machine      elf.Machine
	}{
		{"linux", "amd64", elf.ELFCLASS64, elf.EM_X86_64},
		{"linux", "arm64", elf.ELFCLASS64, elf.EM_AARCH64},
		{"android", "arm64", elf.ELFCLASS64, elf.EM_AARCH64},
	}
	for _, target := range targets {
		t.Run(target.goos+"/"+target.goarch, func(t *testing.T) {
			lib := filepath.Join(dir, "lib"+target.goos+"_"+target.goarch+".so")
			require.NoError(t, BuildFor(context.Background(), dir, lib, target.goos, target.goarch))
			f, err := elf.Open(lib)
			require.NoError(t, err)
			defer f.Close()
			require.Equal(t, target.class, f.Class)
			require.Equal(t, target.machine, f.Machine)
			for _, prog := range f.Progs {
				require.NotEqual(t, elf.PT_INTERP, prog.Type)
			}
			flags, err := f.DynValue(elf.DT_FLAGS_1)
			require.NoError(t, err)
			require.NotEmpty(t, flags)
			require.Zero(t, flags[0]&uint64(elf.DF_1_PIE))
			syms, err := f.DynamicSymbols()
			require.NoError(t, err)
			found := false
			for _, sym := range syms {
				if sym.Name == "add" {
					found = true
					break
				}
			}
			require.True(t, found, "dynamic symbol add")
		})
	}
}

// TestEntryLinks proves the 386 and arm entry against the real runtime.
// Go's internal linker has no PIE for those architectures, so BuildFor
// cannot write the shared library. The test binary still resolves
// runtime.clone, runtime.rt0_go, and runtime.islibrary.
func TestEntryLinks(t *testing.T) {
	repo := moduleRoot(t)
	for _, arch := range []string{"386", "arm"} {
		t.Run("linux/"+arch, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "clib.test")
			cmd := exec.Command("go", "test", "-c",
				"-tags=clibexport",
				"-ldflags=-linkmode=internal -checklinkname=0",
				"-o", out,
				"./x/ffi/clib")
			cmd.Dir = repo
			env := make([]string, 0, len(os.Environ())+4)
			for _, e := range os.Environ() {
				if strings.HasPrefix(e, "GOOS=") || strings.HasPrefix(e, "GOARCH=") || strings.HasPrefix(e, "GOARM=") || strings.HasPrefix(e, "CGO_ENABLED=") {
					continue
				}
				env = append(env, e)
			}
			env = append(env, "CGO_ENABLED=0", "GOOS=linux", "GOARCH="+arch)
			if arch == "arm" {
				env = append(env, "GOARM=7")
			}
			cmd.Env = env
			outb, err := cmd.CombinedOutput()
			require.NoError(t, err, string(outb))
		})
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}
