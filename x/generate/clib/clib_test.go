package clib

import (
	"bytes"
	"errors"
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	gotool "github.com/lewtec/lewkit/x/toolchain/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	if handled, err := gotool.Child(); handled {
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

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
	require.NoError(t, Run(t.Context(), Config{File: header, ABI: "linux/amd64"}))
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
	err := Run(t.Context(), Config{})
	assert.ErrorIs(t, err, ErrHeader)

	empty := write("empty.h", "int log_line(const char *fmt, ...);\n")
	err = Run(t.Context(), Config{File: empty})
	assert.ErrorIs(t, err, ErrEmpty)

	mainFn := write("main.h", "int main(int argc, char **argv);\n")
	err = Run(t.Context(), Config{File: mainFn})
	assert.ErrorIs(t, err, ErrSymbol)

	agg := write("agg.h", "struct P { int x; };\nint sum(struct P p);\n")
	require.NoError(t, Run(t.Context(), Config{File: agg, ABI: "linux/amd64"}))
	body, err := os.ReadFile(filepath.Join(dir, "agg_clib_amd64.go"))
	require.NoError(t, err)
	assert.Contains(t, string(body), "type P struct")
	assert.Contains(t, string(body), "X int32")
	assert.Contains(t, string(body), "Sum func(p P) int32")

	err = Run(t.Context(), Config{File: agg, ABI: "linux/386"})
	assert.ErrorIs(t, err, ErrType)
	err = Run(t.Context(), Config{File: agg, ABI: "linux/arm"})
	assert.ErrorIs(t, err, ErrType)

	wide := write("wide.h", "#include <stdint.h>\nstruct V { int64_t a; int32_t b; };\nstruct V id(struct V v);\n")
	err = Run(t.Context(), Config{File: wide, ABI: "windows/amd64"})
	assert.ErrorIs(t, err, ErrType)
	require.NoError(t, Run(t.Context(), Config{File: wide, ABI: "linux/amd64"}))

	flt := write("flt.h", "struct P { double x; };\ndouble sum(struct P p);\n")
	err = Run(t.Context(), Config{File: flt, ABI: "linux/amd64"})
	assert.ErrorIs(t, err, ErrType)

	big := write("big.h", "struct P { int a, b, c, d, e; };\nint sum(struct P p);\n")
	err = Run(t.Context(), Config{File: big, ABI: "linux/amd64"})
	assert.ErrorIs(t, err, ErrType)

	ios := write("ios.h", "int add(int a, int b);\n")
	err = Run(t.Context(), Config{File: ios, ABI: "ios/arm64"})
	assert.ErrorIs(t, err, ErrHost)

	bsd := write("bsd.h", "int add(int a, int b);\n")
	err = Run(t.Context(), Config{File: bsd, ABI: "freebsd/amd64"})
	assert.ErrorIs(t, err, ErrHost)
}

func TestGenerateDesktop(t *testing.T) {
	dir := t.TempDir()
	header := filepath.Join(dir, "calc.h")
	require.NoError(t, os.WriteFile(header, []byte(`
#include <stdint.h>
int32_t add(int32_t a, int32_t b, int32_t c, int32_t d, int32_t e);
int32_t mix(int32_t a, double b);
int64_t wide(int32_t a, int64_t b);
`), 0o644))
	require.NoError(t, Run(t.Context(), Config{File: header, ABI: "windows/amd64"}))
	win, err := os.ReadFile(filepath.Join(dir, "calc_clib_windows_amd64.go"))
	require.NoError(t, err)
	text := string(win)
	assert.True(t, strings.HasPrefix(text, "//go:build windows && amd64\n"))
	assert.Contains(t, text, "int32(args[4])")
	assert.Contains(t, text, "math.Float64frombits(xmm[1])")
	asm, err := os.ReadFile(filepath.Join(dir, "calc_clib_windows_amd64.s"))
	require.NoError(t, err)
	assert.Contains(t, string(asm), "//go:build windows && clibexport\n")
	assert.Contains(t, string(asm), "MOVL $0, AX")
	exp, err := os.ReadFile(filepath.Join(dir, "_cgo_clib_export.go"))
	require.NoError(t, err)
	assert.Contains(t, string(exp), "//go:build windows && amd64\n")

	require.NoError(t, Run(t.Context(), Config{File: header, ABI: "win64"}))
	_, err = os.Stat(filepath.Join(dir, "calc_clib_windows_amd64.go"))
	require.NoError(t, err)

	require.NoError(t, Run(t.Context(), Config{File: header, ABI: "darwin/amd64"}))
	dar, err := os.ReadFile(filepath.Join(dir, "calc_clib_darwin_amd64.go"))
	require.NoError(t, err)
	assert.Contains(t, string(dar), "int32(args[0])")
	assert.Contains(t, string(dar), "math.Float64frombits(xmm[0])")
	assert.NotContains(t, string(dar), "xmm[1]")
	asm, err = os.ReadFile(filepath.Join(dir, "calc_clib_darwin_amd64.s"))
	require.NoError(t, err)
	assert.Contains(t, string(asm), "//go:build darwin && clibexport\n")

	require.NoError(t, Run(t.Context(), Config{File: header, ABI: "windows/386"}))
	i386, err := os.ReadFile(filepath.Join(dir, "calc_clib_windows_386.go"))
	require.NoError(t, err)
	assert.Contains(t, string(i386), "clib.U64(args, 8)")

	many := filepath.Join(dir, "many.h")
	require.NoError(t, os.WriteFile(many, []byte(`
int sum(int a, int b, int c, int d, int e, int f, int g, int h, int i);
`), 0o644))
	err = Run(t.Context(), Config{File: many, ABI: "windows/amd64"})
	assert.ErrorIs(t, err, ErrType)

	ld := filepath.Join(dir, "ld.h")
	require.NoError(t, os.WriteFile(ld, []byte("long double widen(long double x);\n"), 0o644))
	err = Run(t.Context(), Config{File: ld, ABI: "darwin/arm64"})
	assert.ErrorIs(t, err, ErrType)
	err = Run(t.Context(), Config{File: ld, ABI: "windows/amd64"})
	assert.ErrorIs(t, err, ErrType)

	require.NoError(t, Run(t.Context(), Config{File: header, ABI: "windows"}))
	_, err = os.Stat(filepath.Join(dir, "calc_clib_windows_arm64.go"))
	require.NoError(t, err)
	exp, err = os.ReadFile(filepath.Join(dir, "_cgo_clib_export.go"))
	require.NoError(t, err)
	assert.Contains(t, string(exp), "//go:build windows && (amd64 || arm64 || 386)\n")

	require.NoError(t, Run(t.Context(), Config{File: header, ABI: "darwin"}))
	_, err = os.Stat(filepath.Join(dir, "calc_clib_darwin_arm64.s"))
	require.NoError(t, err)
	exp, err = os.ReadFile(filepath.Join(dir, "_cgo_clib_export.go"))
	require.NoError(t, err)
	assert.Contains(t, string(exp), "//go:build darwin && (amd64 || arm64)\n")
	asm, err = os.ReadFile(filepath.Join(dir, "calc_clib_darwin_arm64.s"))
	require.NoError(t, err)
	assert.Contains(t, string(asm), "MOVD $0, R9")
}

func TestExportTrie(t *testing.T) {
	buf := exportTrie([]exportSym{{name: "add", addr: 0x1234}, {name: "neg", addr: 0x50}})
	addr, ok := trieLookup(buf, "add")
	assert.True(t, ok)
	assert.Equal(t, uint64(0x1234), addr)
	addr, ok = trieLookup(buf, "neg")
	assert.True(t, ok)
	assert.Equal(t, uint64(0x50), addr)
	_, ok = trieLookup(buf, "missing")
	assert.False(t, ok)
	_, ok = trieLookup(buf, "")
	assert.False(t, ok)
}

func TestMakeDLLStub(t *testing.T) {
	buf := bytes.Repeat([]byte{0}, 0x600)
	buf[0] = 'M'
	buf[1] = 'Z'
	put32(buf, 0x3c, 0x80)
	copy(buf[0x80:], []byte{'P', 'E', 0, 0})
	put16(buf, 0x84, peMachineAmd64)
	put16(buf, 0x86, 1)
	put16(buf, 0x94, 240)
	put16(buf, 0x96, 0x22)
	put16(buf, 0x98, peOptPE32Plus)
	put32(buf, 0xa8, 0x1000)
	sect := 0x98 + 240
	copy(buf[sect:], []byte(".text\x00\x00\x00"))
	put32(buf, sect+8, 0x100)
	put32(buf, sect+12, 0x1000)
	put32(buf, sect+16, 0x200)
	put32(buf, sect+20, 0x400)
	for i := 0; i < 16; i++ {
		buf[0x400+i] = 0x90
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "demo.dll")
	require.NoError(t, os.WriteFile(path, buf, 0o755))
	require.NoError(t, MakeDLL(path))
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, uint16(0x2022), u16(got, 0x96))
	assert.Equal(t, []byte{0xb8, 0x01, 0x00, 0x00, 0x00, 0xc3}, got[0x400:0x406])
	err = requireImageExports(path, "windows", []string{"add"})
	require.Error(t, err)
	assert.False(t, errors.Is(err, ErrHost))
	assert.Contains(t, err.Error(), "add")
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
	err := Run(t.Context(), Config{File: header, ABI: "linux/amd64"})
	assert.ErrorIs(t, err, ErrType)

	require.NoError(t, Run(t.Context(), Config{File: header, ABI: "linux/arm64"}))
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
	require.NoError(t, Run(t.Context(), Config{File: armHeader, ABI: "linux/arm"}))
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

	require.NoError(t, Run(t.Context(), Config{File: header, ABI: "linux/386"}))
	i386, err := os.ReadFile(filepath.Join(dir, "calc_clib_386.go"))
	require.NoError(t, err)
	text = string(i386)
	assert.Contains(t, text, "clib.U32(args, 0)")
	assert.Contains(t, text, "clib.U64(args, 4)")
	assert.NotContains(t, text, "args[7]")

	require.NoError(t, Run(t.Context(), Config{File: header, ABI: "android/arm64"}))
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

func put16(buf []byte, off int, v uint16) {
	buf[off] = byte(v)
	buf[off+1] = byte(v >> 8)
}

func u16(buf []byte, off int) uint16 {
	return uint16(buf[off]) | uint16(buf[off+1])<<8
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

func TestHostTable(t *testing.T) {
	want := map[string]bool{
		"linux/amd64":   true,
		"linux/arm64":   true,
		"linux/386":     false,
		"linux/arm":     false,
		"android/amd64": false,
		"android/arm64": true,
		"android/386":   false,
		"android/arm":   false,
		"windows/amd64": true,
		"windows/arm64": true,
		"windows/386":   true,
		"darwin/amd64":  true,
		"darwin/arm64":  true,
	}
	got := map[string]bool{}
	for _, h := range hosts() {
		key := h.goos + "/" + h.goarch
		_, dup := got[key]
		require.False(t, dup, key)
		got[key] = h.link
		abi, err := h.header()
		require.NoError(t, err, key)
		assert.Equal(t, h.goos, abi.GOOS, key)
		assert.Equal(t, h.goarch, abi.GOARCH, key)
	}
	assert.Equal(t, want, got)

	all, err := targetsFor("")
	require.NoError(t, err)
	require.Len(t, all, 4)
	for _, item := range all {
		assert.Equal(t, "linux", item.goos)
		assert.Empty(t, item.fileOS)
	}
	same, err := targetsFor("all")
	require.NoError(t, err)
	assert.Equal(t, all, same)

	android, err := targetsFor("android/arm64")
	require.NoError(t, err)
	require.Len(t, android, 1)
	assert.Equal(t, "android", android[0].goos)
	assert.Equal(t, "arm64", android[0].goarch)
	assert.Empty(t, android[0].fileOS)
	assert.True(t, android[0].link)

	host, err := targetsFor("host")
	require.NoError(t, err)
	require.Len(t, host, 1)
	assert.Equal(t, runtime.GOOS, host[0].goos)
	assert.Equal(t, runtime.GOARCH, host[0].goarch)

	assert.ErrorIs(t, place(machine(-1), nil), ErrType)
}

func TestBuildGate(t *testing.T) {
	ctx := t.Context()
	out := filepath.Join(t.TempDir(), "out")
	err := BuildFor(ctx, t.TempDir(), out, "ios", "arm64")
	assert.EqualError(t, err, "unsupported target: ios/arm64")

	err = BuildFor(ctx, t.TempDir(), out, "linux", "386")
	assert.EqualError(t, err, "unsupported target: linux/386 has no internal PIE shared library")

	dir := t.TempDir()
	export := filepath.Join(dir, exportFileName)
	body := "//go:build (linux || android) && (amd64 || arm64 || 386 || arm)\n\npackage main\n\n//go:cgo_export_dynamic add add\n"
	require.NoError(t, os.WriteFile(export, []byte(body), 0o644))
	err = BuildFor(ctx, dir, filepath.Join(dir, "out.so"), "android", "amd64")
	assert.EqualError(t, err, "unsupported target: android/amd64 has no internal PIE shared library")
	err = BuildFor(ctx, dir, filepath.Join(dir, "out.dll"), "windows", "amd64")
	assert.EqualError(t, err, "unsupported target: generated export does not include windows/amd64")
	require.NoError(t, exportCovers(export, "linux", "amd64"))
	require.NoError(t, exportCovers(export, "android", "arm64"))
	assert.ErrorIs(t, exportCovers(export, "darwin", "arm64"), ErrHost)
}

func TestGateFiles(t *testing.T) {
	dir := filepath.Join(moduleRoot(t), "x", "ffi", "clib")
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	universal := map[string]bool{
		"doc.go": true, "handle.go": true, "ptr.go": true, "word.go": true, "word_test.go": true,
	}
	var plain, gated []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || (!strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, ".s")) {
			continue
		}
		if universal[name] {
			plain = append(plain, name)
			continue
		}
		gated = append(gated, name)
	}
	sort.Strings(plain)
	assert.Equal(t, []string{"doc.go", "handle.go", "ptr.go", "word.go", "word_test.go"}, plain)

	check := func(goos, goarch string, export bool, want []string) {
		t.Helper()
		ctx := &build.Context{GOOS: goos, GOARCH: goarch, Compiler: "gc"}
		if export {
			ctx.BuildTags = []string{"clibexport"}
		}
		var got []string
		for _, name := range gated {
			ok, err := ctx.MatchFile(dir, name)
			require.NoError(t, err, "%s %s/%s", name, goos, goarch)
			if ok {
				got = append(got, name)
			}
		}
		sort.Strings(got)
		sort.Strings(want)
		assert.Equal(t, want, got, "%s/%s export=%v", goos, goarch, export)
	}
	for _, h := range hosts() {
		check(h.goos, h.goarch, true, gateFiles(h.goos, h.goarch, true))
		check(h.goos, h.goarch, false, gateFiles(h.goos, h.goarch, false))
	}
	for _, pair := range [][2]string{
		{"ios", "arm64"},
		{"freebsd", "amd64"},
		{"darwin", "386"},
		{"windows", "arm"},
		{"linux", "ppc64le"},
	} {
		check(pair[0], pair[1], true, nil)
		check(pair[0], pair[1], false, nil)
	}
}

func gateFiles(goos, goarch string, export bool) []string {
	files := []string{"dispatch.go", "slot.go", "slot_test.go"}
	switch goos {
	case "linux", "android":
		files = append(files, "serve.go")
		if export {
			files = append(files, "call_"+goarch+".s")
		}
	case "windows":
		files = append(files, "import_windows.go", "serve_windows.go")
		if export {
			files = append(files, "call_windows_"+goarch+".s")
		}
	case "darwin":
		files = append(files, "import_darwin.go", "serve_darwin.go")
		if export {
			files = append(files, "call_darwin_"+goarch+".s")
		}
	}
	return files
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
			require.FailNow(t, fmt.Sprint("go.mod not found"))
		}
		dir = parent
	}
}
