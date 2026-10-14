package enum

import "github.com/lewtec/lewkit/x/errs"

var (
	// ErrFileRequired means Run was given no Go file.
	ErrFileRequired = errs.Wrap(errs.ErrRequired, "go file required")
	// ErrNoEnum means a file has no lewkit:enum type.
	ErrNoEnum = errs.Wrap(errs.ErrNotFound, "no enum")
	// ErrEnum means a marked type is not a closed integer or string set.
	ErrEnum = errs.Wrap(errs.ErrInvalid, "invalid enum")
)
