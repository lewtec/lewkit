package golang

import "github.com/lewtec/lewkit/x/errs"

var (
	errNil    = errs.Wrap(errs.ErrInvalid, "nil go file")
	errRead   = errs.Wrap(errs.ErrInvalid, "read go source")
	errParse  = errs.Wrap(errs.ErrSyntax, "parse go source")
	errPrint  = errs.Wrap(errs.ErrInvalid, "print go source")
	errSyntax = errs.Wrap(errs.ErrUnsupported, "unsupported go syntax")
)
