package decomp

import (
	"context"
	"crypto/sha256"
	"debug/elf"
	"encoding/binary"
	"encoding/hex"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/disasm"
	xdecomp "github.com/lewtec/lewkit/x/disasm/decomp"
	"github.com/lewtec/lewkit/x/release"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecompUsage(t *testing.T) {
	text, err := cmd.Usage[Cmd](release.Name() + " decomp")
	require.NoError(t, err)
	assert.Contains(t, text, "summary")
	assert.Contains(t, text, "functions")
	assert.Contains(t, text, "listing")
	assert.Contains(t, text, "stream")
	assert.Contains(t, text, "xrefs")
	assert.Contains(t, text, "strings")
	assert.Contains(t, text, "check")
	assert.Contains(t, text, "name")
	assert.Contains(t, text, "note")
	assert.Contains(t, text, "signature")
	assert.Contains(t, text, "view")
	assert.Contains(t, text, "project")
}

func TestDecompViewUsage(t *testing.T) {
	test.RestoreSlog(t)
	view := cmd.ParseOK[cmd.App[Cmd]](t, "view", "--help")
	got := test.Stdout(t, func() {
		require.NoError(t, view.Run(t.Context()))
	})
	assert.Contains(t, got, "listen")
	assert.Contains(t, got, "preview the program database")
}

func TestDecompNameAndNoteStayInTheProject(t *testing.T) {
	test.RestoreSlog(t)
	path := writeELF(t, []byte{0x90, 0xc3})
	database := filepath.Join(t.TempDir(), "program.decomp")
	load := cmd.ParseOK[cmd.App[Cmd]](t, "--project", database, "functions", path)
	require.NoError(t, load.Run(t.Context()))

	rename := cmd.ParseOK[cmd.App[Cmd]](t, "--project", database, "name", "--at", "0x1000", "--name", "greet", path)
	got := test.Stdout(t, func() {
		require.NoError(t, rename.Run(t.Context()))
	})
	assert.Contains(t, got, "greet")

	note := cmd.ParseOK[cmd.App[Cmd]](t, "--project", database, "note", "--at", "0x1001", "--text", "padding", path)
	got = test.Stdout(t, func() {
		require.NoError(t, note.Run(t.Context()))
	})
	assert.Contains(t, got, "padding")
	require.NoError(t, os.Remove(path))

	listing := cmd.ParseOK[cmd.App[Cmd]](t, "--project", database, "listing", "--at", "0x1000", path)
	got = test.Stdout(t, func() {
		require.NoError(t, listing.Run(t.Context()))
	})
	assert.Contains(t, got, "greet")
	assert.Contains(t, got, "; note padding")
}

func TestDecompViewNeedsProject(t *testing.T) {
	test.RestoreSlog(t)
	view := cmd.ParseOK[cmd.App[Cmd]](t, "view")
	err := view.Run(t.Context())
	require.Error(t, err)
	assert.ErrorIs(t, err, errProjectRequired)
}

func TestDecompViewServesTheProject(t *testing.T) {
	test.RestoreSlog(t)
	path := writeELF(t, []byte{0x90, 0xc3})
	database := filepath.Join(t.TempDir(), "program.decomp")
	load := cmd.ParseOK[cmd.App[Cmd]](t, "--project", database, "functions", path)
	require.NoError(t, load.Run(t.Context()))

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := listener.Addr().String()
	require.NoError(t, listener.Close())

	view := cmd.ParseOK[cmd.App[Cmd]](t, "--project", database, "view", "--listen", address)
	ctx, cancel := context.WithCancel(t.Context())
	errc := make(chan error, 1)
	go func() {
		errc <- view.Run(ctx)
	}()

	var response *http.Response
	deadline := time.Now().Add(2 * time.Second)
	for {
		response, err = http.Get("http://" + address + "/")
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			cancel()
			require.NoError(t, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	body, err := io.ReadAll(response.Body)
	require.NoError(t, response.Body.Close())
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Contains(t, string(body), "a.out")
	assert.Contains(t, string(body), "sub_1000")
	assert.Contains(t, string(body), "data-cmdpalette")

	cancel()
	require.NoError(t, <-errc)
}

func TestDecompFunctionsAndListing(t *testing.T) {
	test.RestoreSlog(t)
	path := writeELF(t, []byte{0x90, 0xc3})
	functions := cmd.ParseOK[cmd.App[Cmd]](t, "functions", path)
	got := test.Stdout(t, func() {
		require.NoError(t, functions.Run(t.Context()))
	})
	assert.Contains(t, got, "0x1000")
	assert.Contains(t, got, "sub_1000")

	listing := cmd.ParseOK[cmd.App[Cmd]](t, "listing", "--at", "0x1000", path)
	got = test.Stdout(t, func() {
		require.NoError(t, listing.Run(t.Context()))
	})
	assert.Contains(t, got, "nop")
	assert.Contains(t, got, "ret")
}

func TestDecompProjectKeepsFunctions(t *testing.T) {
	test.RestoreSlog(t)
	path := writeELF(t, []byte{0x90, 0xc3})
	database := filepath.Join(t.TempDir(), "program.decomp")
	first := cmd.ParseOK[cmd.App[Cmd]](t, "--project", database, "functions", path)
	got := test.Stdout(t, func() {
		require.NoError(t, first.Run(t.Context()))
	})
	assert.Contains(t, got, "sub_1000")
	require.NoError(t, os.Remove(path))
	second := cmd.ParseOK[cmd.App[Cmd]](t, "--project", database, "functions", path)
	got = test.Stdout(t, func() {
		require.NoError(t, second.Run(t.Context()))
	})
	assert.Contains(t, got, "sub_1000")
	assert.Contains(t, got, "0x1000")
}

func TestDecompCheckComparesTheCallerCandidate(t *testing.T) {
	test.RestoreSlog(t)
	path := writeELF(t, []byte{0xc3})
	candidate := `{"name":"sub_1000","signature":"void sub_1000(void)","source":"void sub_1000(void) {\n  return;\n}\n","blocks":[{"address":"0x1000","kind":"return"}]}`
	app := cmd.ParseOK[cmd.App[Cmd]](t, "check", "--at", "0x1000", "--text", candidate, path)
	got := test.Stdout(t, func() {
		require.NoError(t, app.Run(t.Context()))
	})
	assert.Contains(t, got, "matched")
	assert.Contains(t, got, "sub_1000")
}

func TestDecompProjectRemembersEachFile(t *testing.T) {
	if os.Getenv("DECOMP_CLI_RELOAD") == "1" {
		test.RestoreSlog(t)
		for _, image := range strings.Split(os.Getenv("DECOMP_IMAGES"), "\n") {
			summary := cmd.ParseOK[cmd.App[Cmd]](t, "--project", os.Getenv("DECOMP_PROJECT"), "summary", image)
			require.NoError(t, summary.Run(t.Context()))
		}
		return
	}

	test.RestoreSlog(t)
	dir := t.TempDir()
	firstPath := filepath.Join(dir, "one", "setup.exe")
	secondPath := filepath.Join(dir, "two", "setup.exe")
	require.NoError(t, os.MkdirAll(filepath.Dir(firstPath), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Dir(secondPath), 0o755))
	first := disasm.ELF64(elf.EM_X86_64, []byte{0xc3})
	second := disasm.ELF64(elf.EM_X86_64, []byte{0x90, 0xc3})
	require.NoError(t, os.WriteFile(firstPath, first, 0o644))
	require.NoError(t, os.WriteFile(secondPath, second, 0o644))
	database := filepath.Join(dir, "program.decomp")

	loadFirst := cmd.ParseOK[cmd.App[Cmd]](t, "--project", database, "summary", firstPath)
	got := test.Stdout(t, func() {
		require.NoError(t, loadFirst.Run(t.Context()))
	})
	assert.Contains(t, got, ".text 0x1")

	loadSecond := cmd.ParseOK[cmd.App[Cmd]](t, "--project", database, "summary", secondPath)
	got = test.Stdout(t, func() {
		require.NoError(t, loadSecond.Run(t.Context()))
	})
	assert.Contains(t, got, ".text 0x2")

	changed := disasm.ELF64(elf.EM_X86_64, []byte{0x90, 0x90, 0xc3})
	require.NoError(t, os.WriteFile(firstPath, changed, 0o644))
	again := cmd.ParseOK[cmd.App[Cmd]](t, "--project", database, "summary", firstPath)
	err := again.Run(t.Context())
	require.Error(t, err)
	assert.ErrorIs(t, err, xdecomp.ErrImage)
	assert.Contains(t, err.Error(), "stored image")
	assert.Contains(t, err.Error(), "new bytes")
	assert.Contains(t, err.Error(), digestOf(first))
	assert.Contains(t, err.Error(), digestOf(changed))
	require.NoError(t, os.WriteFile(firstPath, first, 0o644))

	command := exec.Command(os.Args[0], "-test.run", "^TestDecompProjectRemembersEachFile$")
	command.Env = append(os.Environ(),
		"DECOMP_CLI_RELOAD=1",
		"DECOMP_PROJECT="+database,
		"DECOMP_IMAGES="+firstPath+"\n"+secondPath,
	)
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
	text := string(output)
	assert.Contains(t, text, ".text 0x1")
	assert.Contains(t, text, ".text 0x2")
	assert.NotContains(t, strings.ToLower(text), "capstone")
}

func TestDecompNameAndNoteNeedAProject(t *testing.T) {
	test.RestoreSlog(t)
	path := writeELF(t, []byte{0xe8, 0x01, 0x00, 0x00, 0x00, 0xc3, 0xc3})

	rename := cmd.ParseOK[cmd.App[Cmd]](t, "name", "--at", "0x1006", "--name", "StringAssign", path)
	err := rename.Run(t.Context())
	require.Error(t, err)
	assert.ErrorIs(t, err, xdecomp.ErrNotStored)
	assert.Contains(t, err.Error(), "name was not stored")

	functions := cmd.ParseOK[cmd.App[Cmd]](t, "functions", path)
	got := test.Stdout(t, func() {
		require.NoError(t, functions.Run(t.Context()))
	})
	assert.Contains(t, got, "0x1006")
	assert.Contains(t, got, "sub_1006")
	assert.NotContains(t, got, "StringAssign")

	note := cmd.ParseOK[cmd.App[Cmd]](t, "note", "--at", "0x1006", "--text", "assign", path)
	err = note.Run(t.Context())
	require.Error(t, err)
	assert.ErrorIs(t, err, xdecomp.ErrNotStored)
	assert.Contains(t, err.Error(), "note was not stored")

	database := filepath.Join(t.TempDir(), "program.decomp")
	stored := cmd.ParseOK[cmd.App[Cmd]](t, "--project", database, "name", "--at", "0x1006", "--name", "StringAssign", path)
	got = test.Stdout(t, func() {
		require.NoError(t, stored.Run(t.Context()))
	})
	assert.Contains(t, got, "StringAssign")

	listing := cmd.ParseOK[cmd.App[Cmd]](t, "--project", database, "listing", "--at", "0x1000", path)
	got = test.Stdout(t, func() {
		require.NoError(t, listing.Run(t.Context()))
	})
	assert.Contains(t, got, "call 0x1006 <StringAssign>")

	noted := cmd.ParseOK[cmd.App[Cmd]](t, "--project", database, "note", "--at", "0x1006", "--text", "assign", path)
	got = test.Stdout(t, func() {
		require.NoError(t, noted.Run(t.Context()))
	})
	assert.Contains(t, got, "assign")

	printed := cmd.ParseOK[cmd.App[Cmd]](t, "--project", database, "note", "--at", "0x1006", path)
	got = test.Stdout(t, func() {
		require.NoError(t, printed.Run(t.Context()))
	})
	assert.Contains(t, got, "assign")
}

func TestDecompNotesFollowUses(t *testing.T) {
	test.RestoreSlog(t)
	path := writeELF(t, []byte{0xe8, 0x01, 0x00, 0x00, 0x00, 0xc3, 0xc3})
	database := filepath.Join(t.TempDir(), "program.decomp")
	load := cmd.ParseOK[cmd.App[Cmd]](t, "--project", database, "functions", path)
	require.NoError(t, load.Run(t.Context()))

	first := cmd.ParseOK[cmd.App[Cmd]](t, "--project", database, "note", "--at", "0x1006", "--text", "kernel32!CloseHandle", path)
	require.NoError(t, first.Run(t.Context()))
	second := cmd.ParseOK[cmd.App[Cmd]](t, "--project", database, "note", "--at", "0x1001", "--text", "saved clue", path)
	require.NoError(t, second.Run(t.Context()))

	listing := cmd.ParseOK[cmd.App[Cmd]](t, "--project", database, "listing", "--at", "0x1000", path)
	got := test.Stdout(t, func() {
		require.NoError(t, listing.Run(t.Context()))
	})
	assert.Contains(t, got, "; note kernel32!CloseHandle")

	bare := cmd.ParseOK[cmd.App[Cmd]](t, "--project", database, "note", path)
	got = test.Stdout(t, func() {
		require.NoError(t, bare.Run(t.Context()))
	})
	assert.Contains(t, got, "0x1006 kernel32!CloseHandle")
	assert.Contains(t, got, "0x1001 saved clue")

	clear := cmd.ParseOK[cmd.App[Cmd]](t, "--project", database, "note", "--at", "0x1006", "--text", "", path)
	require.NoError(t, clear.Run(t.Context()))
	bare = cmd.ParseOK[cmd.App[Cmd]](t, "--project", database, "note", path)
	got = test.Stdout(t, func() {
		require.NoError(t, bare.Run(t.Context()))
	})
	assert.NotContains(t, got, "0x1006")
	assert.Contains(t, got, "0x1001 saved clue")
}

func TestDecompEntryCallersAndStrings(t *testing.T) {
	test.RestoreSlog(t)
	path := writeELFAt(t, []byte{0x55, 0xc3}, 0x1000)
	listing := cmd.ParseOK[cmd.App[Cmd]](t, "listing", path)
	got := test.Stdout(t, func() {
		require.NoError(t, listing.Run(t.Context()))
	})
	assert.Contains(t, got, "0x1000")
	assert.Contains(t, got, "push")

	caller := writeELF(t, []byte{0xe8, 0x01, 0x00, 0x00, 0x00, 0xc3, 0xc3})
	xrefs := cmd.ParseOK[cmd.App[Cmd]](t, "xrefs", "--at", "0x1006", caller)
	got = test.Stdout(t, func() {
		require.NoError(t, xrefs.Run(t.Context()))
	})
	assert.Contains(t, got, "0x1000 call")

	rodata := append([]byte("System"), 0)
	rodata = append(rodata, []byte("SysUtils")...)
	rodata = append(rodata, 0)
	image := writeELFTextAndRodata(t, []byte{0x48, 0x8d, 0x3d, 0xf9, 0x0f, 0x00, 0x00, 0xc3}, rodata)
	stringsCmd := cmd.ParseOK[cmd.App[Cmd]](t, "strings", image)
	got = test.Stdout(t, func() {
		require.NoError(t, stringsCmd.Run(t.Context()))
	})
	assert.Contains(t, got, "0x2000 System")
	assert.Contains(t, got, "SysUtils")
	stringListing := cmd.ParseOK[cmd.App[Cmd]](t, "listing", "--at", "0x1000", image)
	got = test.Stdout(t, func() {
		require.NoError(t, stringListing.Run(t.Context()))
	})
	assert.Contains(t, got, `; string "System"`)
}

func TestDecompCheckKeepsAStoredSource(t *testing.T) {
	test.RestoreSlog(t)
	path := writeELF(t, []byte{0xc3})
	database := filepath.Join(t.TempDir(), "program.decomp")
	good := `{"name":"sub_1000","signature":"void sub_1000(void)","source":"void sub_1000(void) { return; }","blocks":[{"address":"0x1000","kind":"return"}]}`
	check := cmd.ParseOK[cmd.App[Cmd]](t, "--project", database, "check", "--at", "0x1000", "--text", good, path)
	got := test.Stdout(t, func() {
		require.NoError(t, check.Run(t.Context()))
	})
	assert.Contains(t, got, "void sub_1000(void) { return; }")

	bad := `{"name":"sub_1000","signature":"void sub_1000(void)","source":"void sub_1000(void) { return; }","blocks":[]}`
	rejected := cmd.ParseOK[cmd.App[Cmd]](t, "--project", database, "check", "--at", "0x1000", "--text", bad, path)
	err := rejected.Run(t.Context())
	require.Error(t, err)
	assert.ErrorIs(t, err, xdecomp.ErrUnmatched)

	listing := cmd.ParseOK[cmd.App[Cmd]](t, "--project", database, "listing", "--at", "0x1000", path)
	got = test.Stdout(t, func() {
		require.NoError(t, listing.Run(t.Context()))
	})
	assert.Contains(t, got, "void sub_1000(void) { return; }")

	missing := cmd.ParseOK[cmd.App[Cmd]](t, "signature", "--at", "0x1000", "--text", "void entry(void)", path)
	err = missing.Run(t.Context())
	require.Error(t, err)
	assert.ErrorIs(t, err, xdecomp.ErrNotStored)
	assert.Contains(t, err.Error(), "signature was not stored")

	signature := cmd.ParseOK[cmd.App[Cmd]](t, "--project", database, "signature", "--at", "0x1000", "--text", "void entry(void)", path)
	got = test.Stdout(t, func() {
		require.NoError(t, signature.Run(t.Context()))
	})
	assert.Contains(t, got, "void entry(void)")

	listing = cmd.ParseOK[cmd.App[Cmd]](t, "--project", database, "listing", "--at", "0x1000", path)
	got = test.Stdout(t, func() {
		require.NoError(t, listing.Run(t.Context()))
	})
	assert.Contains(t, got, "signature void entry(void)")
}

func writeELF(t *testing.T, code []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "a.out")
	require.NoError(t, os.WriteFile(path, disasm.ELF64(elf.EM_X86_64, code), 0o644))
	return path
}

func writeELFAt(t *testing.T, code []byte, entry uint64) string {
	t.Helper()
	image := disasm.ELF64(elf.EM_X86_64, code)
	binary.LittleEndian.PutUint64(image[24:], entry)
	path := filepath.Join(t.TempDir(), "a.out")
	require.NoError(t, os.WriteFile(path, image, 0o644))
	return path
}

func writeELFTextAndRodata(t *testing.T, code, rodata []byte) string {
	t.Helper()
	const (
		headerSize  = 64
		sectionSize = 64
		sectionNum  = 4
	)
	names := []byte("\x00.text\x00.rodata\x00.shstrtab\x00")
	sectionOff := headerSize
	textOff := sectionOff + sectionNum*sectionSize
	rodataOff := textOff + len(code)
	namesOff := rodataOff + len(rodata)
	buf := make([]byte, namesOff+len(names))
	copy(buf, "\x7fELF")
	buf[4] = 2
	buf[5] = 1
	buf[6] = 1
	binary.LittleEndian.PutUint16(buf[16:], 1)
	binary.LittleEndian.PutUint16(buf[18:], uint16(elf.EM_X86_64))
	binary.LittleEndian.PutUint32(buf[20:], 1)
	binary.LittleEndian.PutUint64(buf[24:], 0x1000)
	binary.LittleEndian.PutUint64(buf[40:], uint64(sectionOff))
	binary.LittleEndian.PutUint16(buf[52:], headerSize)
	binary.LittleEndian.PutUint16(buf[58:], sectionSize)
	binary.LittleEndian.PutUint16(buf[60:], sectionNum)
	binary.LittleEndian.PutUint16(buf[62:], 3)
	putSection := func(index, name, kind int, flags, addr, off, size uint64) {
		section := buf[sectionOff+index*sectionSize:]
		binary.LittleEndian.PutUint32(section[0:], uint32(name))
		binary.LittleEndian.PutUint32(section[4:], uint32(kind))
		binary.LittleEndian.PutUint64(section[8:], flags)
		binary.LittleEndian.PutUint64(section[16:], addr)
		binary.LittleEndian.PutUint64(section[24:], off)
		binary.LittleEndian.PutUint64(section[32:], size)
		binary.LittleEndian.PutUint64(section[48:], 1)
	}
	putSection(1, 1, int(elf.SHT_PROGBITS), uint64(elf.SHF_ALLOC|elf.SHF_EXECINSTR), 0x1000, uint64(textOff), uint64(len(code)))
	putSection(2, 7, int(elf.SHT_PROGBITS), uint64(elf.SHF_ALLOC), 0x2000, uint64(rodataOff), uint64(len(rodata)))
	putSection(3, 15, int(elf.SHT_STRTAB), 0, 0, uint64(namesOff), uint64(len(names)))
	copy(buf[textOff:], code)
	copy(buf[rodataOff:], rodata)
	copy(buf[namesOff:], names)
	path := filepath.Join(t.TempDir(), "a.out")
	require.NoError(t, os.WriteFile(path, buf, 0o644))
	return path
}

func digestOf(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
