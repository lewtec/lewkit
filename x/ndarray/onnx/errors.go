package onnx

import "errors"

var (
	ErrEmpty = errors.New("empty model")
	ErrModel = errors.New("invalid model")
	ErrGraph = errors.New("missing graph")
	ErrArity = errors.New("want one input and one output")
	ErrOp    = errors.New("unsupported op")
)
