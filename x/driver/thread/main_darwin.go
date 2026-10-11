//go:build darwin

package thread

import "github.com/lewtec/lewkit/x/ffi/native"

// pthreadMain is pthread_main_np from libc.
var pthreadMain = native.BindOnce[PthreadMain](native.Lazy, libcPath)

// ProcessMain reports whether this goroutine is on the process main OS thread.
func ProcessMain() bool {
	main, err := pthreadMain()
	if err == nil {
		return main.PthreadMainNp() != 0
	}
	return processMainTID != 0 && osThread() == processMainTID
}
