package parse

import (
	"fmt"
	"strings"

	"github.com/lewtec/lewkit/x/errs"
)

// Error is a syntax error at a byte offset. errors.Is matches errs.ErrSyntax.
// Detail is set when the failure is a search bound rather than a token
// the grammar rejected.
type Error struct {
	Offset int
	Line   int
	Col    int
	Want   []string
	Detail string
}

func (e *Error) Error() string {
	if e == nil {
		return "syntax"
	}
	if e.Detail != "" {
		return fmt.Sprintf("syntax: %d:%d: %s", e.Line, e.Col, e.Detail)
	}
	if len(e.Want) == 0 {
		return fmt.Sprintf("syntax: %d:%d", e.Line, e.Col)
	}
	return fmt.Sprintf("syntax: %d:%d: expected %s", e.Line, e.Col, strings.Join(e.Want, ", "))
}

func (e *Error) Unwrap() error { return errs.ErrSyntax }
