package onnx

import (
	"errors"

	"github.com/lewtec/lewkit/x/errs"
)

var (
	ErrEmpty = errs.Wrap(errs.ErrEmpty, "empty model")
	ErrModel = errs.Wrap(errs.ErrInvalid, "invalid model")
	ErrGraph = errs.Wrap(errs.ErrMissing, "missing graph")
	ErrArity = errors.New("want one input and one output")
	ErrOp    = errs.Wrap(errs.ErrUnsupported, "unsupported op")
)
