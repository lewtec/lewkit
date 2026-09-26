package report

import "errors"

var (
	ErrLevel  = errors.New("level")
	ErrFormat = errors.New("format")
	ErrSpan   = errors.New("span out of range")
)
