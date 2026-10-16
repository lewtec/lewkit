// Package onnx is one ONNX model.
//
// [New] returns an empty [File]. Methods return the receiver and store
// the child: input, output, initializer, and node. [Take] reads a
// ModelProto. [Make] writes it. [Take], then [Make], then [Take]
// produce the same file.
//
// The file keeps the model header, opsets, and graph. The graph holds
// inputs, outputs, value infos, initializers, sparse initializers, and
// nodes. A node attribute can be a scalar, a list, a tensor, a graph,
// a sparse tensor, or a type. Metadata, training info, functions, and
// device configuration are left out. Evaluating the graph stays in
// x/ndarray/onnx.
//
// The contract is in DESIGN.md beside the parent directory.
package onnx
