// Package ndarray maps n-d views onto a flat buffer and fuses a graph of
// 21 ALU ops into one kernel.
//
// [Tensor] is the brick. It holds a lazy op tree. Leaves own a host buffer;
// [Tensor.Reshape], [Tensor.Permute], and the other view ops share that
// buffer. ALU ops return a new Tensor. [Tensor.Eval] flattens the tree to
// a [Kernel] (graph descriptor) and [Evaluator.Exec] returns backend code
// ([Exec]) to run it. CPU holds a tape; Vulkan holds a session. Pipelines
// live in
// [github.com/lewtec/lewkit/x/driver/ndeval].
//
// [Zeros], [Ones], [Full], [Rand], [New], [Const], and [Coord] build tensors.
// [Shape] is the dimensions ([]int so Reshape can use -1).
//
// [Of] and [Tracker] are the address map under a tensor. Layers live in
// [github.com/lewtec/lewkit/x/ndarray/nn]. Pictures pack through
// [github.com/lewtec/lewkit/x/ndarray/image].
package ndarray
