package report

import (
	"errors"

	"github.com/lewtec/lewkit/x/errs"
)

var (
	ErrLevel  = errors.New("level")
	ErrFormat = errors.New("format")
	ErrSpan   = errs.Wrap(errs.ErrOutOfRange, "span out of range")
)
