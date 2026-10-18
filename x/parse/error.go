package parse

import (
	"fmt"
	"strings"

	"github.com/lewtec/lewkit/x/errs"
)

// Error is a syntax error at a byte offset. errors.Is matches errs.ErrSyntax.
type Error struct {
	Offset int
	Line   int
	Col    int
	Want   []string
}

func (e *Error) Error() string {
	if e == nil {
		return "syntax"
	}
	if len(e.Want) == 0 {
		return fmt.Sprintf("syntax: %d:%d", e.Line, e.Col)
	}
	return fmt.Sprintf("syntax: %d:%d: expected %s", e.Line, e.Col, strings.Join(e.Want, ", "))
}

func (e *Error) Unwrap() error { return errs.ErrSyntax }
