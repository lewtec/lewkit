package macho

import (
	"bytes"
	stdmacho "debug/macho"
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
	code := []byte{0x90, 0xc3}
	file := New().CPU(Amd64)
	file.Segment("__TEXT").Section("__text").Bytes(code).Symbol("_main")

	first := takeMade(t, file)
	require.Equal(t, Amd64, first.Header.CPU)
	require.Equal(t, TypeObj, first.Kind)
	require.Equal(t, Little, first.Endian)
	text := findSection(t, first, "__text")
	require.Equal(t, code, text.Bytes)
	require.Equal(t, "__TEXT", text.Seg)
	require.Len(t, text.Symbols, 1)
	require.Equal(t, "_main", text.Symbols[0].Name)
	require.NotEmpty(t, first.Commands)
	require.NotEmpty(t, first.Segments)

	var buf bytes.Buffer
	require.NoError(t, first.Make(&buf))
	made := append([]byte(nil), buf.Bytes()...)
	parsed, err := stdmacho.NewFile(bytes.NewReader(made))
	require.NoError(t, err)
	require.Equal(t, stdmacho.CpuAmd64, parsed.Cpu)
	require.NotNil(t, parsed.Symtab)
	var found bool
	for _, sym := range parsed.Symtab.Syms {
		if sym.Name == "_main" {
			found = true
		}
	}
	require.True(t, found, "_main")

	second, err := Take(bytes.NewReader(made))
	require.NoError(t, err)
	require.Equal(t, first, second)
}

func TestComposedDylib(t *testing.T) {
	t.Parallel()
	file := New().CPU(Arm64).Dylib()
	file.Segment("__TEXT").Section("__text").Bytes([]byte{0xc3}).Symbol("_entry")
	file.Entry(0x1000)
	first := takeMade(t, file)
	require.Equal(t, Arm64, first.Header.CPU)
	require.Equal(t, TypeDylib, first.Kind)
	require.Equal(t, uint64(0x1000), first.EntryOff)
	require.NotEmpty(t, first.Dylibs)
	require.Equal(t, lcIDDylib, first.Dylibs[0].Cmd)
	require.Equal(t, "lib.dylib", first.Dylibs[0].Name)
	require.Equal(t, "_entry", findSection(t, first, "__text").Symbols[0].Name)
	second := takeMade(t, first)
	require.Equal(t, first, second)
}

func TestComposed32(t *testing.T) {
	t.Parallel()
	file := New().CPU(I386).Type(TypeObj)
	file.Segment("__TEXT").Section("__text").Bytes([]byte{0xc3}).Symbol("_main")
	first := takeMade(t, file)
	require.Equal(t, I386, first.Header.CPU)
	require.Equal(t, magic32, first.Magic)
	require.Equal(t, "_main", findSection(t, first, "__text").Symbols[0].Name)
	second := takeMade(t, first)
	require.Equal(t, first, second)
}

func TestRealMachO(t *testing.T) {
	t.Parallel()
	goBin, err := exec.LookPath("go")
	if err != nil {
		test.Skip(t, "go is not installed")
	}
	for _, arch := range []string{"amd64", "arm64"} {
		t.Run(arch, func(t *testing.T) {
			t.Parallel()
			roundTripPath(t, buildGo(t, goBin, "darwin", arch))
		})
	}
}

func TestRunMath(t *testing.T) {
	if runtime.GOOS != "darwin" {
		test.Skipf(t, "running a Mach-O needs darwin, have %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	goBin, err := exec.LookPath("go")
	if err != nil {
		test.Skip(t, "go is not installed")
	}
	path := buildMathGo(t, goBin)
	require.Equal(t, mathOutput, runBin(t, path))
	require.Equal(t, mathOutput, runBin(t, rewriteBin(t, path)))
}

func TestRejects(t *testing.T) {
	t.Parallel()
	_, err := Take(nil)
	require.ErrorIs(t, err, ErrNilReader)
	require.ErrorIs(t, err, errs.ErrInvalid)

	require.ErrorIs(t, New().Make(nil), ErrNilWriter)
	require.ErrorIs(t, (*File)(nil).Make(io.Discard), ErrNilFile)

	err = New().Segment("").Make(io.Discard)
	require.ErrorIs(t, err, ErrName)
	err = New().Section("__text").Make(io.Discard)
	require.ErrorIs(t, err, ErrSection)
	err = New().Segment("__TEXT").Section("").Make(io.Discard)
	require.ErrorIs(t, err, ErrName)

	_, err = Take(bytes.NewReader([]byte("nope")))
	require.ErrorIs(t, err, ErrFormat)
	require.ErrorIs(t, err, errs.ErrSyntax)
	_, err = Take(bytes.NewReader(nil))
	require.ErrorIs(t, err, ErrFormat)
}

func buildGo(t *testing.T, goBin, goos, goarch string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n\ngo 1.27\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644))
	out := filepath.Join(dir, "main.bin")
	cmd := exec.Command(goBin, "build", "-o", out, ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GOOS="+goos,
		"GOARCH="+goarch,
		"CGO_ENABLED=0",
		"GO111MODULE=on",
		"GOPROXY=off",
	)
	log, err := cmd.CombinedOutput()
	require.NoError(t, err, string(log))
	return out
}

func roundTripPath(t *testing.T, path string) {
	t.Helper()
	data := mustRead(t, path)
	first, err := Take(bytes.NewReader(data))
	require.NoError(t, err)
	require.NotEmpty(t, first.Segments)
	require.NotEmpty(t, first.Commands)
	require.NotEmpty(t, first.Dylibs)
	var text []byte
	var symbols int
	for _, segment := range first.Segments {
		for _, section := range segment.Sections {
			symbols += len(section.Symbols)
			if section.Name == "__text" {
				text = section.Bytes
			}
		}
	}
	symbols += len(first.Symbols)
	require.NotEmpty(t, text)
	require.NotZero(t, symbols)

	var buf bytes.Buffer
	require.NoError(t, first.Make(&buf))
	made := append([]byte(nil), buf.Bytes()...)
	parsed, err := stdmacho.NewFile(bytes.NewReader(made))
	require.NoError(t, err)
	require.Equal(t, stdmacho.Cpu(first.Header.CPU), parsed.Cpu)
	require.NotEmpty(t, parsed.Sections)

	second, err := Take(bytes.NewReader(made))
	require.NoError(t, err)
	require.Equal(t, first, second)
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
	for _, segment := range file.Segments {
		for _, section := range segment.Sections {
			if section.Name == name {
				return section
			}
		}
	}
	t.Fatalf("missing section %s", name)
	return nil
}

const mathProgram = `package main

import "fmt"

func main() {
	fmt.Printf("%d %d %d\n", 6*7, 100-58, 21+21)
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
