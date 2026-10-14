package tool

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCopyDiskImageUnwrapsVolumeDirectory(t *testing.T) {
	t.Parallel()

	mountpoint := t.TempDir()
	wrapper := filepath.Join(mountpoint, "Obsidian 1.14.4-universal")
	executable := filepath.Join(wrapper, "Obsidian.app", "Contents", "MacOS", "Obsidian")
	require.NoError(t, os.MkdirAll(filepath.Dir(executable), 0o755))
	require.NoError(t, os.WriteFile(executable, []byte("macho"), 0o755))
	require.NoError(t, os.Symlink("/Applications", filepath.Join(wrapper, "Applications")))

	destination := t.TempDir()
	require.NoError(t, copyDiskImage(t.Context(), mountpoint, destination))

	want := filepath.Join(destination, "Obsidian.app", "Contents", "MacOS", "Obsidian")
	require.Equal(t, want, FindBinary(destination, "obsidian"))
	body, err := os.ReadFile(want)
	require.NoError(t, err)
	require.Equal(t, "macho", string(body))
	target, err := os.Readlink(filepath.Join(destination, "Applications"))
	require.NoError(t, err)
	require.Equal(t, "/Applications", target)
	_, err = os.Stat(filepath.Join(destination, "Obsidian 1.14.4-universal"))
	require.Error(t, err)
}

func TestCopyDiskImageKeepsAppAtVolumeRoot(t *testing.T) {
	t.Parallel()

	mountpoint := t.TempDir()
	executable := filepath.Join(mountpoint, "Blender.app", "Contents", "MacOS", "Blender")
	require.NoError(t, os.MkdirAll(filepath.Dir(executable), 0o755))
	require.NoError(t, os.WriteFile(executable, []byte("macho"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(mountpoint, ".DS_Store"), []byte("store"), 0o644))

	destination := t.TempDir()
	require.NoError(t, copyDiskImage(t.Context(), mountpoint, destination))

	want := filepath.Join(destination, "Blender.app", "Contents", "MacOS", "Blender")
	require.Equal(t, want, FindBinary(destination, "blender"))
	_, err := os.Stat(filepath.Join(destination, "Contents"))
	require.Error(t, err)
}

func TestCopyDiskImageDoesNotUnwrapLoneApp(t *testing.T) {
	t.Parallel()

	mountpoint := t.TempDir()
	executable := filepath.Join(mountpoint, "Blender.app", "Contents", "MacOS", "Blender")
	require.NoError(t, os.MkdirAll(filepath.Dir(executable), 0o755))
	require.NoError(t, os.WriteFile(executable, []byte("macho"), 0o755))

	destination := t.TempDir()
	require.NoError(t, copyDiskImage(t.Context(), mountpoint, destination))
	require.Equal(t, filepath.Join(destination, "Blender.app", "Contents", "MacOS", "Blender"), FindBinary(destination, "blender"))
}

func TestExtractDiskImageDoesNotInstallAsBinary(t *testing.T) {
	t.Parallel()

	source := filepath.Join(t.TempDir(), "Obsidian-1.14.4.dmg")
	require.NoError(t, os.WriteFile(source, []byte("not-an-archive-dmg"), 0o644))
	destination := t.TempDir()

	err := Extract(t.Context(), source, destination)
	require.ErrorIs(t, err, ErrDiskImageMount)
	require.Empty(t, FindBinary(destination, "obsidian"))
	_, statErr := os.Stat(filepath.Join(destination, "Obsidian"))
	require.Error(t, statErr)
}
