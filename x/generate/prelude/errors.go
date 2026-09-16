package prelude

import "errors"

var (
	errDirRequired = errors.New("directory required")
	errNoModule    = errors.New("go.mod: no module line")
	errNoGoMod     = errors.New("no go.mod")
	errNoPackage   = errors.New("no package clause")
)
