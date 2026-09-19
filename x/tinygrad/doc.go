// Package tinygrad maps tensor movement onto a flat buffer and fuses a graph
// of tinygrad's 21 ALU ops into one Vulkan compute kernel.
//
// [Of] starts a contiguous row-major view. [Tracker.Reshape], [Tracker.Permute],
// [Tracker.Expand], [Tracker.Pad], [Tracker.Shrink], and [Tracker.Flip] change
// how cells are addressed. [Tracker.Index] turns a logical coordinate into a
// buffer offset and a valid bit (false in padding).
//
// [In] plus [Add], [Mul], and the other 21 ops build an expression. [Compile]
// emits one GLSL compute shader. [Kernel.Run] / [Kernel.Exec] dispatch it once.
package tinygrad
