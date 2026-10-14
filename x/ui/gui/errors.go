package gui

import (
	"errors"

	"github.com/lewtec/lewkit/x/errs"
)

var (
	ErrModel = errors.New("nil model")
	ErrView  = errors.New("nil view")
	// ErrCanceled means the welcome window closed without a folder.
	ErrCanceled = errs.Wrap(errs.ErrCanceled, "directory canceled")
)
