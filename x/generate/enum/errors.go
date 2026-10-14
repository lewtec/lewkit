package enum

import "github.com/lewtec/lewkit/x/errs"

var (
	// ErrNoEnum means Run found no lewkit:enum type.
	ErrNoEnum = errs.Wrap(errs.ErrNotFound, "no enum")
	// ErrEnum means a marked type is not a closed integer or string set.
	ErrEnum = errs.Wrap(errs.ErrInvalid, "invalid enum")
)
