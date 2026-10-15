package golang

import (
	"strconv"
	"strings"
)

// Expr is one expression. The kind selects the fields in use.
//
//	Ident      Name
//	Selector   Path and Name for a package qualifier, or X and Name
//	Call       Fun, Args, Ellipsis
//	Lit        Lit, Text
//	Composite  Type, Elts
//	Unary      Op, X
//	Binary     Op, X, Y
//	Index      X, At
//	IndexList  X, Indexes
//	Slice      X, Low, High, Max
//	Paren      X
//	Type       Type
//	Func       Fn
//	Assert     X, Type
type Expr struct {
	Kind     ExprKind
	Name     string
	Path     string
	Text     string
	Lit      LitKind
	Op       Op
	Ellipsis bool
	X        *Expr
	Y        *Expr
	Fun      *Expr
	Args     []*Expr
	At       *Expr
	Indexes  []*Expr
	Low      *Expr
	High     *Expr
	Max      *Expr
	Type     *Type
	Elts     []*Element
	Fn       *Func
}

// Element is one composite literal element. Key is nil when the
// element is positional.
type Element struct {
	Key   *Expr
	Value *Expr
}

func (*Expr) code() {}

// Id returns an identifier.
func Id(name string) *Expr {
	return &Expr{Kind: ExprIdent, Name: name}
}

// Qual returns a package-qualified name. path is the import path.
func Qual(path, name string) *Expr {
	return &Expr{Kind: ExprSelector, Path: path, Name: name}
}

// Lit returns a literal. It accepts nil, bool, string, integer, and
// float values. The text is the Go spelling, so a string includes
// its quotes.
func Lit(value any) *Expr {
	if value == nil {
		return &Expr{Kind: ExprLit, Lit: LitNil, Text: "nil"}
	}
	switch v := value.(type) {
	case bool:
		text := "false"
		if v {
			text = "true"
		}
		return &Expr{Kind: ExprLit, Lit: LitBool, Text: text}
	case string:
		return &Expr{Kind: ExprLit, Lit: LitString, Text: strconv.Quote(v)}
	case int:
		return intLit(int64(v))
	case int8:
		return intLit(int64(v))
	case int16:
		return intLit(int64(v))
	case int32:
		return intLit(int64(v))
	case int64:
		return intLit(v)
	case uint:
		return uintLit(uint64(v))
	case uint8:
		return uintLit(uint64(v))
	case uint16:
		return uintLit(uint64(v))
	case uint32:
		return uintLit(uint64(v))
	case uint64:
		return uintLit(v)
	case float32:
		return floatLit(float64(v))
	case float64:
		return floatLit(v)
	default:
		return &Expr{Kind: ExprLit, Lit: LitInt, Text: strconv.FormatInt(0, 10)}
	}
}

func intLit(v int64) *Expr {
	return &Expr{Kind: ExprLit, Lit: LitInt, Text: strconv.FormatInt(v, 10)}
}

func uintLit(v uint64) *Expr {
	return &Expr{Kind: ExprLit, Lit: LitInt, Text: strconv.FormatUint(v, 10)}
}

func floatLit(v float64) *Expr {
	text := strconv.FormatFloat(v, 'g', -1, 64)
	if !strings.ContainsAny(text, ".eE") {
		text += ".0"
	}
	return &Expr{Kind: ExprLit, Lit: LitFloat, Text: text}
}

// Sel returns the selector e.name.
func (e *Expr) Sel(name string) *Expr {
	return &Expr{Kind: ExprSelector, X: e, Name: name}
}

// Call returns a call of e. Ellipsis is set later with [Expr.Dots]
// when the last argument is unpacked.
func (e *Expr) Call(args ...*Expr) *Expr {
	if args == nil {
		args = []*Expr{}
	}
	return &Expr{Kind: ExprCall, Fun: e, Args: args}
}

// Dots marks the call as unpacking its last argument.
func (e *Expr) Dots() *Expr {
	if e != nil {
		e.Ellipsis = true
	}
	return e
}

// Binary returns x op y.
func Binary(op Op, x, y *Expr) *Expr {
	return &Expr{Kind: ExprBinary, Op: op, X: x, Y: y}
}

// Unary returns op x.
func Unary(op Op, x *Expr) *Expr {
	return &Expr{Kind: ExprUnary, Op: op, X: x}
}

// Composite returns a composite literal. typ may be nil for a nested
// element whose type is implied.
func Composite(typ *Type, elts ...*Element) *Expr {
	if elts == nil {
		elts = []*Element{}
	}
	return &Expr{Kind: ExprComposite, Type: typ, Elts: elts}
}

// KeyValue returns a keyed composite element.
func KeyValue(key, value *Expr) *Element {
	return &Element{Key: key, Value: value}
}

// Elem returns a positional composite element.
func Elem(value *Expr) *Element {
	return &Element{Value: value}
}

// Index returns e[index].
func (e *Expr) Index(index *Expr) *Expr {
	return &Expr{Kind: ExprIndex, X: e, At: index}
}

// Slice returns a slice expression. A nil bound is omitted.
func (e *Expr) Slice(low, high, max *Expr) *Expr {
	return &Expr{Kind: ExprSlice, X: e, Low: low, High: high, Max: max}
}

// Assert returns e.(typ).
func (e *Expr) Assert(typ *Type) *Expr {
	return &Expr{Kind: ExprAssert, X: e, Type: typ}
}

// Paren returns a parenthesized expression.
func Paren(x *Expr) *Expr {
	return &Expr{Kind: ExprParen, X: x}
}

// FuncLit returns a function literal.
func FuncLit(params, results []*Field, body ...Code) *Expr {
	if params == nil {
		params = []*Field{}
	}
	if results == nil {
		results = []*Field{}
	}
	return &Expr{Kind: ExprFunc, Fn: &Func{
		Parameters: params,
		Returns:    results,
		HasBody:    true,
		Body:       statements(body),
	}}
}
