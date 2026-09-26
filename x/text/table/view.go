package table

import (
	"fmt"
	"reflect"
)

// Field is one column in a spec struct passed to Make. The type argument
// matches the row field of the same name. Name overrides that field's header.
// Format is the default cell format. An empty Name keeps the json name, then
// the Go name.
type Field[V any] struct {
	Name   string
	Format string
	kind   V
}

// View is a Formatter built once. Pass it to Write or cmd.Rows.
type View[T any] struct {
	cols []Column[T]
}

func (v View[T]) Columns() []Column[T] { return v.cols }

// Make builds a View of Row from spec. Spec field order is the column order.
// Each exported spec field is a Field whose type matches the same-named Row
// field. Reflection runs here, not while rendering.
func Make[Row any, Spec any](spec Spec) (View[Row], error) {
	sv := reflect.ValueOf(spec)
	st := sv.Type()
	if st.Kind() == reflect.Pointer {
		st = st.Elem()
		if sv.IsNil() {
			sv = reflect.New(st).Elem()
		} else {
			sv = sv.Elem()
		}
	}
	if st.Kind() != reflect.Struct {
		return View[Row]{}, fmt.Errorf("%w: spec %s is not a struct", ErrColumn, st)
	}
	rt := reflect.TypeFor[Row]()
	for rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
	}
	if rt.Kind() != reflect.Struct {
		return View[Row]{}, fmt.Errorf("%w: row %s is not a struct", ErrColumn, rt)
	}
	var cols []Column[Row]
	seen := map[string]struct{}{}
	for i := range st.NumField() {
		sf := st.Field(i)
		if sf.Anonymous || !sf.IsExported() {
			continue
		}
		kind := sv.Field(i).FieldByName("kind")
		if !kind.IsValid() {
			return View[Row]{}, fmt.Errorf("%w: %s.%s is not a Field", ErrColumn, st.Name(), sf.Name)
		}
		rf, ok := rt.FieldByName(sf.Name)
		if !ok || !rf.IsExported() {
			return View[Row]{}, fmt.Errorf("%w: %s has no %s", ErrColumn, rt.Name(), sf.Name)
		}
		if rf.Type != kind.Type() {
			return View[Row]{}, fmt.Errorf("%w: %s.%s is %s, spec has %s", ErrColumn, rt.Name(), sf.Name, rf.Type, kind.Type())
		}
		header := sv.Field(i).FieldByName("Name").String()
		if header == "" {
			name, skip := jsonName(rf)
			if skip {
				header = sf.Name
			} else {
				header = name
			}
		}
		if _, ok := seen[header]; ok {
			return View[Row]{}, fmt.Errorf("%w: duplicate %s", ErrColumn, header)
		}
		seen[header] = struct{}{}
		idx := append([]int(nil), rf.Index...)
		cols = append(cols, Column[Row]{
			Name:   header,
			Format: sv.Field(i).FieldByName("Format").String(),
			Value:  func(row Row) any { return jsonAt(reflect.ValueOf(row), idx) },
		})
	}
	if len(cols) == 0 {
		return View[Row]{}, fmt.Errorf("%w: %s has no columns", ErrColumn, st.Name())
	}
	return View[Row]{cols: cols}, nil
}

// Must panics when Make fails. A bad spec is a programmer error.
func Must[T any](view View[T], err error) View[T] {
	if err != nil {
		panic(err)
	}
	return view
}
