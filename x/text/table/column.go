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

// Column is one field of Row. Format is how the cell shows in every output.
// An empty Format uses the default text. A Format containing % is a fmt
// verb. A time.Time value uses Format as a time layout.
type Column[Row any] struct {
	Name   string
	Value  func(Row) any
	Format string
}

// Formatter returns the columns of Row in display order. Columns must not
// read the receiver. Each Value func receives the row.
type Formatter[Row any] interface {
	Columns() []Column[Row]
}

// Fields is a Formatter made from a slice.
type Fields[Row any] []Column[Row]

func (fields Fields[Row]) Columns() []Column[Row] { return []Column[Row](fields) }

type boundColumn[Row any] struct {
	name   string
	value  func(Row) any
	format string
}

func (column boundColumn[Row]) text(row Row) (string, error) {
	return formatCell(column.value(row), column.format)
}

func (column boundColumn[Row]) json(row Row) (any, error) {
	value := column.value(row)
	if column.format == "" {
		return value, nil
	}
	return formatCell(value, column.format)
}

// Resolve returns layout's columns, or the Columns method on Row, or the
// exported struct fields. A nil layout uses the method, then the fields.
func Resolve[Row any](layout Formatter[Row]) ([]Column[Row], error) {
	if layout != nil {
		return check(layout.Columns())
	}
	var zero Row
	if found, ok := any(zero).(Formatter[Row]); ok {
		return check(found.Columns())
	}
	if found, ok := any(&zero).(Formatter[Row]); ok {
		return check(found.Columns())
	}
	return columnsOf[Row]()
}

// Select keeps spec's columns in that order. spec is empty, or a comma
// separated list of name and name=format. An empty spec keeps columns.
// name=format replaces that column's Format.
func Select[Row any](columns []Column[Row], spec string) ([]Column[Row], error) {
	if strings.TrimSpace(spec) == "" {
		return columns, nil
	}
	byName := make(map[string]Column[Row], len(columns))
	for _, column := range columns {
		if _, ok := byName[column.Name]; ok {
			return nil, fmt.Errorf("%w: duplicate %s", ErrColumn, column.Name)
		}
		byName[column.Name] = column
	}
	var selected []Column[Row]
	seen := map[string]struct{}{}
	for entry := range strings.SplitSeq(spec, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			return nil, fmt.Errorf("%w: empty name", ErrColumn)
		}
		name, format, hasFormat := strings.Cut(entry, "=")
		name = strings.TrimSpace(name)
		if _, ok := seen[name]; ok {
			return nil, fmt.Errorf("%w: duplicate %s", ErrColumn, name)
		}
		seen[name] = struct{}{}
		column, ok := byName[name]
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrColumn, name)
		}
		if hasFormat {
			column.Format = format
		}
		selected = append(selected, column)
	}
	return selected, nil
}

func check[Row any](columns []Column[Row]) ([]Column[Row], error) {
	for _, column := range columns {
		if column.Name == "" || column.Value == nil {
			return nil, fmt.Errorf("%w: %q", ErrColumn, column.Name)
		}
	}
	return columns, nil
}

func bind[Row any](columns []Column[Row]) ([]boundColumn[Row], error) {
	bound, err := check(columns)
	if err != nil {
		return nil, err
	}
	out := make([]boundColumn[Row], len(bound))
	for index, column := range bound {
		out[index] = boundColumn[Row]{name: column.Name, value: column.Value, format: column.Format}
	}
	return out, nil
}

func columnsOf[Row any]() ([]Column[Row], error) {
	rowType := reflect.TypeFor[Row]()
	for rowType.Kind() == reflect.Pointer {
		rowType = rowType.Elem()
	}
	if rowType.Kind() != reflect.Struct {
		one := Column[Row]{Name: "value", Value: func(row Row) any { return row }}
		return []Column[Row]{one}, nil
	}
	var columns []Column[Row]
	for _, field := range reflect.VisibleFields(rowType) {
		if field.Anonymous || !field.IsExported() {
			continue
		}
		name, skip := jsonName(field)
		if skip {
			continue
		}
		fieldIndex := append([]int(nil), field.Index...)
		columns = append(columns, Column[Row]{
			Name:  name,
			Value: func(row Row) any { return jsonAt(reflect.ValueOf(row), fieldIndex) },
		})
	}
	return columns, nil
}

func jsonName(field reflect.StructField) (string, bool) {
	name := field.Name
	tag, ok := field.Tag.Lookup("json")
	if !ok {
		return name, false
	}
	decoded := tag
	if comma := strings.IndexByte(tag, ','); comma >= 0 {
		decoded = tag[:comma]
	}
	switch decoded {
	case "-":
		return "", true
	case "":
		return name, false
	default:
		return decoded, false
	}
}

func formatCell(value any, format string) (string, error) {
	if format == "" {
		return formatAny(value), nil
	}
	if strings.Contains(format, "%") {
		return fmt.Sprintf(format, value), nil
	}
	if instant, ok := value.(time.Time); ok {
		return instant.Format(format), nil
	}
	return "", fmt.Errorf("%w: format %q", ErrColumn, format)
}
