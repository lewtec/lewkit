// Package onnx is one ONNX graph.
//
// [New] returns an empty [File]. Methods return the receiver and store
// the child: input, output, initializer, and node. [Take] reads a
// ModelProto. [Make] writes it. Evaluating the graph stays in
// x/ndarray/onnx. The contract is in DESIGN.md beside the parent
// directory. This scaffold returns
// [github.com/lewtec/lewkit/x/errs.ErrScaffold].
package onnx
