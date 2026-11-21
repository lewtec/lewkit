// Package native loads shared libraries and binds functions without cgo.
package native

import "github.com/ebitengine/purego"

// Func binds name in lib to fnptr (a pointer to a func variable).
func Func(lib uintptr, name string, fnptr any) {
	purego.RegisterLibFunc(fnptr, lib, name)
}

// Register binds fnptr to a function address (objc_msgSend and the like).
func Register(fnptr any, addr uintptr) {
	purego.RegisterFunc(fnptr, addr)
}
