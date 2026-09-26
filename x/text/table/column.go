package table

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// ErrColumn means a column name, value, or format is missing or unknown.
var ErrColumn = errors.New("bad column")

// Column is one field of T. Format is how the cell shows in every output.
// An empty Format uses the default text. A Format containing % is a fmt
// verb. A time.Time value uses Format as a time layout.
type Column[T any] struct {
	Name   string
	Value  func(T) any
	Format string
}

// Formatter returns the columns of T in display order. Columns must not
// read the receiver. Each Value func receives the row.
type Formatter[T any] interface {
	Columns() []Column[T]
}

// Fields is a Formatter made from a slice.
type Fields[T any] []Column[T]

func (f Fields[T]) Columns() []Column[T] { return []Column[T](f) }

type col[T any] struct {
	name   string
	value  func(T) any
	format string
}

func (c col[T]) text(row T) (string, error) {
	return formatCell(c.value(row), c.format)
}

func (c col[T]) json(row T) (any, error) {
	value := c.value(row)
	if c.format == "" {
		return value, nil
	}
	return formatCell(value, c.format)
}

// Resolve returns layout's columns, or the Columns method on T, or the
// exported struct fields. A nil layout uses the method, then the fields.
func Resolve[T any](layout Formatter[T]) ([]Column[T], error) {
	if layout != nil {
		return check(layout.Columns())
	}
	var zero T
	if found, ok := any(zero).(Formatter[T]); ok {
		return check(found.Columns())
	}
	if found, ok := any(&zero).(Formatter[T]); ok {
		return check(found.Columns())
	}
	return columnsOf[T]()
}

// Select keeps spec's columns in that order. spec is empty, or a comma
// separated list of name and name=format. An empty spec keeps cols.
// name=format replaces that column's Format.
func Select[T any](cols []Column[T], spec string) ([]Column[T], error) {
	if strings.TrimSpace(spec) == "" {
		return cols, nil
	}
	by := make(map[string]Column[T], len(cols))
	for _, c := range cols {
		if _, ok := by[c.Name]; ok {
			return nil, fmt.Errorf("%w: duplicate %s", ErrColumn, c.Name)
		}
		by[c.Name] = c
	}
	var out []Column[T]
	seen := map[string]struct{}{}
	for part := range strings.SplitSeq(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("%w: empty name", ErrColumn)
		}
		name, format, hasFormat := strings.Cut(part, "=")
		name = strings.TrimSpace(name)
		if _, ok := seen[name]; ok {
			return nil, fmt.Errorf("%w: duplicate %s", ErrColumn, name)
		}
		seen[name] = struct{}{}
		c, ok := by[name]
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrColumn, name)
		}
		if hasFormat {
			c.Format = format
		}
		out = append(out, c)
	}
	return out, nil
}

func check[T any](cols []Column[T]) ([]Column[T], error) {
	for _, c := range cols {
		if c.Name == "" || c.Value == nil {
			return nil, fmt.Errorf("%w: %q", ErrColumn, c.Name)
		}
	}
	return cols, nil
}

func bind[T any](cols []Column[T]) ([]col[T], error) {
	out := make([]col[T], len(cols))
	for i, c := range cols {
		if c.Name == "" || c.Value == nil {
			return nil, fmt.Errorf("%w: %q", ErrColumn, c.Name)
		}
		out[i] = col[T]{name: c.Name, value: c.Value, format: c.Format}
	}
	return out, nil
}

func columnsOf[T any]() ([]Column[T], error) {
	t := reflect.TypeFor[T]()
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		one := Column[T]{Name: "value", Value: func(row T) any { return row }}
		return []Column[T]{one}, nil
	}
	var cols []Column[T]
	for _, field := range reflect.VisibleFields(t) {
		if field.Anonymous || !field.IsExported() {
			continue
		}
		name, skip := jsonName(field)
		if skip {
			continue
		}
		idx := append([]int(nil), field.Index...)
		cols = append(cols, Column[T]{
			Name:  name,
			Value: func(row T) any { return jsonAt(reflect.ValueOf(row), idx) },
		})
	}
	return cols, nil
}

func jsonName(field reflect.StructField) (string, bool) {
	name := field.Name
	tag, ok := field.Tag.Lookup("json")
	if !ok {
		return name, false
	}
	jname := tag
	if comma := strings.IndexByte(tag, ','); comma >= 0 {
		jname = tag[:comma]
	}
	switch jname {
	case "-":
		return "", true
	case "":
		return name, false
	default:
		return jname, false
	}
}

func formatCell(v any, format string) (string, error) {
	if format == "" {
		return formatAny(v), nil
	}
	if strings.Contains(format, "%") {
		return fmt.Sprintf(format, v), nil
	}
	if stamp, ok := v.(time.Time); ok {
		return stamp.Format(format), nil
	}
	return "", fmt.Errorf("%w: format %q", ErrColumn, format)
}
