//go:build ios

package dispatch

import (
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// OnMain runs fn on the UIKit main thread.
// A call that is already there runs fn directly.
// libdispatch is loaded with purego so this package stays cgo-free.

var (
	loadOnce sync.Once
	loadErr  error

	pthreadMainNP        func() int
	dispatchSyncF        func(queue uintptr, ctx unsafe.Pointer, work uintptr)
	dispatchGetMainQueue func() uintptr
)

func loadMain() {
	loadOnce.Do(func() {
		handle, err := purego.Dlopen("/usr/lib/libSystem.B.dylib", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err != nil {
			loadErr = err
			return
		}
		purego.RegisterLibFunc(&pthreadMainNP, handle, "pthread_main_np")
		purego.RegisterLibFunc(&dispatchSyncF, handle, "dispatch_sync_f")
		purego.RegisterLibFunc(&dispatchGetMainQueue, handle, "dispatch_get_main_queue")
	})
}

var (
	callMu sync.Mutex
	callFn func()
)

func lewkitDispatchMain(unsafe.Pointer) {
	if callFn != nil {
		callFn()
	}
}

var mainCallback = purego.NewCallback(lewkitDispatchMain)

func OnMain(fn func()) {
	if fn == nil {
		return
	}
	loadMain()
	if loadErr != nil || pthreadMainNP == nil {
		fn()
		return
	}
	if pthreadMainNP() != 0 {
		fn()
		return
	}
	callMu.Lock()
	defer callMu.Unlock()
	callFn = fn
	dispatchSyncF(dispatchGetMainQueue(), nil, mainCallback)
	callFn = nil
}
