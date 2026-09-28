//go:build !windows && !android

package native

import "github.com/ebitengine/purego"

const (
	Lazy   = purego.RTLD_LAZY
	Now    = purego.RTLD_NOW
	Global = purego.RTLD_GLOBAL
	Local  = purego.RTLD_LOCAL
)

func openPath(path string, flags int) (uintptr, error) {
	return purego.Dlopen(path, flags)
}

// Symbol looks up name in lib.
func Symbol(lib uintptr, name string) (uintptr, error) {
	return purego.Dlsym(lib, name)
}
