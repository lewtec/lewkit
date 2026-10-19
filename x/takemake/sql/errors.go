package sql

import "github.com/lewtec/lewkit/x/errs"

var (
	// ErrSyntax means the text is not a statement log this package reads.
	ErrSyntax = errs.Wrap(errs.ErrSyntax, "sql syntax")
	// ErrInvalid means a file or a statement is missing a required part.
	ErrInvalid = errs.Wrap(errs.ErrInvalid, "invalid sql")
)
