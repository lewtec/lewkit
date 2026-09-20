package gui

import "errors"

var (
	ErrModel = errors.New("nil model")
	ErrView  = errors.New("nil view")
)
