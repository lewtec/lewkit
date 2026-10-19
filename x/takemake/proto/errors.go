package proto

import (
	"github.com/lewtec/lewkit/x/errs"
	syntax "github.com/lewtec/lewkit/x/parse/proto"
)

var (
	// ErrSyntax means the text is not a proto file this package reads.
	ErrSyntax = syntax.ErrSyntax
	// ErrValue means a file, a reader, a writer, or a constant is missing.
	ErrValue = errs.Wrap(errs.ErrInvalid, "proto value")
)
