package makefile

// Op is the operator on a variable or a multi-line function.
// The zero value is a define with no operator.
//
//lewkit:enum invalid=none
type Op uint8

const (
	// OpRecursive is =.
	OpRecursive Op = iota + 1
	// OpSimple is :=.
	OpSimple
	// OpImmediate is ::=.
	OpImmediate
	// OpShell is !=.
	OpShell
	// OpAppend is +=.
	OpAppend
	// OpConditional is ?=.
	OpConditional
)

func (op Op) symbol() (string, bool) {
	switch op {
	case OpRecursive:
		return "=", true
	case OpSimple:
		return ":=", true
	case OpImmediate:
		return "::=", true
	case OpShell:
		return "!=", true
	case OpAppend:
		return "+=", true
	case OpConditional:
		return "?=", true
	default:
		return "", false
	}
}
