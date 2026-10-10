//go:build darwin

package thread

import "github.com/lewtec/lewkit/x/ffi/native"

var mainMain Main

// ProcessMain reports whether this goroutine is on the process main OS thread.
var loadMain = native.Once(func() error {
	var err error
	mainMain, err = native.Bind[Main](native.Lazy, libcPath)
	return err
})

func ProcessMain() bool {
	if loadMain() == nil && mainMain.PthreadMainNp != nil {
		return mainMain.PthreadMainNp() != 0
	}
	return processMainTID != 0 && osThread() == processMainTID
}
