//go:build linux

package native

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestElfRunpathReadsStorePaths(t *testing.T) {
	path := filepath.Join(t.TempDir(), "zenity")
	if err := os.WriteFile(path, elf64WithPaths("/nix/store/gtk/lib:/nix/store/glib/lib", ""), 0o755); err != nil {
		t.Fatal(err)
	}
	got := elfRunpath(path)
	want := []string{"/nix/store/gtk/lib", "/nix/store/glib/lib"}
	if len(got) != len(want) {
		t.Fatalf("runpath %q", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("runpath %q", got)
		}
	}
}

func TestElfRunpathExpandsOrigin(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "tool")
	if err := os.WriteFile(path, elf64WithPaths("$ORIGIN/../lib:${ORIGIN}/extra", ""), 0o755); err != nil {
		t.Fatal(err)
	}
	got := elfRunpath(path)
	want := []string{filepath.Join(dir, "../lib"), filepath.Join(dir, "extra")}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("runpath %q", got)
	}
}

func TestElfRunpathPrefersRunpathOverRpath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tool")
	if err := os.WriteFile(path, elf64WithPaths("/run/only", "/rpath/ignored"), 0o755); err != nil {
		t.Fatal(err)
	}
	got := elfRunpath(path)
	if len(got) != 1 || got[0] != "/run/only" {
		t.Fatalf("runpath %q", got)
	}
	rpath := filepath.Join(t.TempDir(), "rpath")
	if err := os.WriteFile(rpath, elf64WithPaths("", "/rpath/only"), 0o755); err != nil {
		t.Fatal(err)
	}
	got = elfRunpath(rpath)
	if len(got) != 1 || got[0] != "/rpath/only" {
		t.Fatalf("rpath %q", got)
	}
}

func TestRunpathsFollowsWrappedBinary(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "zenity")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexec \"$0\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	wrapped := filepath.Join(dir, ".zenity-wrapped")
	if err := os.WriteFile(wrapped, elf64WithPaths("/nix/store/gtk4/lib", ""), 0o755); err != nil {
		t.Fatal(err)
	}
	got := runpathsOf(script)
	if len(got) != 1 || got[0] != "/nix/store/gtk4/lib" {
		t.Fatalf("runpath %q", got)
	}
}

func TestNixStoreBinPattern(t *testing.T) {
	text := []byte("exec /nix/store/abc-zenity-4/bin/.zenity-wrapped \"$@\"\n")
	got := nixStoreBin.FindAll(text, -1)
	if len(got) != 1 || string(got[0]) != "/nix/store/abc-zenity-4/bin/.zenity-wrapped" {
		t.Fatalf("bins %q", got)
	}
}

func TestElfRunpathIgnoresPlainFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "text")
	if err := os.WriteFile(path, []byte("not elf"), 0o644); err != nil {
		t.Fatal(err)
	}
	if elfRunpath(path) != nil {
		t.Fatal("plain file had a runpath")
	}
}

// elf64WithPaths builds a tiny ELF64 whose RUNPATH and RPATH are those strings.
// An empty string omits that tag. RUNPATH wins when both are set.
func elf64WithPaths(runpath, rpath string) []byte {
	const (
		phoff   = 64
		phsize  = 56
		phnum   = 2
		dynOff  = phoff + phsize*phnum
		dynSize = 16 * 4
		strOff  = dynOff + dynSize
	)
	var str []byte
	var runAt, rAt uint64
	if rpath != "" {
		rAt = uint64(len(str))
		str = append(append(str, rpath...), 0)
	}
	if runpath != "" {
		runAt = uint64(len(str))
		str = append(append(str, runpath...), 0)
	}
	buf := make([]byte, strOff+len(str))
	copy(buf, []byte{0x7f, 'E', 'L', 'F', 2, 1, 1})
	binary.LittleEndian.PutUint64(buf[32:], phoff)
	binary.LittleEndian.PutUint16(buf[52:], 64)
	binary.LittleEndian.PutUint16(buf[54:], phsize)
	binary.LittleEndian.PutUint16(buf[56:], phnum)
	putPhdr(buf[phoff:], 1, 0, 0, uint64(len(buf)))
	putPhdr(buf[phoff+phsize:], 2, dynOff, dynOff, dynSize)
	putDyn(buf[dynOff:], 5, strOff)
	next := dynOff + 16
	if rpath != "" {
		putDyn(buf[next:], 15, rAt)
		next += 16
	}
	if runpath != "" {
		putDyn(buf[next:], 29, runAt)
	}
	copy(buf[strOff:], str)
	return buf
}

func putPhdr(b []byte, typ uint32, off, vaddr, size uint64) {
	binary.LittleEndian.PutUint32(b[0:], typ)
	binary.LittleEndian.PutUint64(b[8:], off)
	binary.LittleEndian.PutUint64(b[16:], vaddr)
	binary.LittleEndian.PutUint64(b[32:], size)
	binary.LittleEndian.PutUint64(b[40:], size)
}

func putDyn(b []byte, tag, val uint64) {
	binary.LittleEndian.PutUint64(b[0:], tag)
	binary.LittleEndian.PutUint64(b[8:], val)
}
