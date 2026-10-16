package pe

import (
	"bytes"
	"debug/pe"
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
	code := []byte{0xc3}
	file := New().Machine(Amd64).Subsystem(SubsystemCUI).ImageBase(0x140000000).Entry(0x1000)
	file.Section(".text").Exec().Read().Addr(0x1000).Bytes(code)
	file.Import("kernel32.dll").Symbol("GetProcAddress").ByOrdinal(15)
	file.Export("Hello").RVA(0x1000)

	first := takeMade(t, file)
	require.Equal(t, Amd64, first.Header.Machine)
	require.Equal(t, SubsystemCUI, first.Header.Subsystem)
	require.Equal(t, uint64(0x140000000), first.Header.ImageBase)
	require.Equal(t, uint32(0x1000), first.Header.Entry)
	text := findSection(t, first, ".text")
	require.Equal(t, code, text.Bytes[:text.VirtualSize])
	require.NotZero(t, text.Characteristics&scnExec)
	require.Len(t, first.Imports, 1)
	require.Equal(t, "kernel32.dll", first.Imports[0].Library)
	require.Equal(t, Imported{Name: "GetProcAddress"}, first.Imports[0].Symbols[0])
	require.True(t, first.Imports[0].Symbols[1].ByOrdinal)
	require.Equal(t, uint16(15), first.Imports[0].Symbols[1].Ordinal)
	require.Equal(t, []Export{{Name: "Hello", Ordinal: 1, RVA: 0x1000}}, first.Exports)
	require.Equal(t, "export.dll", first.DllName)

	var buf bytes.Buffer
	require.NoError(t, first.Make(&buf))
	made := append([]byte(nil), buf.Bytes()...)
	parsed, err := pe.NewFile(bytes.NewReader(made))
	require.NoError(t, err)
	defer parsed.Close()
	syms, err := parsed.ImportedSymbols()
	require.NoError(t, err)
	require.Contains(t, syms, "GetProcAddress:kernel32.dll")

	second, err := Take(bytes.NewReader(made))
	require.NoError(t, err)
	require.Equal(t, first, second)
}

func TestComposedPE32(t *testing.T) {
	t.Parallel()
	file := New().Machine(I386).Entry(0x1000)
	file.Section(".text").Exec().Read().Bytes([]byte{0xc3})
	file.Import("kernel32.dll").Symbol("ExitProcess")
	first := takeMade(t, file)
	require.Equal(t, I386, first.Header.Machine)
	require.Equal(t, magic32, first.Magic)
	require.Equal(t, uint64(0x400000), first.Header.ImageBase)
	require.Equal(t, "ExitProcess", first.Imports[0].Symbols[0].Name)
	second := takeMade(t, first)
	require.Equal(t, first, second)
}

func TestRealPE(t *testing.T) {
	t.Parallel()
	goBin, err := exec.LookPath("go")
	if err != nil {
		test.Skip(t, "go is not installed")
	}
	for _, arch := range []string{"amd64", "386"} {
		t.Run(arch, func(t *testing.T) {
			t.Parallel()
			roundTripPath(t, buildGo(t, goBin, "windows", arch))
		})
	}
}

func TestRunMath(t *testing.T) {
	if runtime.GOOS != "windows" {
		test.Skipf(t, "running a PE needs windows, have %s/%s", runtime.GOOS, runtime.GOARCH)
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

	err = New().Section("").Make(io.Discard)
	require.ErrorIs(t, err, ErrName)
	err = New().Section("waytoolong").Make(io.Discard)
	require.ErrorIs(t, err, ErrName)
	err = New().Exec().Make(io.Discard)
	require.ErrorIs(t, err, ErrSection)
	err = New().Symbol("GetProcAddress").Make(io.Discard)
	require.ErrorIs(t, err, ErrImport)

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
	require.NotEmpty(t, first.Sections)
	var text []byte
	for _, section := range first.Sections {
		if section.Name == ".text" {
			text = section.Bytes
		}
	}
	require.NotEmpty(t, text)
	require.NotEmpty(t, first.Imports)
	var kernel bool
	for _, item := range first.Imports {
		if item.Library == "kernel32.dll" {
			kernel = len(item.Symbols) > 0
		}
	}
	require.True(t, kernel, "kernel32.dll import")

	var buf bytes.Buffer
	require.NoError(t, first.Make(&buf))
	made := append([]byte(nil), buf.Bytes()...)
	parsed, err := pe.NewFile(bytes.NewReader(made))
	require.NoError(t, err)
	defer parsed.Close()
	require.Equal(t, uint16(first.Header.Machine), parsed.Machine)

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
	for _, section := range file.Sections {
		if section.Name == name {
			return section
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
	out := filepath.Join(dir, "mathbin.exe")
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
	out := filepath.Join(t.TempDir(), "rewritten.exe")
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
