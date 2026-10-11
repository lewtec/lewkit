//go:build (darwin || linux) && !android

package thread

import "github.com/lewtec/lewkit/x/ffi/native"

// pthread loads libc and returns pthread_self.
// Later calls return that same binding.
var pthread = native.Singleton(func() (Pthread, error) {
	return native.Bind[Pthread](native.Lazy, libcPath)
})

func osThread() uint64 {
	lib, err := pthread()
	if err != nil || lib.PthreadSelf == nil {
		return 0
	}
	return uint64(lib.PthreadSelf())
}
