//go:build linux

package native

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestLibraryDepsSkipsLibc(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "libweb.so")
	body := elf64WithNeeded("$ORIGIN", "libc.so.6", "ld-linux-x86-64.so.2", "libgstreamer-1.0.so.0")
	if err := os.WriteFile(path, body, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "libc.so.6"), []byte("trap"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ld-linux-x86-64.so.2"), []byte("trap"), 0o644); err != nil {
		t.Fatal(err)
	}
	gst := filepath.Join(dir, "libgstreamer-1.0.so.0")
	if err := os.WriteFile(gst, []byte("gst"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := libraryDeps(path)
	if len(got) != 1 || got[0] != gst {
		t.Fatalf("deps %q", got)
	}
	names := elfNeeded(path)
	if len(names) != 3 || names[2] != "libgstreamer-1.0.so.0" {
		t.Fatalf("needed %q", names)
	}
}

func TestOpenLoadsSiblingBeforeLibraryPath(t *testing.T) {
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip(err)
	}
	id := time.Now().UnixNano()
	gstName := fmt.Sprintf("liblewkitgst%d.so", id)
	transName := fmt.Sprintf("liblewkittrans%d.so", id)
	webName := fmt.Sprintf("liblewkitweb%d.so", id)
	root := t.TempDir()
	oldDir := filepath.Join(root, "old")
	prefix := filepath.Join(root, "prefix")
	if err := os.MkdirAll(oldDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(prefix, 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(root, "src")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(name, body string) string {
		path := filepath.Join(src, name)
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	oldC := write("old.c", "void old_only(void) {}\n")
	newC := write("new.c", "void fake_gst_state_get_name(void) {}\n")
	transC := write("trans.c", "extern void fake_gst_state_get_name(void);\nvoid trans_init(void) { fake_gst_state_get_name(); }\n")
	webC := write("web.c", "extern void trans_init(void);\nvoid web_init(void) { trans_init(); }\n")
	oldGST := filepath.Join(oldDir, gstName)
	newGST := filepath.Join(prefix, gstName)
	trans := filepath.Join(prefix, transName)
	web := filepath.Join(prefix, webName)
	runGCC(t, "-shared", "-fPIC", "-Wl,-soname,"+gstName, "-o", oldGST, oldC)
	runGCC(t, "-shared", "-fPIC", "-Wl,-soname,"+gstName, "-o", newGST, newC)
	runGCC(t, "-shared", "-fPIC", "-Wl,-soname,"+transName, "-o", trans, transC, newGST, "-Wl,--enable-new-dtags,-rpath,$ORIGIN")
	runGCC(t, "-shared", "-fPIC", "-Wl,-soname,"+webName, "-o", web, webC, trans, "-Wl,--enable-new-dtags,-rpath,$ORIGIN")
	t.Setenv("LD_LIBRARY_PATH", oldDir)
	lib, err := Open(web, Global|Lazy)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Symbol(lib, "web_init"); err != nil {
		t.Fatal(err)
	}
}

func runGCC(t *testing.T, args ...string) {
	t.Helper()
	cmd := exec.Command("gcc", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gcc %v: %v\n%s", args, err, out)
	}
}

// elf64WithNeeded builds an ELF64 with those DT_NEEDED sonames and an optional
// DT_RUNPATH. runpath may be empty.
func elf64WithNeeded(runpath string, needed ...string) []byte {
	const (
		phoff  = 64
		phsize = 56
		phnum  = 2
	)
	dynCount := 1 + len(needed)
	if runpath != "" {
		dynCount++
	}
	dynOff := phoff + phsize*phnum
	dynSize := 16 * dynCount
	strOff := dynOff + dynSize
	var str []byte
	neededAt := make([]uint64, len(needed))
	for i, name := range needed {
		neededAt[i] = uint64(len(str))
		str = append(append(str, name...), 0)
	}
	var runAt uint64
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
	putPhdr(buf[phoff+phsize:], 2, uint64(dynOff), uint64(dynOff), uint64(dynSize))
	next := dynOff
	putDyn(buf[next:], 5, uint64(strOff))
	next += 16
	for _, at := range neededAt {
		putDyn(buf[next:], 1, at)
		next += 16
	}
	if runpath != "" {
		putDyn(buf[next:], 29, runAt)
	}
	copy(buf[strOff:], str)
	return buf
}
