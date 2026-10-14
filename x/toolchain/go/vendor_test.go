package gotool

import (
	"debug/elf"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	if handled, err := Child(); handled {
		if err != nil {
			os.Stderr.WriteString(err.Error() + "\n")
			os.Exit(1)
		}
		os.Exit(0)
	}
	ensureToolchainEnv()
	os.Exit(m.Run())
}

func TestVendoredExe(t *testing.T) {
	reexec(t, "TestVendoredExe", func(t *testing.T) {
		work := workdir(t)
		src := writeHello(t, work, false)
		cfg, _ := listRuntime(t, work)
		archive := filepath.Join(work, "exe.a")
		mode, shared := hostImage()
		require.NoError(t, Compile(t.Context(), "main", archive, cfg, "", "", shared, []string{src}))
		bin := hostBin(filepath.Join(work, "hello"))
		require.NoError(t, Link(t.Context(), bin, linkImport(t, work, "exe.a"), mode, archive))
		out, err := exec.Command(bin).CombinedOutput()
		require.NoErrorf(t, err, "run: %v\n%s", err, out)
		if string(out) != "toolchain-ok\n" {
			require.FailNowf(t, "", "output %q", out)
		}
	})
}

func TestVendoredPIE(t *testing.T) {
	if runtime.GOOS != "linux" {
		test.Skip(t, "shared object is an ELF file")
	}
	reexec(t, "TestVendoredPIE", func(t *testing.T) {
		work := workdir(t)
		hello := writeHello(t, work, false)
		export := writeHello(t, work, true)
		cfg, _ := listRuntime(t, work)
		archive := filepath.Join(work, "pie.a")
		require.NoError(t, Compile(t.Context(), "main", archive, cfg, "", "", true, []string{hello, export}))
		bin := filepath.Join(work, "hello.pie")
		require.NoError(t, Link(t.Context(), bin, linkImport(t, work, "pie.a"), "pie", archive))
		out, err := exec.Command(bin).CombinedOutput()
		require.NoErrorf(t, err, "pie run: %v\n%s", err, out)
		if string(out) != "toolchain-ok\n" {
			require.FailNowf(t, "", "pie output %q", out)
		}
		if !dynDefined(t, bin, "ToolchainPing") {
			require.FailNow(t, fmt.Sprint("ToolchainPing is not in .dynsym"))
		}
		if !progType(t, bin, elf.PT_INTERP) {
			require.FailNow(t, fmt.Sprint("PIE has no PT_INTERP to clear"))
		}
		if elf.DynFlag1(dynFlags1(t, bin))&elf.DF_1_PIE == 0 {
			require.FailNow(t, fmt.Sprint("PIE has no DF_1_PIE to clear"))
		}
		require.NoError(t, MakeShared(bin))
		f := openELF(t, bin)
		if f.Type != elf.ET_DYN {
			require.FailNowf(t, "", "type %s", f.Type)
		}
		f.Close()
		if progType(t, bin, elf.PT_INTERP) {
			require.FailNow(t, fmt.Sprint("PT_INTERP survived MakeShared"))
		}
		if elf.DynFlag1(dynFlags1(t, bin))&elf.DF_1_PIE != 0 {
			require.FailNow(t, fmt.Sprint("DF_1_PIE survived MakeShared"))
		}
		if !dynDefined(t, bin, "ToolchainPing") {
			require.FailNow(t, fmt.Sprint("ToolchainPing dropped out of .dynsym"))
		}
	})
}

func TestVendoredCShared(t *testing.T) {
	reexec(t, "TestVendoredCShared", func(t *testing.T) {
		work := workdir(t)
		src := writeHello(t, work, false)
		cfg, _ := listRuntime(t, work)
		archive := filepath.Join(work, "exe.a")
		require.NoError(t, Compile(t.Context(), "main", archive, cfg, "", "", false, []string{src}))
		cmd := exec.Command("go", "build", "-buildmode=c-shared", "-o", filepath.Join(work, "via-go.so"), ".")
		cmd.Dir = work
		out, err := cmd.CombinedOutput()
		require.Error(t, err, "go build -buildmode=c-shared succeeded with cgo off")
		if !strings.Contains(string(out), "cgo") {
			require.FailNowf(t, "", "cmd/go rejection was %s", out)
		}
		err = Link(t.Context(), filepath.Join(work, "direct.so"), linkImport(t, work, "exe.a"), "c-shared", archive)
		require.Error(t, err, "in-process c-shared link succeeded")
		if !strings.Contains(err.Error(), "external linking required") {
			require.FailNowf(t, "", "linker rejection was %v", err)
		}
	})
}

func hostImage() (mode string, shared bool) {
	if runtime.GOOS == "darwin" || (runtime.GOOS == "windows" && runtime.GOARCH == "arm64") {
		return "pie", true
	}
	return "exe", false
}

func hostBin(path string) string {
	if runtime.GOOS == "windows" {
		return path + ".exe"
	}
	return path
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

func writeHello(t *testing.T, work string, export bool) string {
	t.Helper()
	if !export {
		path := filepath.Join(work, "hello.go")
		body := "package main\n\nfunc main() { println(\"toolchain-ok\") }\n"
		require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
		return path
	}
	path := filepath.Join(work, "_cgo_export.go")
	body := strings.Join([]string{
		"package main",
		"",
		"//go:cgo_export_dynamic main.ToolchainPing ToolchainPing",
		"",
		"func ToolchainPing() byte { return 7 }",
		"",
	}, "\n")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
	return path
}

func workdir(t *testing.T) string {
	t.Helper()
	work := os.Getenv("TOOLCHAIN_WORK")
	if work == "" {
		require.FailNow(t, fmt.Sprint("TOOLCHAIN_WORK is empty"))
	}
	return work
}

func reexec(t *testing.T, name string, fn func(*testing.T)) {
	t.Helper()
	if os.Getenv("TOOLCHAIN_CHILD") == "1" {
		fn(t)
		return
	}
	work := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=^"+name+"$", "-test.v")
	cmd.Env = append(os.Environ(), "TOOLCHAIN_CHILD=1", "TOOLCHAIN_WORK="+work)
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "%v\n%s", err, out)
	t.Log(string(out))
}

func listRuntime(t *testing.T, work string) (compileCfg, deps string) {
	t.Helper()
	compileCfg, deps, err := runtimeDeps(work)
	require.NoError(t, err)
	return compileCfg, deps
}

func linkImport(t *testing.T, work, archiveName string) string {
	t.Helper()
	deps, err := os.ReadFile(filepath.Join(work, "deps"))
	require.NoError(t, err)
	archive := filepath.Join(work, archiveName)
	body := "# import config\npackagefile main=" + archive + "\n" + string(deps)
	path := archive + ".linkimport"
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
	return path
}

func openELF(t *testing.T, path string) *elf.File {
	t.Helper()
	f, err := elf.Open(path)
	require.NoError(t, err)
	return f
}

func dynDefined(t *testing.T, path, name string) bool {
	t.Helper()
	f := openELF(t, path)
	defer f.Close()
	syms, err := f.DynamicSymbols()
	require.NoError(t, err)
	for _, sym := range syms {
		if sym.Name == name && elf.ST_BIND(sym.Info) == elf.STB_GLOBAL && sym.Section != elf.SHN_UNDEF {
			return true
		}
	}
	return false
}

func progType(t *testing.T, path string, kind elf.ProgType) bool {
	t.Helper()
	f := openELF(t, path)
	defer f.Close()
	for _, prog := range f.Progs {
		if prog.Type == kind {
			return true
		}
	}
	return false
}

func dynFlags1(t *testing.T, path string) uint64 {
	t.Helper()
	f := openELF(t, path)
	defer f.Close()
	vals, err := f.DynValue(elf.DT_FLAGS_1)
	if err != nil || len(vals) == 0 {
		require.FailNowf(t, "", "DT_FLAGS_1: %v", err)
	}
	return vals[0]
}
