// Package ndeval registers ndarray.Evaluator drivers.
//
// CPU is always available. Vulkan wraps the selected
// [github.com/lewtec/lewkit/x/driver/vulkan] GPU.
// Import this package or [github.com/lewtec/lewkit/x/driver/prelude];
// init registers the factories. [github.com/lewtec/lewkit/x/ndarray.Open]
// then picks Vulkan if a GPU exists, else CPU.
package ndeval
