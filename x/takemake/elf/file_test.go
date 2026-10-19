package elf

import (
	"bytes"
	"debug/elf"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/lewtec/lewkit/x/errs"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestComposedRoundTrip(t *testing.T) {
	t.Parallel()
	code := []byte{0x90, 0xc3, 0x90, 0xc3}
	file := New().Class64().Little().Machine(X86_64).Type(Rel).Entry(0x1000)
	file.Section(".text").Alloc().Exec().Addr(0x1000).Align(16).Bytes(code).
		Symbol("main").Func().Global().SymSize(4).Value(0x1000)
	file.Section(".data").Alloc().Write().Addr(0x2000).Bytes([]byte("hi")).
		Symbol("msg").Object().Global().SymSize(2)
	file.Section(".text").Reloc(1, 1, "msg", -4)
	file.Segment(ProgLoad).Perm(PermRead | PermExec).At(0x1000).Cover(".text")

	first := takeMade(t, file)
	require.Equal(t, Class64, first.Class)
	require.Equal(t, Little, first.Endian)
	require.Equal(t, X86_64, first.Header.Machine)
	require.Equal(t, Rel, first.Kind)
	require.Equal(t, uint64(0x1000), first.Header.Entry)
	text := findSection(t, first, ".text")
	require.Equal(t, code, text.Bytes)
	require.Equal(t, FlagAlloc|FlagExec, text.Flags)
	require.Equal(t, uint64(0x1000), text.Addr)
	require.Len(t, text.Symbols, 1)
	require.Equal(t, "main", text.Symbols[0].Name)
	require.Equal(t, SymFunc, text.Symbols[0].Info&0xf)
	require.Equal(t, BindGlobal, text.Symbols[0].Info>>4)
	require.Len(t, text.Relocs, 1)
	require.Equal(t, uint64(1), text.Relocs[0].Offset)
	require.Equal(t, int64(-4), text.Relocs[0].Addend)
	data := findSection(t, first, ".data")
	require.Equal(t, "msg", data.Symbols[0].Name)
	require.NotEmpty(t, first.Segments)
	require.Contains(t, first.Segments[0].Covers, indexOf(first, ".text"))

	second := takeMade(t, first)
	require.Equal(t, first, second)
}

func TestRealELF(t *testing.T) {
	t.Parallel()
	paths := []string{"/bin/ls", "/bin/sh"}
	if goBin, err := exec.LookPath("go"); err == nil {
		paths = append(paths, goBin)
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			roundTripPath(t, path)
		})
	}
}

func TestRunMath(t *testing.T) {
	if runtime.GOOS != "linux" {
		test.Skipf(t, "running an ELF needs linux, have %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	goBin, err := exec.LookPath("go")
	if err != nil {
		test.Skip(t, "go is not installed")
	}
	t.Run("go", func(t *testing.T) {
		path := buildMathGo(t, goBin)
		require.Equal(t, mathOutput, runBin(t, path))
		require.Equal(t, mathOutput, runBin(t, rewriteBin(t, path)))
	})
	t.Run("c", func(t *testing.T) {
		if _, err := exec.LookPath("gcc"); err != nil {
			test.Skip(t, "gcc is not installed")
		}
		path := buildMathC(t)
		require.Equal(t, mathOutput, runBin(t, path))
		require.Equal(t, mathOutput, runBin(t, rewriteBin(t, path)))
	})
}

func TestRealObject(t *testing.T) {
	t.Parallel()
	gcc, err := exec.LookPath("gcc")
	if err != nil {
		test.Skip(t, "gcc is not installed")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "unit.c")
	object := filepath.Join(dir, "unit.o")
	body := "int meaning(void) { return 42; }\nint (*hook)(void) = meaning;\n"
	require.NoError(t, os.WriteFile(src, []byte(body), 0o644))
	cmd := exec.Command(gcc, "-c", "-o", object, src)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	roundTripPath(t, object)

	taken, err := Take(bytes.NewReader(mustRead(t, object)))
	require.NoError(t, err)
	var found bool
	for _, section := range taken.Sections {
		for _, symbol := range section.Symbols {
			if symbol.Name == "meaning" {
				found = true
				require.Equal(t, SymFunc, symbol.Info&0xf)
			}
		}
	}
	require.True(t, found, "meaning")
	parsed, err := elf.Open(object)
	require.NoError(t, err)
	defer parsed.Close()
	require.Equal(t, elf.ET_REL, parsed.Type)
}

func TestRejects(t *testing.T) {
	t.Parallel()
	_, err := Take(nil)
	require.ErrorIs(t, err, ErrNilReader)
	require.ErrorIs(t, err, errs.ErrInvalid)

	require.ErrorIs(t, New().Make(nil), ErrNilWriter)
	require.ErrorIs(t, (*File)(nil).Make(io.Discard), ErrNilFile)

	err = New().Section("").Make(io.Discard)
	require.ErrorIs(t, err, ErrName)

	_, err = Take(bytes.NewReader([]byte("nope")))
	require.ErrorIs(t, err, ErrFormat)
	require.ErrorIs(t, err, errs.ErrSyntax)

	_, err = Take(bytes.NewReader(nil))
	require.ErrorIs(t, err, ErrFormat)
}

func roundTripPath(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		test.Skipf(t, "missing %s", path)
	}
	if info.IsDir() {
		test.Skipf(t, "%s is a directory", path)
	}
	data := mustRead(t, path)
	if len(data) < 4 || string(data[:4]) != "\x7fELF" {
		test.Skipf(t, "%s is not an ELF", path)
	}
	first, err := Take(bytes.NewReader(data))
	require.NoError(t, err)
	require.NotEmpty(t, first.Sections)
	var text []byte
	for _, section := range first.Sections {
		if section.Name == ".text" {
			text = section.Bytes
		}
	}
	require.NotEmpty(t, text)
	var buf bytes.Buffer
	require.NoError(t, first.Make(&buf))
	made := append([]byte(nil), buf.Bytes()...)
	second, err := Take(bytes.NewReader(made))
	require.NoError(t, err)
	require.Equal(t, first, second)

	parsed, err := elf.NewFile(bytes.NewReader(made))
	require.NoError(t, err)
	defer parsed.Close()
	require.Equal(t, elf.Machine(first.Header.Machine), parsed.Machine)
	require.Equal(t, elf.Class(first.Class), parsed.Class)
}

func takeMade(t *testing.T, file *File) *File {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, file.Make(&buf))
	got, err := Take(&buf)
	require.NoError(t, err)
	return got
}

func findSection(t *testing.T, file *File, name string) *Section {
	t.Helper()
	for _, section := range file.Sections {
		if section.Name == name {
			return section
		}
	}
	require.Failf(t, "missing section", "%s", name)
	return nil
}

func indexOf(file *File, name string) int {
	for i, section := range file.Sections {
		if section.Name == name {
			return i
		}
	}
	return -1
}

const mathProgram = `package main

import "fmt"

func main() {
	fmt.Printf("%d %d %d\n", 6*7, 100-58, 21+21)
}
`

const mathC = `#include <stdio.h>
int main(void) {
	printf("%d %d %d\n", 6 * 7, 100 - 58, 21 + 21);
	return 0;
}
`

const mathOutput = "42 42 42\n"

func buildMathGo(t *testing.T, goBin string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n\ngo 1.27\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte(mathProgram), 0o644))
	out := filepath.Join(dir, "mathbin")
	cmd := exec.Command(goBin, "build", "-o", out, ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GOOS="+runtime.GOOS,
		"GOARCH="+runtime.GOARCH,
		"CGO_ENABLED=0",
		"GO111MODULE=on",
		"GOPROXY=off",
	)
	log, err := cmd.CombinedOutput()
	require.NoError(t, err, string(log))
	return out
}

func buildMathC(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	src := filepath.Join(dir, "math.c")
	out := filepath.Join(dir, "mathc")
	require.NoError(t, os.WriteFile(src, []byte(mathC), 0o644))
	cmd := exec.Command("gcc", "-o", out, src)
	log, err := cmd.CombinedOutput()
	require.NoError(t, err, string(log))
	return out
}

func rewriteBin(t *testing.T, path string) string {
	t.Helper()
	file, err := Take(bytes.NewReader(mustRead(t, path)))
	require.NoError(t, err)
	var buf bytes.Buffer
	require.NoError(t, file.Make(&buf))
	out := filepath.Join(t.TempDir(), "rewritten")
	require.NoError(t, os.WriteFile(out, buf.Bytes(), 0o755))
	return out
}

func runBin(t *testing.T, path string) string {
	t.Helper()
	out, err := exec.Command(path).CombinedOutput()
	require.NoError(t, err, string(out))
	return string(out)
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return data
}
