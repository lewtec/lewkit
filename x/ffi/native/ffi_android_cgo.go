//go:build android && cgo

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

// Func binds name in lib to fnptr (a pointer to a func variable).
func Func(lib uintptr, name string, fnptr any) {
	purego.RegisterLibFunc(fnptr, lib, name)
}

// Register binds fnptr to a function address.
func Register(fnptr any, addr uintptr) {
	purego.RegisterFunc(fnptr, addr)
}
