package table

import (
	"fmt"
	"reflect"
)

// Field is one column in a spec struct passed to Make. The type argument
// matches the row field of the same name. Name overrides that field's header.
// Format is the default cell format. An empty Name keeps the json name, then
// the Go name.
type Field[Value any] struct {
	Name   string
	Format string
	value  Value
}

// View is a Formatter built once. Pass it to Write or cmd.Rows.
type View[Row any] struct {
	columns []Column[Row]
}

func (view View[Row]) Columns() []Column[Row] { return view.columns }

// Make builds a View of Row from spec. Spec field order is the column order.
// Each exported spec field is a Field whose type matches the same-named Row
// field. Reflection runs here, not while rendering.
func Make[Row any, Spec any](spec Spec) (View[Row], error) {
	specValue := reflect.ValueOf(spec)
	specType := specValue.Type()
	if specType.Kind() == reflect.Pointer {
		specType = specType.Elem()
		if specValue.IsNil() {
			specValue = reflect.New(specType).Elem()
		} else {
			specValue = specValue.Elem()
		}
	}
	if specType.Kind() != reflect.Struct {
		return View[Row]{}, fmt.Errorf("%w: spec %s is not a struct", ErrColumn, specType)
	}
	rowType := reflect.TypeFor[Row]()
	for rowType.Kind() == reflect.Pointer {
		rowType = rowType.Elem()
	}
	if rowType.Kind() != reflect.Struct {
		return View[Row]{}, fmt.Errorf("%w: row %s is not a struct", ErrColumn, rowType)
	}
	var columns []Column[Row]
	seen := map[string]struct{}{}
	for index := range specType.NumField() {
		specField := specType.Field(index)
		if specField.Anonymous || !specField.IsExported() {
			continue
		}
		kind := specValue.Field(index).FieldByName("value")
		if !kind.IsValid() {
			return View[Row]{}, fmt.Errorf("%w: %s.%s is not a Field", ErrColumn, specType.Name(), specField.Name)
		}
		rowField, ok := rowType.FieldByName(specField.Name)
		if !ok || !rowField.IsExported() {
			return View[Row]{}, fmt.Errorf("%w: %s has no %s", ErrColumn, rowType.Name(), specField.Name)
		}
		if rowField.Type != kind.Type() {
			return View[Row]{}, fmt.Errorf("%w: %s.%s is %s, spec has %s", ErrColumn, rowType.Name(), specField.Name, rowField.Type, kind.Type())
		}
		header := specValue.Field(index).FieldByName("Name").String()
		if header == "" {
			name, skip := jsonName(rowField)
			if skip {
				header = specField.Name
			} else {
				header = name
			}
		}
		if _, ok := seen[header]; ok {
			return View[Row]{}, fmt.Errorf("%w: duplicate %s", ErrColumn, header)
		}
		seen[header] = struct{}{}
		fieldIndex := append([]int(nil), rowField.Index...)
		columns = append(columns, Column[Row]{
			Name:   header,
			Format: specValue.Field(index).FieldByName("Format").String(),
			Value:  func(row Row) any { return jsonAt(reflect.ValueOf(row), fieldIndex) },
		})
	}
	if len(columns) == 0 {
		return View[Row]{}, fmt.Errorf("%w: %s has no columns", ErrColumn, specType.Name())
	}
	return View[Row]{columns: columns}, nil
}

// Must panics when Make fails. A bad spec is a programmer error.
func Must[Row any](view View[Row], err error) View[Row] {
	if err != nil {
		panic(err)
	}
	return view
}
