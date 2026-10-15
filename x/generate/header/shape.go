package header

// Shape is the class of a C type. The values match the parser's kind
// constants so a generator outside this package can switch on a type
// without naming that unexported kind.
//
//lewkit:enum
type Shape uint8

const (
	ShapeInvalid Shape = iota
	ShapeVoid
	ShapeBool
	ShapeSInt
	ShapeUInt
	ShapeFloat
	ShapePtr
	ShapeArray
	ShapeFunc
	ShapeStruct
	ShapeUnion
	ShapeEnum
	ShapeAlias
)

// ShapeOf reports the class of t. A nil type is ShapeInvalid.
func ShapeOf(t *Type) Shape {
	if t == nil {
		return ShapeInvalid
	}
	return Shape(t.Kind)
}
