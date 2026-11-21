//go:build !windows

package native

import "github.com/ebitengine/purego"

const (
	Lazy   = purego.RTLD_LAZY
	Now    = purego.RTLD_NOW
	Global = purego.RTLD_GLOBAL
	Local  = purego.RTLD_LOCAL
)

// Open loads path. flags 0 means Lazy.
func Open(path string, flags int) (uintptr, error) {
	if flags == 0 {
		flags = Lazy
	}
	return purego.Dlopen(path, flags)
}

// Symbol looks up name in lib.
func Symbol(lib uintptr, name string) (uintptr, error) {
	return purego.Dlsym(lib, name)
}
