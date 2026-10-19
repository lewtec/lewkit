package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lewtec/lewkit/x/build/internal/toolexec"
)

func TestMain(m *testing.M) {
	switch os.Getenv("LEWKIT_ANDROITOOL_HELPER") {
	case "fail":
		fmt.Fprintln(os.Stderr, "helper-said")
		os.Exit(1)
	case "silent":
		os.Exit(1)
	case "ok":
		fmt.Fprintln(os.Stdout, "helper-ok")
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestPeelExport(t *testing.T) {
	rest, export := peelExport([]string{
		"/tmp/x/_cgo_android_export.go",
		"/usr/lib/compile",
		"-V=full",
	})
	require.Equal(t, "/tmp/x/_cgo_android_export.go", export)
	require.Equal(t, []string{"/usr/lib/compile", "-V=full"}, rest)

	rest, export = peelExport([]string{"/usr/lib/compile", "-p", entryPkg})
	require.Empty(t, export)
	require.Len(t, rest, 3)
}

func TestVersionProbe(t *testing.T) {
	require.True(t, toolexec.VersionProbe([]string{"/usr/lib/compile", "-V=full"}))
	require.False(t, toolexec.VersionProbe([]string{"/usr/lib/compile", "-o", "a.o", "-V=full"}))
}

func TestRunToolForwardsChildText(t *testing.T) {
	t.Setenv("LEWKIT_ANDROITOOL_HELPER", "fail")
	var buf bytes.Buffer
	code := runTool([]string{os.Args[0]}, &buf)
	require.Equal(t, 1, code)
	require.Contains(t, buf.String(), "helper-said")
}

func TestRunToolNamesSilentExit(t *testing.T) {
	t.Setenv("LEWKIT_ANDROITOOL_HELPER", "silent")
	var buf bytes.Buffer
	code := runTool([]string{os.Args[0]}, &buf)
	require.Equal(t, 1, code)
	require.Contains(t, buf.String(), "exit status 1")
}

func TestRunToolPassesSuccessText(t *testing.T) {
	t.Setenv("LEWKIT_ANDROITOOL_HELPER", "ok")
	var buf bytes.Buffer
	code := runTool([]string{os.Args[0]}, &buf)
	require.Equal(t, 0, code)
	require.Contains(t, buf.String(), "helper-ok")
}

func TestExportStampTracksBytes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "_cgo_android_export.go")
	require.NoError(t, os.WriteFile(path, []byte("package entry\n"), 0o644))
	first, err := toolexec.HashStamp("lewkitexport=", exportCacheGen, path)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, []byte("package entry\n\n"), 0o644))
	second, err := toolexec.HashStamp("lewkitexport=", exportCacheGen, path)
	require.NoError(t, err)
	require.NotEmpty(t, first)
	require.NotEqual(t, first, second)
	require.Contains(t, first, "lewkitexport="+exportCacheGen+"-")
}
