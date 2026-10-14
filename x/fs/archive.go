package fs

import (
	"github.com/lewtec/lewkit/x/errs"
)

// ErrEmptyArchive means the archive has no members.
var ErrEmptyArchive = errs.Wrap(errs.ErrEmpty, "empty archive")
