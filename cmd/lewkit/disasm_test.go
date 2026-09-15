package main

import (
	"debug/elf"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/disasm"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDisasmUsage(t *testing.T) {
	text, err := cmd.Usage[disasmCmd]("lewkit disasm")
	require.NoError(t, err)
	assert.Contains(t, text, "hex")
	assert.Contains(t, text, "raw")
	assert.Contains(t, text, "file")
	assert.Contains(t, text, "--architecture")
}

func TestDisasmHex(t *testing.T) {
	app := cmd.ParseOK[cmd.App[root]](t, "--sentry-dsn", "", "disasm", "hex", "90c3")
	got := test.Stdout(t, func() {
		require.NoError(t, app.Run(t.Context()))
	})
	assert.Contains(t, got, "nop")
	assert.Contains(t, got, "ret")
}

func TestDisasmHexCount(t *testing.T) {
	app := cmd.ParseOK[cmd.App[root]](t, "--sentry-dsn", "", "disasm", "--count", "1", "hex", "90c3")
	got := test.Stdout(t, func() {
		require.NoError(t, app.Run(t.Context()))
	})
	assert.Contains(t, got, "nop")
	assert.NotContains(t, got, "ret")
}

func TestDisasmRaw(t *testing.T) {
	path := filepath.Join(t.TempDir(), "code.bin")
	require.NoError(t, os.WriteFile(path, []byte{0x90, 0xc3}, 0o644))
	app := cmd.ParseOK[cmd.App[root]](t, "--sentry-dsn", "", "disasm", "raw", path)
	got := test.Stdout(t, func() {
		require.NoError(t, app.Run(t.Context()))
	})
	assert.Equal(t, 2, strings.Count(got, "\n"))
	assert.Contains(t, got, "nop")
	assert.Contains(t, got, "ret")
}

func TestDisasmFile(t *testing.T) {
	raw := buildELF64([]byte{0x90, 0xc3})
	_, err := disasm.ReadText(bytesReader(raw), int64(len(raw)), "")
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "a.out")
	require.NoError(t, os.WriteFile(path, raw, 0o644))
	app := cmd.ParseOK[cmd.App[root]](t, "--sentry-dsn", "", "disasm", "file", path)
	got := test.Stdout(t, func() {
		require.NoError(t, app.Run(t.Context()))
	})
	assert.Contains(t, got, "0x00001000")
	assert.Contains(t, got, "nop")
	assert.Contains(t, got, "ret")
}

func TestRootUsageListsDisasm(t *testing.T) {
	text, err := cmd.Usage[cmd.App[root]]("lewkit")
	require.NoError(t, err)
	assert.Contains(t, text, "disasm")
}

func buildELF64(code []byte) []byte {
	const (
		ehsize  = 64
		shsize  = 64
		shnum   = 3
		shoff   = ehsize
		textoff = ehsize + shnum*shsize
	)
	shstr := []byte("\x00.text\x00.shstrtab\x00")
	buf := make([]byte, textoff+len(code)+len(shstr))
	copy(buf[0:], "\x7fELF")
	buf[4] = 2
	buf[5] = 1
	buf[6] = 1
	binary.LittleEndian.PutUint16(buf[16:], 1)
	binary.LittleEndian.PutUint16(buf[18:], uint16(elf.EM_X86_64))
	binary.LittleEndian.PutUint32(buf[20:], 1)
	binary.LittleEndian.PutUint64(buf[40:], shoff)
	binary.LittleEndian.PutUint16(buf[52:], ehsize)
	binary.LittleEndian.PutUint16(buf[58:], shsize)
	binary.LittleEndian.PutUint16(buf[60:], shnum)
	binary.LittleEndian.PutUint16(buf[62:], 2)

	text := buf[shoff+shsize:]
	binary.LittleEndian.PutUint32(text[0:], 1)
	binary.LittleEndian.PutUint32(text[4:], 1)
	binary.LittleEndian.PutUint64(text[8:], uint64(elf.SHF_ALLOC|elf.SHF_EXECINSTR))
	binary.LittleEndian.PutUint64(text[16:], 0x1000)
	binary.LittleEndian.PutUint64(text[24:], uint64(textoff))
	binary.LittleEndian.PutUint64(text[32:], uint64(len(code)))
	binary.LittleEndian.PutUint64(text[48:], 1)

	strtab := buf[shoff+2*shsize:]
	binary.LittleEndian.PutUint32(strtab[0:], 7)
	binary.LittleEndian.PutUint32(strtab[4:], 3)
	binary.LittleEndian.PutUint64(strtab[24:], uint64(textoff+len(code)))
	binary.LittleEndian.PutUint64(strtab[32:], uint64(len(shstr)))
	binary.LittleEndian.PutUint64(strtab[48:], 1)

	copy(buf[textoff:], code)
	copy(buf[textoff+len(code):], shstr)
	return buf
}
