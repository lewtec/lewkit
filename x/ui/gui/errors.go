package gui

import "errors"

var (
	ErrModel = errors.New("nil model")
	ErrView  = errors.New("nil view")
	// ErrCanceled means the welcome window closed without a folder.
	ErrCanceled = errors.New("directory canceled")
)
