//go:build linux

package native

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// glDriverLib is the NixOS graphics driver. hardware.graphics.enable
// points it at the Mesa or NVIDIA userspace that matches the kernel.
const glDriverLib = "/run/opengl-driver/lib"

// glSonames are the GL entry points conda ships beside WebKit.
// libGLdispatch comes before libEGL because libEGL needs it.
var glSonames = []string{
	"libGLdispatch.so.0",
	"libGLX.so.0",
	"libEGL.so.1",
	"libGL.so.1",
	"libOpenGL.so.0",
	"libGLESv2.so.2",
	"libgbm.so.1",
}

var (
	hostGLOnce sync.Once
	hostGLOn   bool
)

// HostGL binds the NixOS OpenGL driver when one is visible.
// conda-forge libraries carry DT_RPATH $ORIGIN, and nix-ld searches
// NIX_LD_LIBRARY_PATH after that rpath, so conda's libEGL is the one
// that answers eglInitialize. Mapping the driver SONAME first keeps
// that call on the host driver. LD_PRELOAD carries the same files into
// WebKit's web process. A second call returns the first result.
func HostGL() bool {
	hostGLOnce.Do(bindHostGL)
	return hostGLOn
}

func bindHostGL() {
	libs := existingGLLibs(glDirs(glDriverLib, envList("NIX_LD_LIBRARY_PATH")))
	if !glHasEGL(libs) {
		return
	}
	hostGLOn = true
	for _, path := range libs {
		if resident(filepath.Base(path)) {
			continue
		}
		_, _ = Open(path, Global|Now)
	}
	if merged := mergePreload(os.Getenv("LD_PRELOAD"), libs); merged != os.Getenv("LD_PRELOAD") {
		_ = os.Setenv("LD_PRELOAD", merged)
	}
	setGLEnv(glDriverLib)
	// Probe after the host driver is mapped, so the smoke sees that EGL
	// and a failure is recorded before WebKit caches its renderer.
	DMABufSmoke()
}

// sharedMemoryRendererEnv is the fallback after the DMA-BUF smoke fails.
// A WebKit that still composites then ships shared memory instead of
// calling eglCreateImage on every frame. A value already set is left alone.
// A passing smoke does not call this.
func sharedMemoryRendererEnv(disable, forceSHM string) (key, value string, ok bool) {
	if disable != "" || forceSHM != "" {
		return "", "", false
	}
	return "WEBKIT_DMABUF_RENDERER_FORCE_SHM", "1", true
}

func glDirs(driver string, nixLD []string) []string {
	var dirs []string
	seen := map[string]bool{}
	add := func(dir string) {
		if dir == "" || seen[dir] {
			return
		}
		seen[dir] = true
		dirs = append(dirs, dir)
	}
	add(driver)
	for _, dir := range nixLD {
		add(dir)
	}
	return dirs
}

func existingGLLibs(dirs []string) []string {
	var out []string
	for _, name := range glSonames {
		for _, dir := range dirs {
			path := filepath.Join(dir, name)
			info, err := os.Stat(path)
			if err != nil || info.IsDir() {
				continue
			}
			out = append(out, path)
			break
		}
	}
	return out
}

func glHasEGL(libs []string) bool {
	for _, path := range libs {
		if filepath.Base(path) == "libEGL.so.1" {
			return true
		}
	}
	return false
}

func mergePreload(existing string, paths []string) string {
	have := map[string]bool{}
	for _, part := range strings.FieldsFunc(existing, isPreloadSep) {
		have[part] = true
	}
	var add []string
	for _, path := range paths {
		if path == "" || have[path] {
			continue
		}
		have[path] = true
		add = append(add, path)
	}
	if len(add) == 0 {
		return existing
	}
	joined := strings.Join(add, ":")
	if existing == "" {
		return joined
	}
	return joined + ":" + existing
}

func isPreloadSep(r rune) bool {
	return r == ':' || r == ' '
}

func setGLEnv(driver string) {
	trySetDir("LIBGL_DRIVERS_PATH", filepath.Join(driver, "dri"))
	trySetDir("GBM_BACKENDS_PATH", filepath.Join(driver, "gbm"))
	trySetDir("__EGL_VENDOR_LIBRARY_DIRS", filepath.Join(filepath.Dir(driver), "share", "glvnd", "egl_vendor.d"))
}

func trySetDir(key, path string) {
	if os.Getenv(key) != "" || path == "" {
		return
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return
	}
	_ = os.Setenv(key, path)
}
