// Package ndeval registers ndarray.Evaluator drivers.
//
// CPU is always available. The Vulkan, Metal, and OpenGL evaluators register
// from the blank imports below. Import this package or
// [github.com/lewtec/lewkit/x/driver/prelude].
// [github.com/lewtec/lewkit/x/ndarray.Open] picks the highest-weight
// compatible evaluator. OpenGL is weight 20, after Vulkan and before CPU.
// On Apple that factory is incompatible and Metal stays the evaluator.
package ndeval

import (
	_ "github.com/lewtec/lewkit/x/driver/ndeval/metal"
	_ "github.com/lewtec/lewkit/x/driver/ndeval/opengl"
	_ "github.com/lewtec/lewkit/x/driver/ndeval/vulkan"
)
