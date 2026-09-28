//go:build darwin

package thread

import "github.com/lewtec/lewkit/x/ffi/native"

var pthreadMainNP func() int32

// ProcessMain reports whether this goroutine is on the process main OS thread.
var loadMain = native.Once(func() error {
	if err := loadLibc(); err != nil {
		return err
	}
	return native.Bind(libc, "pthread_main_np", &pthreadMainNP)
})

func ProcessMain() bool {
	if loadMain() == nil && pthreadMainNP != nil {
		return pthreadMainNP() != 0
	}
	return processMainTID != 0 && osThread() == processMainTID
}
