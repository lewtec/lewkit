package schema

import "github.com/lewtec/lewkit/x/errs"

var (
	// ErrNotFound means a table, column, or index is not in the catalog.
	ErrNotFound = errs.Wrap(errs.ErrNotFound, "catalog entry not found")
	// ErrDuplicate means a table, column, or index is already in the catalog.
	ErrDuplicate = errs.Wrap(errs.ErrDuplicate, "duplicate catalog entry")
	// ErrInvalid means a statement does not fit the catalog.
	ErrInvalid = errs.Wrap(errs.ErrInvalid, "invalid catalog change")
)
