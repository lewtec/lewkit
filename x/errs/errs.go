// Package errs holds the meta sentinels shared across the module.
// A specific sentinel wraps the parent that matches its meaning so
// errors.Is(err, ErrNotFound) matches every not-found error. Prefer one
// of these before adding a new sentinel. The same condition is one sentinel.
package errs

import "errors"

// Meta sentinels. The text is the general name of the condition.
var (
	ErrNotFound      = errors.New("not found")
	ErrInvalid       = errors.New("invalid")
	ErrMissing       = errors.New("missing")
	ErrUnsupported   = errors.New("unsupported")
	ErrAlreadyExists = errors.New("already exists")
	ErrUnavailable   = errors.New("unavailable")
	ErrClosed        = errors.New("closed")
	ErrCanceled      = errors.New("canceled")
	ErrEmpty         = errors.New("empty")
	ErrNilContext    = errors.New("nil context")
	ErrSyntax        = errors.New("syntax")
	ErrCycle         = errors.New("cycle")
	ErrDuplicate     = errors.New("duplicate")
	ErrOutOfRange    = errors.New("out of range")
	ErrRequired      = errors.New("required")
	ErrUnknown       = errors.New("unknown")
)

// Wrap returns an error whose text is text and that unwraps to parent.
// errors.Is matches parent. text stays the specific sentence.
func Wrap(parent error, text string) error {
	return &wrapped{text: text, parent: parent}
}

type wrapped struct {
	text   string
	parent error
}

func (w *wrapped) Error() string { return w.text }

func (w *wrapped) Unwrap() error { return w.parent }
