package header

// kind is the C type's shape after typedefs are followed for layout.
type kind uint8

const (
	kindInvalid kind = iota
	kindVoid
	kindBool
	kindSInt
	kindUInt
	kindFloat
	kindPtr
	kindArray
	kindFunc
	kindStruct
	kindUnion
	kindEnum
	kindAlias
)

// Type is one C type. Struct, union, and enum values are shared by tag
// so a later definition fills the forward declaration.
type Type struct {
	Kind       kind
	Size       int
	Align      int
	Elem       *Type
	Len        int
	Fields     []*Field
	Params     []Param
	Variadic   bool
	Enums      []Enumerator
	Tag        string
	Spell      string
	prefer     string
	Go         string
	Packed     bool
	AlignAttr  int
	Incomplete bool
	listed     bool
	laid       bool
	laying     bool
	natKnown   bool
	nat        bool
}

// Field is one struct or union member.
type Field struct {
	Name      string
	Go        string
	Type      *Type
	Offset    int
	IsBit     bool
	BitWidth  int
	BitOff    int
	Container int
	CSize     int
	AlignAttr int
	Packed    bool
}

// Enumerator is one enum constant.
type Enumerator struct {
	Name  string
	Go    string
	Value int64
}

// Param is one function parameter after array decay.
type Param struct {
	Name string
	Type *Type
}

// Func is one C function that Bind resolves.
type Func struct {
	Name   string
	Go     string
	Ret    *Type
	Params []Param
}

// Object is one C object. The generated accessor returns its address.
type Object struct {
	Name string
	Go   string
	Type *Type
}

// Const is one integer macro.
type Const struct {
	Name  string
	Go    string
	Value int64
}

// Unit is the binding model for one header and the includes it pulled in.
type Unit struct {
	Target  Target
	Types   []*Type
	Consts  []Const
	Funcs   []Func
	Objects []Object
	Skipped []string
}

func (t *Type) flex() bool {
	return t != nil && t.Kind == kindArray && t.Len < 0
}
