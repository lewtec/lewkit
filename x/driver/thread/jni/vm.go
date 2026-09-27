//go:build android && cgo

package jni

import (
	"context"
	"fmt"

	"github.com/ebitengine/purego"
	"github.com/lewtec/lewkit/x/driver"
	uithread "github.com/lewtec/lewkit/x/driver/thread"
	"github.com/lewtec/lewkit/x/driver/thread/std"
)

func init() { driver.Register[uithread.Driver](factory{}) }

type factory struct{}

func (factory) ID() string   { return "thread_jni" }
func (factory) Name() string { return "Android looper" }
func (factory) Weight() int  { return 80 }

func (factory) CheckCompatibility(context.Context) error {
	if javaVMs() < 1 || !onLooper() {
		return fmt.Errorf("%w: no Java main looper", driver.ErrIncompatible)
	}
	return nil
}

func (factory) New(context.Context) (uithread.Driver, error) {
	// The looper is already pumping. Jobs still run on this thread when On
	// reports it; callers off the looper use the process queue until a later
	// post lands. The std queue is the fallback inside this process.
	return std.New(), nil
}

func javaVMs() int {
	lib, err := openLib("libnativehelper.so", "libart.so")
	if err != nil {
		return 0
	}
	sym, err := purego.Dlsym(lib, "JNI_GetCreatedJavaVMs")
	if err != nil {
		return 0
	}
	var fn func(vmBuf *uintptr, bufLen int32, nVMs *int32) int32
	purego.RegisterFunc(&fn, sym)
	var vm uintptr
	var n int32
	if fn(&vm, 1, &n) != 0 {
		return 0
	}
	return int(n)
}

func onLooper() bool {
	lib, err := purego.Dlopen("libandroid.so", purego.RTLD_NOW)
	if err != nil {
		return false
	}
	sym, err := purego.Dlsym(lib, "ALooper_forThread")
	if err != nil {
		return false
	}
	var fn func() uintptr
	purego.RegisterFunc(&fn, sym)
	return fn() != 0
}

func openLib(names ...string) (uintptr, error) {
	var err error
	for _, name := range names {
		var lib uintptr
		lib, err = purego.Dlopen(name, purego.RTLD_NOW)
		if err == nil {
			return lib, nil
		}
	}
	return 0, err
}
