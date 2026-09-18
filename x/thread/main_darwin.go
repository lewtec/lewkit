//go:build darwin

package thread

import (
	"sync"

	"github.com/ebitengine/purego"
)

var (
	mainOnce sync.Once
	mainNP   func() int32
)

// ProcessMain reports whether this goroutine is on the process main OS thread.
func ProcessMain() bool {
	mainOnce.Do(func() {
		lib, err := purego.Dlopen(libcPath, purego.RTLD_LAZY)
		if err != nil {
			return
		}
		purego.RegisterLibFunc(&mainNP, lib, "pthread_main_np")
	})
	if mainNP != nil {
		return mainNP() != 0
	}
	return processMainTID != 0 && osThread() == processMainTID
}
