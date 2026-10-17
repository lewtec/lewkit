package graphviz

import "github.com/lewtec/lewkit/x/errs"

var (
	// ErrSyntax means the bytes are not Graphviz source this package reads.
	ErrSyntax = errs.Wrap(errs.ErrSyntax, "graphviz syntax")
	// ErrRead means the reader is missing or failed.
	ErrRead = errs.Wrap(errs.ErrInvalid, "graphviz read")
	// ErrWrite means the writer is missing or failed.
	ErrWrite = errs.Wrap(errs.ErrMissing, "nil writer")
	// ErrFile means the file is missing.
	ErrFile = errs.Wrap(errs.ErrMissing, "nil file")
	// ErrForm means a model this package does not print.
	ErrForm = errs.Wrap(errs.ErrUnsupported, "unsupported graphviz")
)
