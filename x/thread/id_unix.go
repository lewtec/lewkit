//go:build darwin || linux

package thread

import (
	"sync"

	"github.com/ebitengine/purego"
)

var (
	idOnce sync.Once
	self   func() uintptr
)

func osThread() uint64 {
	idOnce.Do(func() {
		lib, err := purego.Dlopen(libcPath, purego.RTLD_LAZY)
		if err != nil {
			return
		}
		purego.RegisterLibFunc(&self, lib, "pthread_self")
	})
	if self == nil {
		return 0
	}
	return uint64(self())
}
