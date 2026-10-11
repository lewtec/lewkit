//go:build (darwin || linux) && !android

package thread

import "github.com/lewtec/lewkit/x/ffi/native"

// pthread is pthread_self from libc.
var pthread = native.BindOnce[Pthread](native.Lazy, libcPath)

func osThread() uint64 {
	lib, err := pthread()
	if err != nil {
		return 0
	}
	return uint64(lib.PthreadSelf())
}
