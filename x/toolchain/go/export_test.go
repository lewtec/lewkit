package gotool

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCgoExportFilesUsesPackageCopy(t *testing.T) {
	checkout := t.TempDir()
	module := t.TempDir()
	name := "_cgo_android_export.go"
	require.NoError(t, os.WriteFile(filepath.Join(checkout, name), []byte("package android\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(module, name), []byte("package android\n"), 0o644))
	tool := filepath.Join(checkout, "androidtoolexec")
	require.NoError(t, os.WriteFile(tool, []byte("#!/bin/sh\n"), 0o755))
	export := filepath.Join(checkout, name)

	got, err := cgoExportFiles(tool+" "+export, module)
	require.NoError(t, err)
	require.Equal(t, []string{filepath.Join(module, name)}, got)

	got, err = cgoExportFiles(tool+" "+export, checkout)
	require.NoError(t, err)
	require.Equal(t, []string{export}, got)

	got, err = cgoExportFiles(tool+" "+export, t.TempDir())
	require.NoError(t, err)
	require.Empty(t, got)

	_, err = cgoExportFiles("'unterminated", module)
	require.ErrorIs(t, err, ErrToolexec)
}
