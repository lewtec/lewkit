package header

import "errors"

var (
	errHeader          = errors.New("header required")
	errSyntax          = errors.New("syntax error")
	errType            = errors.New("unknown type")
	errInclude         = errors.New("include not found")
	errLayout          = errors.New("layout")
	errExpr            = errors.New("constant expression")
	errABI             = ErrLayoutID
	errPackage         = errors.New("package name")
	ErrParserFail      = errors.New("")
	ErrParserParseFile = errors.New("")
	ErrRun             = errors.New("")
)
