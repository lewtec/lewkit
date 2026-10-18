package proto

import "github.com/lewtec/lewkit/x/errs"

var (
	// ErrSyntax means the text is not a proto file this package reads.
	ErrSyntax = errs.Wrap(errs.ErrSyntax, "proto syntax")
)
