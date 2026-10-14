//go:build linux && amd64

package clib

import (
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"encoding/binary"
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
	require.NoError(t, Run(t.Context(), Config{File: filepath.Join(dir, "add.h")}))
	lib := filepath.Join(dir, "libdemo.so")
	require.NoError(t, Build(t.Context(), dir, lib))

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

func TestPython(t *testing.T) {
	repo := moduleRoot(t)
	dir := t.TempDir()
	for _, name := range []string{"api.h", "main.go", "gorun.py", "run.py"} {
		buf, err := os.ReadFile(filepath.Join("testdata", "python", name))
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), buf, 0o644))
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/gorun\n\ngo 1.27.0\n\nrequire github.com/lewtec/lewkit v0.0.0\n\nreplace github.com/lewtec/lewkit => "+repo+"\n"), 0o644))
	require.NoError(t, Run(t.Context(), Config{
		File: filepath.Join(dir, "api.h"),
		ABI:  "linux/amd64",
	}))
	lib := filepath.Join(dir, "libgorun.so")
	require.NoError(t, Build(t.Context(), dir, lib))

	cmd := exec.Command("python3", filepath.Join(dir, "run.py"), lib)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	require.Contains(t, string(out), "ok")
}

func TestCrossLink(t *testing.T) {
	repo := moduleRoot(t)
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/demo\n\ngo 1.27.0\n\nrequire github.com/lewtec/lewkit v0.0.0\n\nreplace github.com/lewtec/lewkit => "+repo+"\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "add.h"), []byte("#include <stdint.h>\nint32_t add(int32_t a, int32_t b);\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644))
	require.NoError(t, Run(t.Context(), Config{File: filepath.Join(dir, "add.h")}))

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
			require.NoError(t, BuildFor(t.Context(), dir, lib, target.goos, target.goarch))
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

// TestDesktopLink builds a Windows DLL and a Darwin dylib with the internal
// linker and reads the export back. This host does not load either image.
func TestDesktopLink(t *testing.T) {
	repo := moduleRoot(t)
	writeDemo := func(t *testing.T, abi string) string {
		t.Helper()
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/demo\n\ngo 1.27.0\n\nrequire github.com/lewtec/lewkit v0.0.0\n\nreplace github.com/lewtec/lewkit => "+repo+"\n"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "add.h"), []byte("#include <stdint.h>\nint32_t add(int32_t a, int32_t b);\n"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644))
		require.NoError(t, Run(t.Context(), Config{File: filepath.Join(dir, "add.h"), ABI: abi}))
		return dir
	}

	t.Run("windows", func(t *testing.T) {
		dir := writeDemo(t, "windows")
		cases := []struct {
			arch    string
			machine uint16
			stub    []byte
		}{
			{"amd64", pe.IMAGE_FILE_MACHINE_AMD64, []byte{0xb8, 0x01, 0x00, 0x00, 0x00, 0xc3}},
			{"arm64", pe.IMAGE_FILE_MACHINE_ARM64, []byte{0x20, 0x00, 0x80, 0x52, 0xc0, 0x03, 0x5f, 0xd6}},
			{"386", pe.IMAGE_FILE_MACHINE_I386, []byte{0xb8, 0x01, 0x00, 0x00, 0x00, 0xc2, 0x0c, 0x00}},
		}
		for _, tc := range cases {
			t.Run(tc.arch, func(t *testing.T) {
				lib := filepath.Join(dir, "demo_"+tc.arch+".dll")
				require.NoError(t, BuildFor(t.Context(), dir, lib, "windows", tc.arch))
				f, err := pe.Open(lib)
				require.NoError(t, err)
				defer f.Close()
				require.Equal(t, tc.machine, f.Machine)
				require.NotZero(t, f.Characteristics&pe.IMAGE_FILE_DLL)
				require.NotZero(t, f.Characteristics&pe.IMAGE_FILE_EXECUTABLE_IMAGE)
				raw, err := os.ReadFile(lib)
				require.NoError(t, err)
				entry, dir := peEntry(f)
				off, ok := peOff(f, entry)
				require.True(t, ok)
				require.Equal(t, tc.stub, raw[off:off+len(tc.stub)])
				names := peExportNames(t, f, raw, dir)
				require.Contains(t, names, "add")
				imps, err := f.ImportedSymbols()
				require.NoError(t, err)
				require.Contains(t, imps, "CreatePipe:kernel32.dll")
				require.Contains(t, imps, "WriteFile:kernel32.dll")
				require.Contains(t, imps, "WaitForSingleObject:kernel32.dll")
			})
		}
	})

	t.Run("darwin", func(t *testing.T) {
		dir := writeDemo(t, "darwin")
		cases := []struct {
			arch string
			cpu  macho.Cpu
		}{
			{"amd64", macho.CpuAmd64},
			{"arm64", macho.CpuArm64},
		}
		for _, tc := range cases {
			t.Run(tc.arch, func(t *testing.T) {
				lib := filepath.Join(dir, "libdemo_"+tc.arch+".dylib")
				require.NoError(t, BuildFor(t.Context(), dir, lib, "darwin", tc.arch))
				f, err := macho.Open(lib)
				require.NoError(t, err)
				defer f.Close()
				require.Equal(t, macho.TypeDylib, f.Type)
				require.Equal(t, tc.cpu, f.Cpu)
				require.Zero(t, f.Flags&0x200000)
				raw, err := os.ReadFile(lib)
				require.NoError(t, err)
				var sawID, sawSig bool
				var trie []byte
				for _, load := range f.Loads {
					cmdb := load.Raw()
					if len(cmdb) < 8 {
						continue
					}
					cmd := binary.LittleEndian.Uint32(cmdb)
					switch cmd {
					case 0xd:
						sawID = true
						require.Contains(t, string(cmdb), "@rpath/libdemo_"+tc.arch+".dylib")
					case 0x5, 0xe, 0x80000028:
						require.FailNowf(t, "", "executable load command %#x", cmd)
					case 0x22, 0x80000022:
						require.GreaterOrEqual(t, len(cmdb), 48)
						off := binary.LittleEndian.Uint32(cmdb[40:])
						sz := binary.LittleEndian.Uint32(cmdb[44:])
						require.NotZero(t, sz)
						require.LessOrEqual(t, int(off)+int(sz), len(raw))
						trie = raw[off : off+sz]
					case 0x1d:
						require.GreaterOrEqual(t, len(cmdb), 16)
						off := binary.LittleEndian.Uint32(cmdb[8:])
						require.GreaterOrEqual(t, len(raw), int(off)+4)
						require.Equal(t, uint32(0xfade0cc0), binary.BigEndian.Uint32(raw[off:]))
						sawSig = true
					}
				}
				require.True(t, sawID)
				require.True(t, sawSig)
				addr, ok := trieLookup(trie, "add")
				require.True(t, ok, "export trie names add")
				require.NotZero(t, addr)
			})
		}
	})
}

func peEntry(f *pe.File) (uint32, pe.DataDirectory) {
	switch oh := f.OptionalHeader.(type) {
	case *pe.OptionalHeader64:
		return oh.AddressOfEntryPoint, oh.DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_EXPORT]
	case *pe.OptionalHeader32:
		return oh.AddressOfEntryPoint, oh.DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_EXPORT]
	default:
		return 0, pe.DataDirectory{}
	}
}

func peOff(f *pe.File, rva uint32) (int, bool) {
	for _, s := range f.Sections {
		span := s.VirtualSize
		if s.Size > span {
			span = s.Size
		}
		if rva < s.VirtualAddress || rva >= s.VirtualAddress+span {
			continue
		}
		return int(s.Offset) + int(rva-s.VirtualAddress), true
	}
	return 0, false
}

func peExportNames(t *testing.T, f *pe.File, raw []byte, dir pe.DataDirectory) []string {
	t.Helper()
	require.NotZero(t, dir.Size)
	off, ok := peOff(f, dir.VirtualAddress)
	require.True(t, ok)
	require.GreaterOrEqual(t, len(raw), off+40)
	n := binary.LittleEndian.Uint32(raw[off+24:])
	namesRVA := binary.LittleEndian.Uint32(raw[off+32:])
	namesOff, ok := peOff(f, namesRVA)
	require.True(t, ok)
	names := make([]string, 0, n)
	for i := uint32(0); i < n; i++ {
		at := namesOff + int(i)*4
		require.LessOrEqual(t, at+4, len(raw))
		nrva := binary.LittleEndian.Uint32(raw[at:])
		noff, ok := peOff(f, nrva)
		require.True(t, ok)
		end := noff
		for end < len(raw) && raw[end] != 0 {
			end++
		}
		names = append(names, string(raw[noff:end]))
	}
	return names
}

// TestEntryLinks proves the 386 and arm entry against the real runtime.
// Those pairs have no internal PIE, so BuildFor returns ErrHost before
// the linker. The test binary still resolves runtime.clone, runtime.rt0_go,
// and runtime.islibrary.
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
