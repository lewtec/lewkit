package sarif

import "github.com/lewtec/lewkit/x/errs"

var (
	// ErrValue means a file, a reader, a writer, or a child is missing.
	ErrValue = errs.Wrap(errs.ErrInvalid, "sarif value")
	// ErrSyntax means the bytes are not a SARIF log this package reads.
	ErrSyntax = errs.Wrap(errs.ErrSyntax, "sarif syntax")
	// ErrForm means a version or property this package does not write.
	ErrForm = errs.Wrap(errs.ErrUnsupported, "unsupported sarif")
)
