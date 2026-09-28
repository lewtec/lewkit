package native

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// SearchDirs is the directory chain for a bare soname.
// extra directories come first, then LEWKIT_LIB, the dynamic linker path,
// the Nix profiles, and the usual system library directories.
// Empty entries and duplicates are dropped.
func SearchDirs(extra ...string) []string {
	var dirs []string
	seen := map[string]bool{}
	add := func(dir string) {
		if dir == "" || seen[dir] {
			return
		}
		seen[dir] = true
		dirs = append(dirs, dir)
	}
	for _, dir := range extra {
		add(dir)
	}
	for _, dir := range envList("LEWKIT_LIB") {
		add(dir)
	}
	if runtime.GOOS == "windows" {
		return dirs
	}
	for _, dir := range envList("LD_LIBRARY_PATH") {
		add(dir)
	}
	for _, dir := range envList("DYLD_LIBRARY_PATH") {
		add(dir)
	}
	add("/run/current-system/sw/lib")
	if home, err := os.UserHomeDir(); err == nil {
		add(filepath.Join(home, ".nix-profile", "lib"))
	}
	add("/usr/lib")
	add("/usr/lib64")
	add("/usr/local/lib")
	if runtime.GOOS == "darwin" {
		add("/opt/homebrew/lib")
	}
	return dirs
}

func envList(key string) []string {
	return strings.Split(os.Getenv(key), string(os.PathListSeparator))
}

func expandName(name string, dirs []string) []string {
	if name == "" {
		return nil
	}
	if filepath.IsAbs(name) || strings.ContainsAny(name, `/\`) {
		return []string{name}
	}
	out := make([]string, 0, len(dirs)+1)
	out = append(out, name)
	for _, dir := range dirs {
		out = append(out, filepath.Join(dir, name))
	}
	return out
}

type chainKey struct {
	flags int
	names string
}

var chains sync.Map // chainKey -> func() (uintptr, error)

func tryChain(flags int, names []string) (uintptr, error) {
	dirs := SearchDirs()
	var last error
	for _, name := range names {
		for _, path := range expandName(name, dirs) {
			lib, err := Open(path, flags)
			if err == nil {
				return lib, nil
			}
			last = err
		}
	}
	if last == nil {
		return 0, errNoLibrary
	}
	return 0, last
}

// OpenChain loads the first name that opens and remembers that result.
//
// A bare name is tried as itself, then inside each SearchDirs entry.
// A path is tried only as given. The chain for these flags and names runs
// once: later calls return the first handle or the first error.
func OpenChain(flags int, names ...string) (uintptr, error) {
	if len(names) == 0 {
		return 0, errNoLibrary
	}
	if flags == 0 {
		flags = Lazy
	}
	key := chainKey{flags: flags, names: strings.Join(names, "\x00")}
	if value, ok := chains.Load(key); ok {
		return value.(func() (uintptr, error))()
	}
	run := Singleton(func() (uintptr, error) {
		return tryChain(flags, names)
	})
	actual, existed := chains.LoadOrStore(key, run)
	if existed {
		return actual.(func() (uintptr, error))()
	}
	return run()
}
