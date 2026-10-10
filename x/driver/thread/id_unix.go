//go:build (darwin || linux) && !android

package thread

import "github.com/lewtec/lewkit/x/ffi/native"

var (
	pthread  Pthread
	loadLibc = native.Once(func() error {
		var err error
		pthread, err = native.Bind[Pthread](native.Lazy, libcPath)
		return err
	})
)

func osThread() uint64 {
	if err := loadLibc(); err != nil {
		return 0
	}
	if pthread.PthreadSelf == nil {
		return 0
	}
	return uint64(pthread.PthreadSelf())
}
