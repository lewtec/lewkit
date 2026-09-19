// Package ndeval registers ndarray.Evaluator drivers.
//
// CPU is always available. Vulkan wraps the selected
// [github.com/lewtec/lewkit/x/driver/vulkan] GPU.
// Import [github.com/lewtec/lewkit/x/driver/prelude] or this package
// so the factories register. [github.com/lewtec/lewkit/x/ndarray.Open]
// then picks Vulkan if a GPU exists, else CPU.
package ndeval

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	ffivulkan "github.com/lewtec/lewkit/x/ffi/vulkan"
	"github.com/lewtec/lewkit/x/ndarray"
)

// Open is the highest-weight compatible evaluator.
func Open(ctx context.Context) (ndarray.Evaluator, error) {
	return driver.Get[ndarray.Evaluator](ctx)
}

// New wraps an already-open FFI device. The caller closes the device.
func New(device *ffivulkan.Device) ndarray.Evaluator {
	return newGPU(device, false)
}
