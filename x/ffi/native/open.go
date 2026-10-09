package native

import (
	"errors"
	"sync"
)

// loadLibrary is the platform opener. Tests replace it.
var loadLibrary = openPath

type librarySlot struct {
	mu     sync.Mutex
	handle uintptr
	flags  int
	loaded bool
	flight *libraryCall
}

type libraryCall struct {
	done   chan struct{}
	handle uintptr
	flags  int
	err    error
}

var (
	libraries     sync.Map // path -> *librarySlot
	errNoLibrary  = errors.New("no library path")
	errNilContext = errors.New("nil context")
)

func slotFor(path string) *librarySlot {
	if v, ok := libraries.Load(path); ok {
		return v.(*librarySlot)
	}
	fresh := &librarySlot{}
	actual, existed := libraries.LoadOrStore(path, fresh)
	if existed {
		return actual.(*librarySlot)
	}
	return fresh
}

// mergeFlags ORs mode bits. RTLD_NOW wins over RTLD_LAZY. RTLD_GLOBAL wins over RTLD_LOCAL.
func mergeFlags(have, want int) int {
	out := have | want
	if out&Now != 0 {
		out &^= Lazy
	}
	if out&Global != 0 {
		out &^= Local
	}
	return out
}

func flagsCover(have, want int) bool {
	return mergeFlags(have, want) == have
}

// Open loads path. flags 0 means Lazy.
//
// Concurrent callers of the same path share one load. A successful load is
// reused when the cached flags already cover the request. A later request
// with stronger flags loads again so RTLD_GLOBAL or RTLD_NOW can take effect.
// A failed load is not remembered.
func Open(path string, flags int) (uintptr, error) {
	if flags == 0 {
		flags = Lazy
	}
	slot := slotFor(path)
	for {
		slot.mu.Lock()
		if slot.loaded && flagsCover(slot.flags, flags) {
			handle := slot.handle
			slot.mu.Unlock()
			return handle, nil
		}
		if slot.flight != nil {
			call := slot.flight
			slot.mu.Unlock()
			<-call.done
			if flagsCover(call.flags, flags) {
				return call.handle, call.err
			}
			continue
		}
		openFlags := flags
		if slot.loaded {
			openFlags = mergeFlags(slot.flags, flags)
		}
		call := &libraryCall{done: make(chan struct{}), flags: openFlags}
		slot.flight = call
		slot.mu.Unlock()

		handle, err := loadLibrary(path, openFlags)

		slot.mu.Lock()
		if err == nil {
			slot.handle = handle
			slot.flags = openFlags
			slot.loaded = true
			call.handle = handle
		}
		call.err = err
		slot.flight = nil
		close(call.done)
		slot.mu.Unlock()
		return handle, err
	}
}

// OpenFirst loads the first path that succeeds. flags 0 means Lazy.
// Every path failing returns the last error. No paths returns an error.
func OpenFirst(flags int, paths ...string) (uintptr, error) {
	var last error
	for _, path := range paths {
		lib, err := Open(path, flags)
		if err == nil {
			return lib, nil
		}
		last = err
	}
	if last == nil {
		last = errNoLibrary
	}
	return 0, last
}
