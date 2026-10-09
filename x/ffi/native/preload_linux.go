//go:build linux

package native

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ebitengine/purego"
)

// preloadSiblingDeps opens DT_NEEDED libraries from path's directory and its
// DT_RUNPATH before dlopen. LD_LIBRARY_PATH is searched before DT_RUNPATH, so
// an older libgstreamer would satisfy conda-forge libgsttranscoder and miss
// gst_state_get_name. A soname that is already mapped stays loaded.
func preloadSiblingDeps(path string, flags int) {
	if !filepath.IsAbs(path) {
		return
	}
	// Dlopen directly. Open calls openPath, and openPath calls this function.
	for _, dep := range preloadOrder(path) {
		if resident(filepath.Base(dep)) {
			continue
		}
		_, _ = purego.Dlopen(dep, flags)
	}
}

// rtldNoLoad asks the dynamic linker whether a soname is already mapped.
const rtldNoLoad = 0x4

func resident(soname string) bool {
	handle, err := purego.Dlopen(soname, purego.RTLD_LAZY|rtldNoLoad)
	return err == nil && handle != 0
}

// Loaded reports whether soname is already mapped in this process.
func Loaded(soname string) bool {
	if soname == "" {
		return false
	}
	return resident(soname)
}

var (
	preloadMu   sync.Mutex
	preloadSeen = map[string]bool{}
)

// preloadOrder is a post-order of absolute dependencies. path itself is not
// included. A library already being loaded is skipped so cycles return.
func preloadOrder(path string) []string {
	preloadMu.Lock()
	defer preloadMu.Unlock()
	if preloadSeen[path] {
		return nil
	}
	seen := map[string]bool{path: true}
	var order []string
	var walk func(string)
	walk = func(name string) {
		if seen[name] {
			return
		}
		seen[name] = true
		for _, dep := range libraryDeps(name) {
			walk(dep)
		}
		order = append(order, name)
	}
	for _, dep := range libraryDeps(path) {
		walk(dep)
	}
	preloadSeen[path] = true
	for _, dep := range order {
		preloadSeen[dep] = true
	}
	return order
}

// libraryDeps is the DT_NEEDED files that live beside path or on its runpath.
// libc stays with the process.
func libraryDeps(path string) []string {
	names := elfNeeded(path)
	if len(names) == 0 {
		return nil
	}
	dirs := []string{filepath.Dir(path)}
	for _, dir := range elfRunpath(path) {
		if dir != "" && dir != dirs[0] {
			dirs = append(dirs, dir)
		}
	}
	var out []string
	seen := map[string]bool{}
	for _, name := range names {
		if skipPreload(name) {
			continue
		}
		for _, dir := range dirs {
			candidate := filepath.Join(dir, name)
			if seen[candidate] {
				break
			}
			info, err := os.Stat(candidate)
			if err != nil || info.IsDir() {
				continue
			}
			seen[candidate] = true
			out = append(out, candidate)
			break
		}
	}
	return out
}

func skipPreload(name string) bool {
	base := filepath.Base(name)
	switch base {
	case "libc.so.6", "libm.so.6", "libdl.so.2", "libpthread.so.0",
		"librt.so.1", "libresolv.so.2", "libutil.so.1", "libanl.so.1":
		return true
	}
	return strings.HasPrefix(base, "ld-linux") || strings.HasPrefix(base, "libnss_")
}
