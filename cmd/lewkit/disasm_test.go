package main

import (
	"debug/elf"
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

func TestDisasmBadArchitecture(t *testing.T) {
	err := cmd.ParseErr[cmd.App[root]](t, "--sentry-dsn", "", "disasm", "--architecture", "itanium", "hex", "90")
	assert.ErrorIs(t, err, cmd.ErrInvalidArgument)
	assert.Contains(t, err.Error(), "want one of")
}

func TestDisasmUsage(t *testing.T) {
	text, err := cmd.Usage[disasmCmd]("lewkit disasm")
	require.NoError(t, err)
	assert.Contains(t, text, "hex")
	assert.Contains(t, text, "raw")
	assert.Contains(t, text, "file")
	assert.Contains(t, text, "--architecture")
}

func TestDisasmHex(t *testing.T) {
	test.RestoreSlog(t)
	app := cmd.ParseOK[cmd.App[root]](t, "--sentry-dsn", "", "disasm", "hex", "90c3")
	got := test.Stdout(t, func() {
		require.NoError(t, app.Run(t.Context()))
	})
	assert.Contains(t, got, "nop")
	assert.Contains(t, got, "ret")
}

func TestDisasmHexCount(t *testing.T) {
	test.RestoreSlog(t)
	app := cmd.ParseOK[cmd.App[root]](t, "--sentry-dsn", "", "disasm", "--count", "1", "hex", "90c3")
	got := test.Stdout(t, func() {
		require.NoError(t, app.Run(t.Context()))
	})
	assert.Contains(t, got, "nop")
	assert.NotContains(t, got, "ret")
}

func TestDisasmRaw(t *testing.T) {
	test.RestoreSlog(t)
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
	test.RestoreSlog(t)
	raw := disasm.ELF64(elf.EM_X86_64, []byte{0x90, 0xc3})
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
