//go:build android && !cgo

package native

import (
	"errors"
	"runtime"
	"unsafe"
)

// dlopen comes from libdl.so. libc.so does not export it to the app, and
// libdl.so.2 is a glibc soname that Android does not ship.
const (
	Lazy   = 1
	Now    = 2
	Global = 0x100
	Local  = 0
)

func openPath(path string, flags int) (uintptr, error) {
	if !arm64Loader {
		return 0, errors.New("load " + path + ": android dynamic calls are arm64")
	}
	if flags == 0 {
		flags = Lazy
	}
	buf := append([]byte(path), 0)
	handle := libcDlopen(&buf[0], flags)
	runtime.KeepAlive(buf)
	if handle == 0 {
		return 0, errors.New("load " + path + ": " + dlError())
	}
	return handle, nil
}

// Symbol looks up name in lib.
func Symbol(lib uintptr, name string) (uintptr, error) {
	if !arm64Loader {
		return 0, errors.New("symbol " + name + ": android dynamic calls are arm64")
	}
	buf := append([]byte(name), 0)
	addr := libcDlsym(lib, &buf[0])
	runtime.KeepAlive(buf)
	if addr == 0 {
		return 0, errors.New("symbol " + name + ": " + dlError())
	}
	return addr, nil
}

// Func binds name in lib to fnptr (a pointer to a func variable).
func Func(lib uintptr, name string, fnptr any) {
	addr, err := Symbol(lib, name)
	if err != nil {
		panic(err)
	}
	Register(fnptr, addr)
}

func dlError() string {
	p := libcDlerror()
	if p == nil {
		return "dynamic linker error"
	}
	n := 0
	for *(*byte)(unsafe.Add(unsafe.Pointer(p), n)) != 0 {
		n++
		if n > 1<<16 {
			break
		}
	}
	return string(unsafe.Slice(p, n))
}
