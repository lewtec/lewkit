//go:build linux && amd64

package clib

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
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
