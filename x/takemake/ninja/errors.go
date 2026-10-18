package ninja

import (
	"github.com/lewtec/lewkit/x/errs"
	syntax "github.com/lewtec/lewkit/x/parse/ninja"
)

var (
	errFile   = errs.Wrap(errs.ErrInvalid, "nil ninja")
	errRead   = errs.Wrap(errs.ErrInvalid, "ninja read")
	errWrite  = errs.Wrap(errs.ErrInvalid, "ninja write")
	errSyntax = syntax.ErrSyntax
)
