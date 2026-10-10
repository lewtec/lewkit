//go:build (darwin || linux) && !android

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
		got, err := native.Bind[Pthread](lib)
		if err != nil {
			return err
		}
		self = got.PthreadSelf
		return nil
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
