package glsl

import "github.com/lewtec/lewkit/x/errs"

var (
	// ErrSyntax means the text is not a shader this package reads.
	ErrSyntax = errs.Wrap(errs.ErrSyntax, "glsl syntax")
	// ErrValue means a file, a reader, or a writer is missing.
	ErrValue = errs.Wrap(errs.ErrInvalid, "glsl value")
)
