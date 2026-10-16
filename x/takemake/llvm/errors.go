package llvm

import "github.com/lewtec/lewkit/x/errs"

var (
	ErrNilReader   = errs.Wrap(errs.ErrInvalid, "llvm: nil reader")
	ErrNilWriter   = errs.Wrap(errs.ErrInvalid, "llvm: nil writer")
	ErrNilFile     = errs.Wrap(errs.ErrInvalid, "llvm: nil file")
	ErrRead        = errs.Wrap(errs.ErrInvalid, "llvm: read")
	ErrModule      = errs.Wrap(errs.ErrInvalid, "llvm: invalid module")
	ErrSyntax      = errs.Wrap(errs.ErrSyntax, "llvm: syntax")
	ErrUnsupported = errs.Wrap(errs.ErrUnsupported, "llvm: unsupported")
)
