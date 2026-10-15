package proto

// Value is one option constant.
// The kind selects which field holds the constant.
type Value struct {
	Kind    Kind
	Text    string
	Int     int64
	Float   float64
	Bool    bool
	Entries []Option
	List    []Value
}

// String returns a quoted string constant.
func String(text string) Value {
	return Value{Kind: KindString, Text: text}
}

// Bool returns a true or false constant.
func Bool(value bool) Value {
	return Value{Kind: KindBool, Bool: value}
}

// Int returns an integer constant.
func Int(value int64) Value {
	return Value{Kind: KindInt, Int: value}
}

// Float returns a floating-point constant.
func Float(value float64) Value {
	return Value{Kind: KindFloat, Float: value}
}

// Ident returns an identifier constant, such as SPEED.
func Ident(name string) Value {
	return Value{Kind: KindIdent, Text: name}
}

// Aggregate returns a { name: value } constant.
func Aggregate(entries ...Option) Value {
	return Value{Kind: KindAggregate, Entries: entries}
}

// List returns a [value, value] constant.
func List(values ...Value) Value {
	return Value{Kind: KindList, List: values}
}
