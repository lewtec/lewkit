package avutil

import "github.com/lewtec/lewkit/x/errs"

var (
	// ErrInvalidData means a bitstream or header is not usable.
	ErrInvalidData = errs.Wrap(errs.ErrInvalid, "invalid data")
	// ErrOption means an option name or value is not usable.
	ErrOption = errs.Wrap(errs.ErrInvalid, "invalid option")
	// ErrRange means a number is outside the accepted interval.
	ErrRange = errs.Wrap(errs.ErrOutOfRange, "out of range")
	// ErrBug means an internal invariant failed.
	ErrBug = errs.Wrap(errs.ErrInvalid, "internal error")
	// ErrUnsupported means this build has no implementation for the request.
	ErrUnsupported = errs.Wrap(errs.ErrUnsupported, "unsupported")
)
