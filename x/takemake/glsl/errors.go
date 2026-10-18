package glsl

import (
	"github.com/lewtec/lewkit/x/errs"
	syntax "github.com/lewtec/lewkit/x/parse/glsl"
)

var (
	// ErrSyntax means the text is not a shader this package reads.
	ErrSyntax = syntax.ErrSyntax
	// ErrValue means a file, a reader, or a writer is missing.
	ErrValue = errs.Wrap(errs.ErrInvalid, "glsl value")
)
