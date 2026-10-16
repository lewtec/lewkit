package makefile

import "github.com/lewtec/lewkit/x/errs"

var (
	errFile   = errs.Wrap(errs.ErrInvalid, "nil makefile")
	errRead   = errs.Wrap(errs.ErrInvalid, "makefile read")
	errWrite  = errs.Wrap(errs.ErrInvalid, "makefile write")
	errSyntax = errs.Wrap(errs.ErrSyntax, "makefile syntax")
)
