package glsl

import "github.com/lewtec/lewkit/x/errs"

var (
	// ErrSyntax means the text is not GLSL this package reads.
	ErrSyntax = errs.Wrap(errs.ErrSyntax, "glsl syntax")
)
