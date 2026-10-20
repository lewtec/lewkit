package androiddex

import "github.com/lewtec/lewkit/x/errs"

var (
	// ErrPackage means the working directory is not the Android host package.
	ErrPackage = errs.Wrap(errs.ErrNotFound, "android host package")
	// ErrResource means a resids.go constant and its Java field do not match.
	ErrResource = errs.Wrap(errs.ErrInvalid, "android resource id")
	// ErrTool means javac, d8, or android.jar is not already on the machine.
	ErrTool = errs.Wrap(errs.ErrNotFound, "android dex tool")
	// ErrCompile means javac or d8 did not produce one classes.dex.
	ErrCompile = errs.Wrap(errs.ErrInvalid, "android dex compile")
	// ErrDependency means an androidx artifact could not be read.
	ErrDependency = errs.Wrap(errs.ErrUnavailable, "android dex dependency")

	// errMissingEntry is a zip member the artifact does not carry.
	// R.txt is optional. classes.jar is not.
	errMissingEntry = errs.Wrap(errs.ErrNotFound, "zip entry")
)
