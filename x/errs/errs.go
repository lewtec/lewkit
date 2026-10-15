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

// Wrap returns an error whose text is text.
// errors.Is matches parent and each extra error, the same as %w.
// text stays the sentence. Parent text is not joined in.
func Wrap(parent error, text string, extra ...error) error {
	parents := make([]error, 0, 1+len(extra))
	if parent != nil {
		parents = append(parents, parent)
	}
	for _, err := range extra {
		if err != nil {
			parents = append(parents, err)
		}
	}
	if len(parents) == 1 {
		return &wrapped{text: text, parent: parents[0]}
	}
	return &wrappedMany{text: text, parents: parents}
}

type wrapped struct {
	text   string
	parent error
}

func (w *wrapped) Error() string { return w.text }

func (w *wrapped) Unwrap() error { return w.parent }

// wrappedMany is several parents with one sentence.
type wrappedMany struct {
	text    string
	parents []error
}

func (w *wrappedMany) Error() string { return w.text }

func (w *wrappedMany) Unwrap() []error { return w.parents }
