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
	dispatchAsyncF       func(queue uintptr, ctx unsafe.Pointer, work uintptr)
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
		purego.RegisterLibFunc(&dispatchAsyncF, handle, "dispatch_async_f")
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

var (
	postMu     sync.Mutex
	postFns    []func()
	postQueued bool
)

func postedMain(unsafe.Pointer) {
	postMu.Lock()
	fns := postFns
	postFns = nil
	postQueued = false
	postMu.Unlock()
	for _, fn := range fns {
		fn()
	}
}

var postCallback = purego.NewCallback(postedMain)

// PostMain queues fn on the main queue.
// It stays queued when the caller is already on that thread, so a run loop
// can run fn while CFRunLoopRun is still on the stack.
func PostMain(fn func()) {
	if fn == nil {
		return
	}
	loadMain()
	if loadErr != nil || dispatchAsyncF == nil || dispatchGetMainQueue == nil {
		fn()
		return
	}
	postMu.Lock()
	postFns = append(postFns, fn)
	queue := !postQueued
	postQueued = true
	postMu.Unlock()
	if queue {
		dispatchAsyncF(dispatchGetMainQueue(), nil, postCallback)
	}
}
