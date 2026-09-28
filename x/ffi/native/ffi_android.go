//go:build android && !cgo

package native

import "fmt"

const (
	Lazy   = 0
	Now    = 0
	Global = 0
	Local  = 0
)

// Open reports that this process has no pure-Go dynamic loader.
func Open(path string, flags int) (uintptr, error) {
	_ = flags
	return 0, fmt.Errorf("load %s: cgo is required on android", path)
}

// Symbol reports that this process has no pure-Go dynamic loader.
func Symbol(lib uintptr, name string) (uintptr, error) {
	_ = lib
	return 0, fmt.Errorf("symbol %s: cgo is required on android", name)
}

// Func reports that this process has no pure-Go dynamic loader.
func Func(lib uintptr, name string, fnptr any) {
	_ = lib
	_ = name
	_ = fnptr
}

// Register reports that this process has no pure-Go dynamic loader.
func Register(fnptr any, addr uintptr) {
	_ = fnptr
	_ = addr
}
