package header

import "errors"

var (
	errHeader  = errors.New("header required")
	errSyntax  = errors.New("syntax error")
	errType    = errors.New("unknown type")
	errInclude = errors.New("include not found")
	errLayout  = errors.New("layout")
	errExpr    = errors.New("constant expression")
	errABI     = errors.New("unknown abi")
	errPackage = errors.New("package name")
)
