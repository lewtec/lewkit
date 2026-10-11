package clib

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAdd(t *testing.T) {
	dir := t.TempDir()
	header := filepath.Join(dir, "add.h")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644))
	require.NoError(t, os.WriteFile(header, []byte(`
#include <stdint.h>
int32_t add(int32_t a, int32_t b);
double scale(double x);
float half(float x);
int32_t deref(int32_t *p);
void touch(void);
int foo(int a);
int Foo(int b);
int log_line(const char *fmt, ...);
`), 0o644))
	require.NoError(t, Run(context.Background(), Config{File: header, ABI: "linux/amd64"}))
	goSrc, err := os.ReadFile(filepath.Join(dir, "add_clib_amd64.go"))
	require.NoError(t, err)
	text := string(goSrc)
	assert.True(t, strings.HasPrefix(text, "//go:build (linux || android) && amd64\n"))
	assert.Contains(t, text, "package main")
	assert.Regexp(t, `Add\s+func\(a int32, b int32\) int32`, text)
	assert.Regexp(t, `Scale\s+func\(x float64\) float64`, text)
	assert.Regexp(t, `Half\s+func\(x float32\) float32`, text)
	assert.Regexp(t, `Deref\s+func\(p \*int32\) int32`, text)
	assert.Regexp(t, `Touch\s+func\(\)`, text)
	assert.Regexp(t, `Foo\s+func\(a int32\) int32`, text)
	assert.Regexp(t, `Foo2\s+func\(b int32\) int32`, text)
	assert.Contains(t, text, "clib.Handle(0, callAdd)")
	assert.Contains(t, text, "math.Float64frombits(xmm[0])")
	assert.Contains(t, text, "math.Float32frombits(uint32(xmm[0]))")
	assert.Contains(t, text, "(*int32)(clib.Ptr(uintptr(args[0])))")
	assert.Contains(t, text, "clib.KindF64")
	assert.Contains(t, text, "clib.KindF32")
	assert.Contains(t, text, "uint64(int64(v))")
	assert.Contains(t, text, "Skipped variadic: log_line")
	assert.NotContains(t, text, "func callLog")

	asm, err := os.ReadFile(filepath.Join(dir, "add_clib_amd64.s"))
	require.NoError(t, err)
	assert.Contains(t, string(asm), "TEXT add(SB),NOSPLIT|NOFRAME,$0")
	assert.Contains(t, string(asm), "MOVL $0, AX")
	assert.Contains(t, string(asm), "TEXT scale(SB),NOSPLIT|NOFRAME,$0")
	assert.Contains(t, string(asm), "github\u00b7com\u2215lewtec\u2215lewkit\u2215x\u2215ffi\u2215clib\u00b7call(SB)")

	exp, err := os.ReadFile(filepath.Join(dir, "_cgo_clib_export.go"))
	require.NoError(t, err)
	assert.Contains(t, string(exp), "package main")
	assert.Contains(t, string(exp), "//go:cgo_export_dynamic add add")
	assert.Contains(t, string(exp), "//go:cgo_export_dynamic scale scale")
	assert.NotContains(t, string(exp), "log_line")
}

func TestGenerateRejects(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) string {
		t.Helper()
		path := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
		return path
	}
	err := Run(context.Background(), Config{})
	assert.ErrorIs(t, err, ErrHeader)

	empty := write("empty.h", "int log_line(const char *fmt, ...);\n")
	err = Run(context.Background(), Config{File: empty})
	assert.ErrorIs(t, err, ErrEmpty)

	mainFn := write("main.h", "int main(int argc, char **argv);\n")
	err = Run(context.Background(), Config{File: mainFn})
	assert.ErrorIs(t, err, ErrSymbol)

	agg := write("agg.h", "struct P { int x; };\nint sum(struct P p);\n")
	err = Run(context.Background(), Config{File: agg, ABI: "linux/amd64"})
	assert.ErrorIs(t, err, ErrType)

	win := write("win.h", "int add(int a, int b);\n")
	err = Run(context.Background(), Config{File: win, ABI: "windows/amd64"})
	assert.ErrorIs(t, err, ErrHost)
}

func TestGenerateArches(t *testing.T) {
	dir := t.TempDir()
	header := filepath.Join(dir, "calc.h")
	require.NoError(t, os.WriteFile(header, []byte(`
#include <stdint.h>
int32_t add(int32_t a, int32_t b, int32_t c, int32_t d, int32_t e, int32_t f, int32_t g, int32_t h);
int64_t wide(int32_t a, int64_t b);
float mix(float a, double b, float c);
int32_t pack(int8_t a, double b);
`), 0o644))
	err := Run(context.Background(), Config{File: header, ABI: "linux/amd64"})
	assert.ErrorIs(t, err, ErrType)

	require.NoError(t, Run(context.Background(), Config{File: header, ABI: "linux/arm64"}))
	arm64, err := os.ReadFile(filepath.Join(dir, "calc_clib_arm64.go"))
	require.NoError(t, err)
	assert.Contains(t, string(arm64), "int32(args[7])")
	asm, err := os.ReadFile(filepath.Join(dir, "calc_clib_arm64.s"))
	require.NoError(t, err)
	assert.Contains(t, string(asm), "MOVD $0, R9")

	// arm keeps four integer registers and does not spill, so the eight-word
	// add above is rejected there. The rest of the header still exercises
	// the even-odd int64 pair and the VFP backfill hole.
	armHeader := filepath.Join(dir, "arm.h")
	require.NoError(t, os.WriteFile(armHeader, []byte(`
#include <stdint.h>
int32_t add(int32_t a, int32_t b, int32_t c, int32_t d);
int64_t wide(int32_t a, int64_t b);
float mix(float a, double b, float c);
int32_t pack(int8_t a, double b);
`), 0o644))
	require.NoError(t, Run(context.Background(), Config{File: armHeader, ABI: "linux/arm"}))
	armSrc, err := os.ReadFile(filepath.Join(dir, "arm_clib_arm.go"))
	require.NoError(t, err)
	text := string(armSrc)
	assert.Contains(t, text, "int32(args[0])")
	assert.Contains(t, text, "uint64(uint32(args[2]))|uint64(uint32(args[3]))<<32")
	assert.Contains(t, text, "math.Float32frombits(uint32(xmm[0]))")
	assert.Contains(t, text, "math.Float64frombits(xmm[1])")
	assert.Contains(t, text, "math.Float32frombits(uint32(xmm[0]>>32))")
	asm, err = os.ReadFile(filepath.Join(dir, "arm_clib_arm.s"))
	require.NoError(t, err)
	assert.Contains(t, string(asm), "MOVW.W R4, -8(R13)")

	require.NoError(t, Run(context.Background(), Config{File: header, ABI: "linux/386"}))
	i386, err := os.ReadFile(filepath.Join(dir, "calc_clib_386.go"))
	require.NoError(t, err)
	text = string(i386)
	assert.Contains(t, text, "clib.U32(args, 0)")
	assert.Contains(t, text, "clib.U64(args, 4)")
	assert.NotContains(t, text, "args[7]")

	require.NoError(t, Run(context.Background(), Config{File: header, ABI: "android/arm64"}))
	_, err = os.Stat(filepath.Join(dir, "calc_clib_arm64.go"))
	require.NoError(t, err)
}

func TestMakeSharedClearsPIE32(t *testing.T) {
	buf := bytes.Repeat([]byte{0}, 160)
	copy(buf[:4], []byte{0x7f, 'E', 'L', 'F'})
	buf[4] = 1
	buf[5] = 1
	buf[6] = 1
	// e_phoff
	buf[28] = 52
	// e_phentsize, e_phnum
	buf[42] = 32
	buf[44] = 2
	// PT_INTERP at 52. p_offset 120, p_filesz 4.
	buf[52] = elfPTInterp
	put32(buf, 52+4, 120)
	put32(buf, 52+16, 4)
	// PT_DYNAMIC at 84. p_offset 128, p_filesz 16.
	buf[84] = elfPTDynamic
	put32(buf, 84+4, 128)
	put32(buf, 84+16, 16)
	// DT_FLAGS_1 = DF_1_PIE, then DT_NULL.
	put32(buf, 128, uint32(elfDTFlags1))
	put32(buf, 132, elfDF1PIE)
	dir := t.TempDir()
	path := filepath.Join(dir, "lib.so")
	require.NoError(t, os.WriteFile(path, buf, 0o755))
	require.NoError(t, MakeShared(path))
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, byte(elfPTNull), got[52])
	assert.Equal(t, uint32(0), u32(got, 132))
}

func TestMakeSharedClearsPIE(t *testing.T) {
	buf := bytes.Repeat([]byte{0}, 240)
	copy(buf[:4], []byte{0x7f, 'E', 'L', 'F'})
	buf[4] = 2
	buf[5] = 1
	buf[6] = 1
	// e_phoff
	buf[32] = 64
	// e_phentsize
	buf[54] = 56
	// e_phnum
	buf[56] = 2
	// PT_INTERP at 64. p_offset 180, p_filesz 4.
	buf[64] = elfPTInterp
	put64(buf, 64+8, 180)
	put64(buf, 64+32, 4)
	// PT_DYNAMIC at 120. p_offset 200, p_filesz 32.
	buf[120] = elfPTDynamic
	put64(buf, 120+8, 200)
	put64(buf, 120+32, 32)
	// DT_FLAGS_1 = DF_1_PIE, then DT_NULL.
	put64(buf, 200, elfDTFlags1)
	put64(buf, 208, elfDF1PIE)
	dir := t.TempDir()
	path := filepath.Join(dir, "lib.so")
	require.NoError(t, os.WriteFile(path, buf, 0o755))
	require.NoError(t, MakeShared(path))
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, byte(elfPTNull), got[64])
	assert.Equal(t, uint64(0), u64(got, 208))
}

func put32(buf []byte, off int, v uint32) {
	buf[off] = byte(v)
	buf[off+1] = byte(v >> 8)
	buf[off+2] = byte(v >> 16)
	buf[off+3] = byte(v >> 24)
}

func u32(buf []byte, off int) uint32 {
	return uint32(buf[off]) | uint32(buf[off+1])<<8 | uint32(buf[off+2])<<16 | uint32(buf[off+3])<<24
}

func put64(buf []byte, off int, v uint64) {
	buf[off] = byte(v)
	buf[off+1] = byte(v >> 8)
	buf[off+2] = byte(v >> 16)
	buf[off+3] = byte(v >> 24)
	buf[off+4] = byte(v >> 32)
	buf[off+5] = byte(v >> 40)
	buf[off+6] = byte(v >> 48)
	buf[off+7] = byte(v >> 56)
}

func u64(buf []byte, off int) uint64 {
	var v uint64
	for i := 0; i < 8; i++ {
		v |= uint64(buf[off+i]) << (8 * i)
	}
	return v
}
