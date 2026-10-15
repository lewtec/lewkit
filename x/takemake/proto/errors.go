package proto

import "github.com/lewtec/lewkit/x/errs"

// errScaffold is [Take] and [Make] until this package has a model.
var errScaffold = errs.Wrap(errs.ErrUnsupported, "proto scaffold")
