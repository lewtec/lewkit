//go:build darwin || linux

package thread

import (
	"sync"

	"github.com/lewtec/lewkit/x/ffi/native"
)

var (
	libcOnce sync.Once
	libc     uintptr
	self     func() uintptr
)

func loadLibc() {
	libcOnce.Do(func() {
		lib, err := native.Open(libcPath, native.Lazy)
		if err != nil {
			return
		}
		libc = lib
		native.Func(lib, "pthread_self", &self)
	})
}

func osThread() uint64 {
	loadLibc()
	if self == nil {
		return 0
	}
	return uint64(self())
}
