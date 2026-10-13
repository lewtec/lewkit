package clib

import "errors"

var (
	ErrHeader = errors.New("header required")
	ErrEmpty  = errors.New("no functions")
	ErrType   = errors.New("unsupported type")
	ErrSymbol = errors.New("unsupported symbol")
	ErrHost   = errors.New("unsupported target")
)
