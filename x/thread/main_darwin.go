//go:build darwin

package thread

import "github.com/lewtec/lewkit/x/ffi"

var pthreadMainNP func() int32

// ProcessMain reports whether this goroutine is on the process main OS thread.
func ProcessMain() bool {
	loadLibc()
	if pthreadMainNP == nil && libc != 0 {
		ffi.Func(libc, "pthread_main_np", &pthreadMainNP)
	}
	if pthreadMainNP != nil {
		return pthreadMainNP() != 0
	}
	return processMainTID != 0 && osThread() == processMainTID
}
