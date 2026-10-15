package schema

import "github.com/lewtec/lewkit/x/errs"

// errScaffold is [Database.Apply] and [Database.Make] until the catalog exists.
var errScaffold = errs.Wrap(errs.ErrUnsupported, "schema scaffold")
