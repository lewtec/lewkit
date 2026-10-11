package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

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
	require.True(t, versionProbe([]string{"/usr/lib/compile", "-V=full"}))
	require.False(t, versionProbe([]string{"/usr/lib/compile", "-o", "a.o", "-V=full"}))
}

func TestExportStampTracksBytes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "_cgo_android_export.go")
	require.NoError(t, os.WriteFile(path, []byte("package entry\n"), 0o644))
	first, err := exportStamp(path)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, []byte("package entry\n\n"), 0o644))
	second, err := exportStamp(path)
	require.NoError(t, err)
	require.NotEmpty(t, first)
	require.NotEqual(t, first, second)
}
