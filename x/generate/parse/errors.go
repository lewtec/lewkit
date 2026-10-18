package parse

import "github.com/lewtec/lewkit/x/errs"

var (
	// ErrNoPackage means the working directory is not a Go package.
	ErrNoPackage = errs.Wrap(errs.ErrNotFound, "parse: no package")
	// ErrTable means ParseTable did not return an LK01 table.
	ErrTable = errs.Wrap(errs.ErrInvalid, "parse: table")
)
