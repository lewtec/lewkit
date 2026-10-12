package main

import (
	"context"
	"debug/elf"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

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
	assert.Contains(t, text, "name")
	assert.Contains(t, text, "note")
	assert.Contains(t, text, "view")
	assert.Contains(t, text, "project")
}

func TestDecompViewUsage(t *testing.T) {
	test.RestoreSlog(t)
	view := cmd.ParseOK[cmd.App[root]](t, "--sentry-dsn", "", "decomp", "view", "--help")
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
	load := cmd.ParseOK[cmd.App[root]](t, "--sentry-dsn", "", "decomp", "--project", database, "functions", path)
	require.NoError(t, load.Run(t.Context()))

	rename := cmd.ParseOK[cmd.App[root]](t, "--sentry-dsn", "", "decomp", "--project", database, "name", "--at", "0x1000", "--name", "greet", path)
	got := test.Stdout(t, func() {
		require.NoError(t, rename.Run(t.Context()))
	})
	assert.Contains(t, got, "greet")

	note := cmd.ParseOK[cmd.App[root]](t, "--sentry-dsn", "", "decomp", "--project", database, "note", "--at", "0x1001", "--text", "padding", path)
	got = test.Stdout(t, func() {
		require.NoError(t, note.Run(t.Context()))
	})
	assert.Contains(t, got, "padding")
	require.NoError(t, os.Remove(path))

	listing := cmd.ParseOK[cmd.App[root]](t, "--sentry-dsn", "", "decomp", "--project", database, "listing", "--at", "0x1000", path)
	got = test.Stdout(t, func() {
		require.NoError(t, listing.Run(t.Context()))
	})
	assert.Contains(t, got, "greet")
	assert.Contains(t, got, "; note padding")
}

func TestDecompViewNeedsProject(t *testing.T) {
	test.RestoreSlog(t)
	view := cmd.ParseOK[cmd.App[root]](t, "--sentry-dsn", "", "decomp", "view")
	err := view.Run(t.Context())
	require.Error(t, err)
	assert.ErrorIs(t, err, errProjectRequired)
}

func TestDecompViewServesTheProject(t *testing.T) {
	test.RestoreSlog(t)
	path := writeELF(t, []byte{0x90, 0xc3})
	database := filepath.Join(t.TempDir(), "program.decomp")
	load := cmd.ParseOK[cmd.App[root]](t, "--sentry-dsn", "", "decomp", "--project", database, "functions", path)
	require.NoError(t, load.Run(t.Context()))

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := listener.Addr().String()
	require.NoError(t, listener.Close())

	view := cmd.ParseOK[cmd.App[root]](t, "--sentry-dsn", "", "decomp", "--project", database, "view", "--listen", address)
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
			t.Fatal(err)
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

func TestDecompProjectKeepsFunctions(t *testing.T) {
	test.RestoreSlog(t)
	path := writeELF(t, []byte{0x90, 0xc3})
	database := filepath.Join(t.TempDir(), "program.decomp")
	first := cmd.ParseOK[cmd.App[root]](t, "--sentry-dsn", "", "decomp", "--project", database, "functions", path)
	got := test.Stdout(t, func() {
		require.NoError(t, first.Run(t.Context()))
	})
	assert.Contains(t, got, "sub_1000")
	require.NoError(t, os.Remove(path))
	second := cmd.ParseOK[cmd.App[root]](t, "--sentry-dsn", "", "decomp", "--project", database, "functions", path)
	got = test.Stdout(t, func() {
		require.NoError(t, second.Run(t.Context()))
	})
	assert.Contains(t, got, "sub_1000")
	assert.Contains(t, got, "0x1000")
}

func TestDecompMatchUsesHarness(t *testing.T) {
	test.RestoreSlog(t)
	t.Setenv("XAI_API_KEY", "")
	path := writeELF(t, []byte{0xc3})
	bin := filepath.Join(t.TempDir(), "grok")
	candidate := `{"name":"sub_1000","signature":"void sub_1000(void)","source":"void sub_1000(void) {\n  return;\n}\n","blocks":[{"address":"0x1000","kind":"return"}]}`
	payload, err := json.Marshal(map[string]string{"text": candidate, "stopReason": "end_turn"})
	require.NoError(t, err)
	script := "#!/bin/sh\ncat <<'END'\n" + string(payload) + "\nEND\n"
	require.NoError(t, os.WriteFile(bin, []byte(script), 0o755))
	t.Setenv("PATH", filepath.Dir(bin)+string(os.PathListSeparator)+os.Getenv("PATH"))
	app := cmd.ParseOK[cmd.App[root]](t, "--sentry-dsn", "", "decomp", "match", "--at", "0x1000", "--rounds", "1", path)
	got := test.Stdout(t, func() {
		require.NoError(t, app.Run(t.Context()))
	})
	assert.Contains(t, got, "matched in 1")
	assert.Contains(t, got, "sub_1000")
}

func TestDecompMatchEndpointNeedsKey(t *testing.T) {
	test.RestoreSlog(t)
	t.Setenv("XAI_API_KEY", "")
	marker := filepath.Join(t.TempDir(), "called")
	bin := filepath.Join(filepath.Dir(marker), "grok")
	script := "#!/bin/sh\ntouch " + marker + "\nexit 1\n"
	require.NoError(t, os.WriteFile(bin, []byte(script), 0o755))
	t.Setenv("PATH", filepath.Dir(bin)+string(os.PathListSeparator)+os.Getenv("PATH"))
	path := writeELF(t, []byte{0xc3})
	app := cmd.ParseOK[cmd.App[root]](t, "--sentry-dsn", "", "decomp", "match", "--at", "0x1000", "--endpoint", "http://127.0.0.1:9", path)
	err := app.Run(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "api key")
	_, statErr := os.Stat(marker)
	assert.ErrorIs(t, statErr, os.ErrNotExist)
}

func writeELF(t *testing.T, code []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "a.out")
	require.NoError(t, os.WriteFile(path, disasm.ELF64(elf.EM_X86_64, code), 0o644))
	return path
}
