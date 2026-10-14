//go:build linux && !android

package native

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGLDirsPreferTheNixOSDriver(t *testing.T) {
	dirs := glDirs("/run/opengl-driver/lib", []string{"/nix/ld/lib", "/run/opengl-driver/lib", ""})
	if len(dirs) != 2 || dirs[0] != "/run/opengl-driver/lib" || dirs[1] != "/nix/ld/lib" {
		require.FailNowf(t, "", "glDirs = %q", dirs)
	}
}

func TestExistingGLLibsTakeTheFirstDirectory(t *testing.T) {
	root := t.TempDir()
	driver := filepath.Join(root, "driver")
	nixld := filepath.Join(root, "nix-ld")
	require.NoError(t, os.MkdirAll(driver, 0o755))
	require.NoError(t, os.MkdirAll(nixld, 0o755))
	egl := filepath.Join(driver, "libEGL.so.1")
	require.NoError(t, os.WriteFile(egl, []byte("egl"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(nixld, "libEGL.so.1"), []byte("other"), 0o644))
	gbm := filepath.Join(nixld, "libgbm.so.1")
	require.NoError(t, os.WriteFile(gbm, []byte("gbm"), 0o644))
	libs := existingGLLibs(glDirs(driver, []string{nixld}))
	if len(libs) != 2 || libs[0] != egl || libs[1] != gbm {
		require.FailNowf(t, "", "libs = %q", libs)
	}
	if !glHasEGL(libs) {
		require.FailNow(t, fmt.Sprint("libEGL missing"))
	}
}

func TestMergePreloadKeepsExistingEntries(t *testing.T) {
	got := mergePreload("/already.so: /space.so", []string{"/run/opengl-driver/lib/libEGL.so.1", "/already.so"})
	want := "/run/opengl-driver/lib/libEGL.so.1:/already.so: /space.so"
	require.Equal(t, want, got)
	if mergePreload("", nil) != "" {
		require.FailNow(t, fmt.Sprint("empty preload changed"))
	}
}

func TestSharedMemoryRendererLeavesAnExplicitChoice(t *testing.T) {
	key, value, ok := sharedMemoryRendererEnv("", "")
	if !ok || key != "WEBKIT_DMABUF_RENDERER_FORCE_SHM" || value != "1" {
		require.FailNowf(t, "", "shared memory env = %s=%s ok=%v", key, value, ok)
	}
	if _, _, ok := sharedMemoryRendererEnv("1", ""); ok {
		require.FailNow(t, fmt.Sprint("WEBKIT_DISABLE_DMABUF_RENDERER was replaced"))
	}
	if _, _, ok := sharedMemoryRendererEnv("", "0"); ok {
		require.FailNow(t, fmt.Sprint("WEBKIT_DMABUF_RENDERER_FORCE_SHM was replaced"))
	}
}

func TestTrySetDirLeavesAValueAlone(t *testing.T) {
	t.Setenv("LEWKIT_GL_TEST", "keep")
	trySetDir("LEWKIT_GL_TEST", t.TempDir())
	if os.Getenv("LEWKIT_GL_TEST") != "keep" {
		require.FailNowf(t, "", "env = %q", os.Getenv("LEWKIT_GL_TEST"))
	}
	t.Setenv("LEWKIT_GL_TEST_EMPTY", "")
	dir := t.TempDir()
	trySetDir("LEWKIT_GL_TEST_EMPTY", dir)
	if os.Getenv("LEWKIT_GL_TEST_EMPTY") != dir {
		require.FailNowf(t, "", "env = %q", os.Getenv("LEWKIT_GL_TEST_EMPTY"))
	}
	trySetDir("LEWKIT_GL_TEST_MISSING", filepath.Join(t.TempDir(), "missing"))
	if os.Getenv("LEWKIT_GL_TEST_MISSING") != "" {
		require.FailNowf(t, "", "missing dir was exported: %q", os.Getenv("LEWKIT_GL_TEST_MISSING"))
	}
}
