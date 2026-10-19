package c

import "github.com/lewtec/lewkit/x/errs"

var (
	errSyntax = errs.Wrap(errs.ErrSyntax, "c syntax")
	errRead   = errs.Wrap(errs.ErrInvalid, "c read")
	errWrite  = errs.Wrap(errs.ErrMissing, "nil writer")
	errFile   = errs.Wrap(errs.ErrMissing, "nil file")
	errForm   = errs.Wrap(errs.ErrUnsupported, "unsupported c")
)
