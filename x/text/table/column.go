package table

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// ErrColumn means a column has no name, no value, or a bad table tag.
var ErrColumn = errors.New("bad column")

// Column is one field of T. Write prints columns in slice order for every format.
// Text is the table and CSV cell. JSON is the JSONL value.
// A nil Text uses the JSON value. A nil JSON uses the text.
type Column[T any] struct {
	Name string
	Text func(T) string
	JSON func(T) any
}

type col[T any] struct {
	name string
	text func(T) string
	json func(T) any
}

type ranked[T any] struct {
	col   col[T]
	order int
	pos   int
}

func bind[T any](c Column[T]) (col[T], error) {
	if c.Name == "" || (c.Text == nil && c.JSON == nil) {
		return col[T]{}, fmt.Errorf("%w: %q", ErrColumn, c.Name)
	}
	text, jv := c.Text, c.JSON
	if text == nil {
		get := jv
		text = func(row T) string { return formatAny(get(row)) }
	}
	if jv == nil {
		get := text
		jv = func(row T) any { return get(row) }
	}
	return col[T]{name: c.Name, text: text, json: jv}, nil
}

func useColumns[T any](cols []Column[T]) ([]col[T], error) {
	if len(cols) == 0 {
		return columnsOf[T]()
	}
	out := make([]col[T], len(cols))
	for i, c := range cols {
		bound, err := bind(c)
		if err != nil {
			return nil, err
		}
		out[i] = bound
	}
	return out, nil
}

func columnsOf[T any]() ([]col[T], error) {
	t := reflect.TypeFor[T]()
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		one, err := bind(Column[T]{
			Name: "value",
			Text: func(row T) string { return formatAny(row) },
			JSON: func(row T) any { return row },
		})
		if err != nil {
			return nil, err
		}
		return []col[T]{one}, nil
	}
	var cols []ranked[T]
	for pos, field := range reflect.VisibleFields(t) {
		if field.Anonymous || !field.IsExported() {
			continue
		}
		name, order, skip, err := columnMeta(field, pos)
		if err != nil {
			return nil, err
		}
		if skip {
			continue
		}
		idx := append([]int(nil), field.Index...)
		bound, err := bind(Column[T]{
			Name: name,
			Text: func(row T) string { return cellAt(reflect.ValueOf(row), idx) },
			JSON: func(row T) any { return jsonAt(reflect.ValueOf(row), idx) },
		})
		if err != nil {
			return nil, err
		}
		cols = append(cols, ranked[T]{col: bound, order: order, pos: pos})
	}
	sort.SliceStable(cols, func(i, j int) bool {
		if cols[i].order != cols[j].order {
			return cols[i].order < cols[j].order
		}
		return cols[i].pos < cols[j].pos
	})
	out := make([]col[T], len(cols))
	for i, c := range cols {
		out[i] = c.col
	}
	return out, nil
}

// columnMeta reads the table and json tags.
// table:"-" skips the field. table:"name,order=N" sets the header and sort key.
// The default key is the field index, so a smaller order comes first.
// A json name is the header when table does not set one. json:"-" skips the field
// unless a table tag names it or reorders it.
func columnMeta(field reflect.StructField, pos int) (name string, order int, skip bool, err error) {
	name = field.Name
	order = pos
	jsonSkip := false
	if tag, ok := field.Tag.Lookup("json"); ok {
		jname := tag
		if comma := strings.IndexByte(tag, ','); comma >= 0 {
			jname = tag[:comma]
		}
		switch jname {
		case "-":
			jsonSkip = true
		case "":
		default:
			name = jname
		}
	}
	tag, ok := field.Tag.Lookup("table")
	if !ok {
		return name, order, jsonSkip, nil
	}
	tname, opts := splitTag(tag)
	if tname == "-" {
		return "", 0, true, nil
	}
	if tname != "" {
		name = tname
	} else if jsonSkip {
		name = field.Name
	}
	for _, opt := range opts {
		n, ok, perr := parseOrder(opt)
		if perr != nil {
			return "", 0, false, fmt.Errorf("%w: %s %s", ErrColumn, field.Name, perr.Error())
		}
		if !ok {
			return "", 0, false, fmt.Errorf("%w: %s has %q", ErrColumn, field.Name, opt)
		}
		order = n
	}
	return name, order, false, nil
}

func splitTag(tag string) (name string, opts []string) {
	parts := strings.Split(tag, ",")
	if len(parts) == 0 {
		return "", nil
	}
	if !strings.Contains(parts[0], "=") {
		return parts[0], parts[1:]
	}
	return "", parts
}

func parseOrder(opt string) (int, bool, error) {
	key, val, ok := strings.Cut(opt, "=")
	if !ok || key != "order" {
		return 0, false, nil
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return 0, false, err
	}
	return n, true, nil
}
