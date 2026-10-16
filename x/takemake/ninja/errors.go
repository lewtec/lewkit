package ninja

import "github.com/lewtec/lewkit/x/errs"

var (
	errFile   = errs.Wrap(errs.ErrInvalid, "nil ninja")
	errRead   = errs.Wrap(errs.ErrInvalid, "ninja read")
	errWrite  = errs.Wrap(errs.ErrInvalid, "ninja write")
	errSyntax = errs.Wrap(errs.ErrSyntax, "ninja syntax")
)
