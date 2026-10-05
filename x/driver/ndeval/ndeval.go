// Package ndeval registers ndarray.Evaluator drivers.
//
// CPU is always available. The Vulkan and Metal evaluators register from
// the blank imports below. Import this package or
// [github.com/lewtec/lewkit/x/driver/prelude].
// [github.com/lewtec/lewkit/x/ndarray.Open] picks the highest-weight
// compatible evaluator.
package ndeval

import (
	_ "github.com/lewtec/lewkit/x/driver/ndeval/metal"
	_ "github.com/lewtec/lewkit/x/driver/ndeval/vulkan"
)
