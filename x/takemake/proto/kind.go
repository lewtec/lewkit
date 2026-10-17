package proto

// Label is the cardinality keyword printed before a field type.
// The zero value prints nothing.
//
//lewkit:enum invalid=none
type Label uint8

const (
	LabelOptional Label = iota + 1
	LabelRequired
	LabelRepeated
)

// ImportKind is the modifier on an import line.
// The zero value is a plain import.
//
//lewkit:enum invalid=plain
type ImportKind uint8

const (
	ImportPublic ImportKind = iota + 1 // public
	ImportWeak                         // weak
)

// Kind is the shape of an option constant.
//
//lewkit:enum invalid=unknown
type Kind uint8

const (
	KindString Kind = iota + 1
	KindBool
	KindInt
	KindFloat
	KindIdent
	KindAggregate
	KindList
)
