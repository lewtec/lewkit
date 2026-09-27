//go:build darwin || linux

package std

import (
	"sync"

	"github.com/lewtec/lewkit/x/ffi/native"
)

var (
	libcOnce sync.Once
	self     func() uintptr
)

func osThread() uint64 {
	libcOnce.Do(func() {
		lib, err := native.Open(libcPath, native.Lazy)
		if err != nil {
			return
		}
		native.Func(lib, "pthread_self", &self)
	})
	if self == nil {
		return 0
	}
	return uint64(self())
}
