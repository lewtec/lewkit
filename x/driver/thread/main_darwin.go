//go:build darwin

package thread

import "github.com/lewtec/lewkit/x/ffi/native"

// pthreadMain loads libc and returns pthread_main_np.
// Later calls return that same binding.
var pthreadMain = native.Singleton(func() (PthreadMain, error) {
	return native.Bind[PthreadMain](native.Lazy, libcPath)
})

// ProcessMain reports whether this goroutine is on the process main OS thread.
func ProcessMain() bool {
	main, err := pthreadMain()
	if err == nil && main.PthreadMainNp != nil {
		return main.PthreadMainNp() != 0
	}
	return processMainTID != 0 && osThread() == processMainTID
}
