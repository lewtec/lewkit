//go:build android

package android

import (
	"fmt"

	"github.com/lewtec/lewkit/x/ffi/native"
)

// JavaVMs returns how many Java VMs this process has created.
func JavaVMs() (int, error) {
	lib, err := open("libnativehelper.so", "libart.so")
	if err != nil {
		return 0, err
	}
	sym, err := native.Symbol(lib, "JNI_GetCreatedJavaVMs")
	if err != nil {
		return 0, err
	}
	var fn func(vmBuf *uintptr, bufLen int32, nVMs *int32) int32
	native.Register(&fn, sym)
	var vm uintptr
	var n int32
	if rc := fn(&vm, 1, &n); rc != 0 {
		return 0, fmt.Errorf("JNI_GetCreatedJavaVMs: %d", rc)
	}
	return int(n), nil
}

// OnLooper reports whether this thread runs the Android main looper.
func OnLooper() (bool, error) {
	lib, err := native.Open("libandroid.so", native.Now)
	if err != nil {
		return false, err
	}
	sym, err := native.Symbol(lib, "ALooper_forThread")
	if err != nil {
		return false, err
	}
	var fn func() uintptr
	native.Register(&fn, sym)
	return fn() != 0, nil
}

func open(names ...string) (uintptr, error) {
	var err error
	for _, name := range names {
		var lib uintptr
		lib, err = native.Open(name, native.Now)
		if err == nil {
			return lib, nil
		}
	}
	return 0, err
}
