// Package onnx reads ONNX ModelProto graphs and lowers them onto ndarray.
//
// FunctionOf[T] walks the ONNX graph and lowers each node onto ndarray.
// Concat, reduce, scan, and gather are ndarray graphs (Pad/Add, Shrink loops).
// T is ndarray.Number. INT64 TensorProto values are shapes, not T.
//
// Node tests live in testdata/node/<name>/model.onnx plus test_data_set_*/
// input_*.pb and output_*.pb. CoverageFromResults scores them against ai.onnx.
package onnx
