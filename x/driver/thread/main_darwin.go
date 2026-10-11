//go:build darwin

package thread

import "github.com/lewtec/lewkit/x/ffi/native"

var pthreadMain PthreadMain

// ProcessMain reports whether this goroutine is on the process main OS thread.
var loadMain = native.Once(func() error {
	var err error
	pthreadMain, err = native.Bind[PthreadMain](native.Lazy, libcPath)
	return err
})

func ProcessMain() bool {
	if loadMain() == nil && pthreadMain.PthreadMainNp != nil {
		return pthreadMain.PthreadMainNp() != 0
	}
	return processMainTID != 0 && osThread() == processMainTID
}
