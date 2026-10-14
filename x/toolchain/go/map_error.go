package gotool

import (
	"github.com/lewtec/lewkit/x/errs"
)

// ErrJitExecutableMappingIsUnavailable means the OS refused an executable mapping.
var ErrJitExecutableMappingIsUnavailable = errs.Wrap(errs.ErrUnavailable, "jit: executable mapping is unavailable")
