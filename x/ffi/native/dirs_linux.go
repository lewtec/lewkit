//go:build linux

package native

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	_ "github.com/lewtec/lewkit/x/driver/exec/prelude"
)

// hostLibDirs are Linux library directories outside the fixed FHS list.
// NixOS keeps GTK in a store path named by zenity's RUNPATH, and Debian
// keeps it in a multiarch directory.
func hostLibDirs() []string {
	var dirs []string
	seen := map[string]bool{}
	add := func(dir string) {
		if dir == "" || seen[dir] {
			return
		}
		seen[dir] = true
		dirs = append(dirs, dir)
	}
	// The graphics driver and nix-ld come before a conda prefix. conda's
	// DT_RPATH still wins at dlopen time; HostGL maps the driver first.
	for _, dir := range glDirs(glDriverLib, envList("NIX_LD_LIBRARY_PATH")) {
		add(dir)
	}
	for _, profile := range envList("NIX_PROFILES") {
		add(filepath.Join(profile, "lib"))
	}
	user := os.Getenv("USER")
	if user == "" {
		user = os.Getenv("LOGNAME")
	}
	if user != "" {
		add("/etc/profiles/per-user/" + user + "/lib")
	}
	if home, err := os.UserHomeDir(); err == nil {
		add(filepath.Join(home, ".local", "state", "nix", "profiles", "profile", "lib"))
	}
	if prefix := os.Getenv("CONDA_PREFIX"); prefix != "" {
		add(filepath.Join(prefix, "lib"))
	}
	// mise exec conda:webkit2gtk4.1 puts <prefix>/bin or <prefix>/.mise-bins
	// on PATH and does not set CONDA_PREFIX. The libraries sit in <prefix>/lib.
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if dir == "" || !filepath.IsAbs(dir) {
			continue
		}
		switch filepath.Base(dir) {
		case "bin", ".mise-bins":
			add(filepath.Join(filepath.Dir(dir), "lib"))
		}
	}
	if dirs := toolRunpaths.Load(); dirs != nil {
		for _, dir := range *dirs {
			add(dir)
		}
	}
	switch runtime.GOARCH {
	case "amd64":
		add("/usr/lib/x86_64-linux-gnu")
		add("/lib/x86_64-linux-gnu")
	case "arm64":
		add("/usr/lib/aarch64-linux-gnu")
		add("/lib/aarch64-linux-gnu")
	case "386":
		add("/usr/lib/i386-linux-gnu")
		add("/lib/i386-linux-gnu")
	}
	return dirs
}

var (
	toolRunpathOnce sync.Once
	toolRunpaths    atomic.Pointer[[]string]
)

// Prepare resolves zenity, gtk4-launch, gtk-launch, and epiphany through the
// exec driver and keeps their library directories for SearchDirs.
// The first caller context is the one that runs the lookup.
func Prepare(ctx context.Context) error {
	if ctx == nil {
		return errNilContext
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	HostGL()
	DMABufSmoke()
	toolRunpathOnce.Do(func() {
		seen := map[string]bool{}
		var dirs []string
		for _, name := range []string{"zenity", "gtk4-launch", "gtk-launch", "epiphany"} {
			path := toolPath(ctx, name)
			if path == "" {
				continue
			}
			for _, dir := range runpathsOf(path) {
				if dir == "" || seen[dir] {
					continue
				}
				seen[dir] = true
				dirs = append(dirs, dir)
			}
		}
		toolRunpaths.Store(&dirs)
	})
	return nil
}

// toolPath asks the exec driver for name and keeps an absolute path.
// A relative result is not a toolkit install path.
func toolPath(ctx context.Context, name string) string {
	path, err := execdriver.Which(ctx, name)
	if err != nil || !filepath.IsAbs(path) {
		return ""
	}
	return path
}
