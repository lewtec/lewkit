package main

import (
	"debug/elf"
	"os"
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/disasm"
	"github.com/lewtec/lewkit/x/release"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecompUsage(t *testing.T) {
	text, err := cmd.Usage[decompCmd](release.Name() + " decomp")
	require.NoError(t, err)
	assert.Contains(t, text, "summary")
	assert.Contains(t, text, "functions")
	assert.Contains(t, text, "listing")
	assert.Contains(t, text, "match")
}

func TestDecompFunctionsAndListing(t *testing.T) {
	test.RestoreSlog(t)
	path := writeELF(t, []byte{0x90, 0xc3})
	functions := cmd.ParseOK[cmd.App[root]](t, "--sentry-dsn", "", "decomp", "functions", path)
	got := test.Stdout(t, func() {
		require.NoError(t, functions.Run(t.Context()))
	})
	assert.Contains(t, got, "0x1000")
	assert.Contains(t, got, "sub_1000")

	listing := cmd.ParseOK[cmd.App[root]](t, "--sentry-dsn", "", "decomp", "listing", "--at", "0x1000", path)
	got = test.Stdout(t, func() {
		require.NoError(t, listing.Run(t.Context()))
	})
	assert.Contains(t, got, "nop")
	assert.Contains(t, got, "ret")
}

func TestDecompMatchNeedsKey(t *testing.T) {
	test.RestoreSlog(t)
	t.Setenv("XAI_API_KEY", "")
	path := writeELF(t, []byte{0xc3})
	app := cmd.ParseOK[cmd.App[root]](t, "--sentry-dsn", "", "decomp", "match", "--at", "0x1000", path)
	err := app.Run(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "api key")
}

func writeELF(t *testing.T, code []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "a.out")
	require.NoError(t, os.WriteFile(path, disasm.ELF64(elf.EM_X86_64, code), 0o644))
	return path
}
