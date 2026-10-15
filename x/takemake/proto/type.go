package proto

// Scalar field types. Named and map types are built with [Named] and [Map].
var (
	TypeDouble   = Named("double")
	TypeFloat    = Named("float")
	TypeInt32    = Named("int32")
	TypeInt64    = Named("int64")
	TypeUint32   = Named("uint32")
	TypeUint64   = Named("uint64")
	TypeSint32   = Named("sint32")
	TypeSint64   = Named("sint64")
	TypeFixed32  = Named("fixed32")
	TypeFixed64  = Named("fixed64")
	TypeSfixed32 = Named("sfixed32")
	TypeSfixed64 = Named("sfixed64")
	TypeBool     = Named("bool")
	TypeString   = Named("string")
	TypeBytes    = Named("bytes")
)

// Type is a field type. Name is a scalar or a named type.
// Key and Elem are set together for a map.
type Type struct {
	Name string
	Key  string
	Elem string
}

// Named returns a scalar or message type.
func Named(name string) Type {
	return Type{Name: name}
}

// Map returns map<key, elem>.
func Map(key, elem Type) Type {
	return Type{Key: key.String(), Elem: elem.String()}
}

// String is the type as it is printed in a proto file.
func (typ Type) String() string {
	if typ.Key != "" || typ.Elem != "" {
		return "map<" + typ.Key + ", " + typ.Elem + ">"
	}
	return typ.Name
}
