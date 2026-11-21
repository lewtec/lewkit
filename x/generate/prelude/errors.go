package prelude

import (
	"errors"

	"github.com/lewtec/lewkit/x/generate"
)

var (
	errDirRequired = generate.ErrDirRequired
	errNoGoMod     = generate.ErrNoGoMod
	errNoPackage   = errors.New("no package clause")
)
