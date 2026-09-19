//go:build darwin

package vulkan

import (
	"log/slog"
	"sync"

	"github.com/lewtec/lewkit/x/ffi"
)

var (
	platformOnce       sync.Once
	metalDefaultDevice uintptr
	createMetalDevice  func() uintptr
)

// ensurePlatform wakes Metal so MoltenVK can enumerate a GPU without a window.
// Doctor never starts AppKit; triangle does, which is why the UI saw Vulkan
// and doctor did not.
func ensurePlatform() {
	platformOnce.Do(func() {
		lib, err := ffi.Open("/System/Library/Frameworks/Metal.framework/Metal", ffi.Lazy)
		if err != nil {
			slog.Debug("vulkan metal framework", "err", err)
			return
		}
		ffi.Func(lib, "MTLCreateSystemDefaultDevice", &createMetalDevice)
		if createMetalDevice == nil {
			slog.Debug("vulkan metal create missing")
			return
		}
		metalDefaultDevice = createMetalDevice()
		slog.Debug("vulkan metal device", "ok", metalDefaultDevice != 0)
	})
}
