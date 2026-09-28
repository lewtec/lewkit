//go:build darwin || linux

package thread

import "github.com/lewtec/lewkit/x/ffi/native"

var (
	libc     uintptr
	self     func() uintptr
	loadLibc = native.Once(func() error {
		lib, err := native.OpenChain(native.Lazy, libcPath)
		if err != nil {
			return err
		}
		libc = lib
		return native.Bind(lib, "pthread_self", &self)
	})
)

func osThread() uint64 {
	if err := loadLibc(); err != nil {
		return 0
	}
	if self == nil {
		return 0
	}
	return uint64(self())
}
