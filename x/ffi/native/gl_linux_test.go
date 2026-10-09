//go:build linux

package native

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGLDirsPreferTheNixOSDriver(t *testing.T) {
	dirs := glDirs("/run/opengl-driver/lib", []string{"/nix/ld/lib", "/run/opengl-driver/lib", ""})
	if len(dirs) != 2 || dirs[0] != "/run/opengl-driver/lib" || dirs[1] != "/nix/ld/lib" {
		t.Fatalf("glDirs = %q", dirs)
	}
}

func TestExistingGLLibsTakeTheFirstDirectory(t *testing.T) {
	root := t.TempDir()
	driver := filepath.Join(root, "driver")
	nixld := filepath.Join(root, "nix-ld")
	if err := os.MkdirAll(driver, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(nixld, 0o755); err != nil {
		t.Fatal(err)
	}
	egl := filepath.Join(driver, "libEGL.so.1")
	if err := os.WriteFile(egl, []byte("egl"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nixld, "libEGL.so.1"), []byte("other"), 0o644); err != nil {
		t.Fatal(err)
	}
	gbm := filepath.Join(nixld, "libgbm.so.1")
	if err := os.WriteFile(gbm, []byte("gbm"), 0o644); err != nil {
		t.Fatal(err)
	}
	libs := existingGLLibs(glDirs(driver, []string{nixld}))
	if len(libs) != 2 || libs[0] != egl || libs[1] != gbm {
		t.Fatalf("libs = %q", libs)
	}
	if !glHasEGL(libs) {
		t.Fatal("libEGL missing")
	}
}

func TestMergePreloadKeepsExistingEntries(t *testing.T) {
	got := mergePreload("/already.so: /space.so", []string{"/run/opengl-driver/lib/libEGL.so.1", "/already.so"})
	want := "/run/opengl-driver/lib/libEGL.so.1:/already.so: /space.so"
	if got != want {
		t.Fatalf("mergePreload = %q", got)
	}
	if mergePreload("", nil) != "" {
		t.Fatal("empty preload changed")
	}
}

func TestSharedMemoryRendererLeavesAnExplicitChoice(t *testing.T) {
	key, value, ok := sharedMemoryRendererEnv("", "")
	if !ok || key != "WEBKIT_DMABUF_RENDERER_FORCE_SHM" || value != "1" {
		t.Fatalf("shared memory env = %s=%s ok=%v", key, value, ok)
	}
	if _, _, ok := sharedMemoryRendererEnv("1", ""); ok {
		t.Fatal("WEBKIT_DISABLE_DMABUF_RENDERER was replaced")
	}
	if _, _, ok := sharedMemoryRendererEnv("", "0"); ok {
		t.Fatal("WEBKIT_DMABUF_RENDERER_FORCE_SHM was replaced")
	}
}

func TestTrySetDirLeavesAValueAlone(t *testing.T) {
	t.Setenv("LEWKIT_GL_TEST", "keep")
	trySetDir("LEWKIT_GL_TEST", t.TempDir())
	if os.Getenv("LEWKIT_GL_TEST") != "keep" {
		t.Fatalf("env = %q", os.Getenv("LEWKIT_GL_TEST"))
	}
	t.Setenv("LEWKIT_GL_TEST_EMPTY", "")
	dir := t.TempDir()
	trySetDir("LEWKIT_GL_TEST_EMPTY", dir)
	if os.Getenv("LEWKIT_GL_TEST_EMPTY") != dir {
		t.Fatalf("env = %q", os.Getenv("LEWKIT_GL_TEST_EMPTY"))
	}
	trySetDir("LEWKIT_GL_TEST_MISSING", filepath.Join(t.TempDir(), "missing"))
	if os.Getenv("LEWKIT_GL_TEST_MISSING") != "" {
		t.Fatalf("missing dir was exported: %q", os.Getenv("LEWKIT_GL_TEST_MISSING"))
	}
}
