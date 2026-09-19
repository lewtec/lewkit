// Package ndarray maps n-d views onto a flat buffer and fuses a graph of
// 21 ALU ops into one kernel.
//
// [Tensor] is the brick. It holds a lazy op tree. Leaves own a host buffer;
// [Tensor.Reshape], [Tensor.Permute], and the other view ops share that
// buffer. ALU ops return a new Tensor. [Tensor.Eval] compiles the tree to
// a [Kernel] (GLSL + CPU tape) and runs it on an [Evaluator]. Device
// pipelines live in [github.com/lewtec/lewkit/x/driver/ndeval].
//
// [Zeros], [Ones], [Full], [Rand], [New], [Const], and [Coord] build tensors.
// [Shape] is the dimensions ([]int so Reshape can use -1).
//
// [Of] and [Tracker] are the address map under a tensor. Layers live in
// [github.com/lewtec/lewkit/x/ndarray/nn]. Drawing is
// [github.com/lewtec/lewkit/x/ndarray/image].
package ndarray
