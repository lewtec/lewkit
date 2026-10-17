package makefile

// Kind is the test on a conditional branch.
// A plain else has no kind.
//
//lewkit:enum invalid=none
type Kind uint8

const (
	KindIfEq Kind = iota + 1
	KindIfNeq
	KindIfDef
	KindIfNdef
)
