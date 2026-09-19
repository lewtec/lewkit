// Package tinygrad maps tensor movement onto a flat buffer without copying.
//
// [Of] starts a contiguous row-major view. [Tracker.Reshape], [Tracker.Permute],
// [Tracker.Expand], [Tracker.Pad], [Tracker.Shrink], and [Tracker.Flip] change
// how cells are addressed. [Tracker.Index] turns a logical coordinate into a
// buffer offset and a valid bit (false in padding).
package tinygrad
