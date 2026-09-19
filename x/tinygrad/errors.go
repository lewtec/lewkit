package tinygrad

import "errors"

var (
	ErrShape  = errors.New("invalid shape")
	ErrAxis   = errors.New("invalid axis")
	ErrSize   = errors.New("size mismatch")
	ErrExpand = errors.New("invalid expand")
	ErrPad    = errors.New("invalid pad")
	ErrShrink = errors.New("invalid shrink")
	ErrIndex  = errors.New("index out of range")
)
