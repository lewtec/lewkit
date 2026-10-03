//go:build darwin

package dispatch

import (
	"runtime"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/lewtec/lewkit/x/ffi/native"
)

var (
	loadOnce sync.Once
	loadErr  error
	syncFn   func(queue uintptr, ctx unsafe.Pointer, work uintptr)
	mainQ    func() uintptr
	mainNP   func() int32
	callMu   sync.Mutex
	callFn   func()
	tramp    uintptr
)

// OnMain runs fn on the UIKit main queue when this process is iOS.
// macOS keeps its own UI thread, so fn runs on the caller there.
func OnMain(fn func()) {
	if fn == nil {
		return
	}
	if runtime.GOOS != "ios" {
		fn()
		return
	}
	loadOnce.Do(load)
	if loadErr != nil || mainNP == nil || syncFn == nil || mainQ == nil {
		fn()
		return
	}
	if mainNP() != 0 {
		fn()
		return
	}
	callMu.Lock()
	defer callMu.Unlock()
	callFn = fn
	syncFn(mainQ(), nil, tramp)
	callFn = nil
}

func load() {
	lib, err := native.Open("/usr/lib/libSystem.B.dylib", native.Global|native.Lazy)
	if err != nil {
		loadErr = err
		return
	}
	native.Func(lib, "dispatch_sync_f", &syncFn)
	native.Func(lib, "dispatch_get_main_queue", &mainQ)
	native.Func(lib, "pthread_main_np", &mainNP)
	tramp = purego.NewCallback(func(unsafe.Pointer) {
		if callFn != nil {
			callFn()
		}
	})
}
