// Package native loads shared libraries and binds functions without cgo.
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

// Func binds name in lib to fnptr (a pointer to a func variable).
func Func(lib uintptr, name string, fnptr any) {
	purego.RegisterLibFunc(fnptr, lib, name)
}

// Symbol looks up name in lib.
func Symbol(lib uintptr, name string) (uintptr, error) {
	return purego.Dlsym(lib, name)
}

// Register binds fnptr to a function address (objc_msgSend and the like).
func Register(fnptr any, addr uintptr) {
	purego.RegisterFunc(fnptr, addr)
}
