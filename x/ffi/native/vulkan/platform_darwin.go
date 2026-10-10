//go:build darwin

package vulkan

import (
	"log/slog"
	"sync"

	"github.com/lewtec/lewkit/x/ffi/native"
)

var (
	platformOnce       sync.Once
	metalDefaultDevice uintptr
	metal              Metal
)

// ensurePlatform wakes Metal so MoltenVK can enumerate a GPU without a window.
// Doctor never starts AppKit; triangle does, which is why the UI saw Vulkan
// and doctor did not.
func ensurePlatform() {
	platformOnce.Do(func() {
		var err error
		metal, err = native.Bind[Metal](native.Lazy, "/System/Library/Frameworks/Metal.framework/Metal")
		if err != nil {
			slog.Debug("vulkan metal create", "err", err)
			return
		}
		if metal.MTLCreateSystemDefaultDevice == nil {
			slog.Debug("vulkan metal create missing")
			return
		}
		metalDefaultDevice = metal.MTLCreateSystemDefaultDevice()
		slog.Debug("vulkan metal device", "ok", metalDefaultDevice != 0)
	})
}
