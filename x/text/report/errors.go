package report

import "github.com/lewtec/lewkit/x/errs"

var (
	ErrSpan = errs.Wrap(errs.ErrOutOfRange, "span out of range")
)
