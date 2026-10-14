package ndarray

import (
	"errors"

	"github.com/lewtec/lewkit/x/errs"
)

var (
	ErrShape  = errs.Wrap(errs.ErrInvalid, "invalid shape")
	ErrAxis   = errs.Wrap(errs.ErrInvalid, "invalid axis")
	ErrSize   = errors.New("size mismatch")
	ErrExpand = errs.Wrap(errs.ErrInvalid, "invalid expand")
	ErrPad    = errs.Wrap(errs.ErrInvalid, "invalid pad")
	ErrShrink = errs.Wrap(errs.ErrInvalid, "invalid shrink")
	ErrIndex  = errs.Wrap(errs.ErrOutOfRange, "index out of range")
	ErrOp     = errs.Wrap(errs.ErrInvalid, "invalid op")
	ErrType   = errs.Wrap(errs.ErrInvalid, "invalid dtype")
)
