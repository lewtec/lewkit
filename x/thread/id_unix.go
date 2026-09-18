//go:build darwin || linux

package thread

import (
	"sync"

	"github.com/lewtec/lewkit/x/ffi"
)

var (
	libcOnce sync.Once
	libc     uintptr
	self     func() uintptr
)

func loadLibc() {
	libcOnce.Do(func() {
		lib, err := ffi.Open(libcPath, ffi.Lazy)
		if err != nil {
			return
		}
		libc = lib
		ffi.Func(lib, "pthread_self", &self)
	})
}

func osThread() uint64 {
	loadLibc()
	if self == nil {
		return 0
	}
	return uint64(self())
}
