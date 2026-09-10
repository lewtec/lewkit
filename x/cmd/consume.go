package cmd

import (
	"fmt"
	"reflect"
)

type consumeMode struct {
	allPos   bool
	stopDash bool
	optional bool
}

func isDashType(t reflect.Type) bool {
	return t == reflect.TypeFor[Dash]()
}

func isPosKind(k fieldKind) bool {
	switch k {
	case kindPositional, kindRest, kindProduct, kindArray, kindDash:
		return true
	default:
		return false
	}
}

func isConsumable(t reflect.Type) bool {
	if t == nil {
		return false
	}
	if isDashType(t) {
		return true
	}
	if t.Kind() == reflect.Pointer {
		return isConsumable(t.Elem())
	}
	if (rvalue{reflect.New(t)}).hasParse() {
		return true
	}
	switch t.Kind() {
	case reflect.Slice, reflect.Array:
		return isConsumable(t.Elem())
	case reflect.Struct:
		return isProduct(t)
	default:
		return false
	}
}

func isProduct(t reflect.Type) bool {
	if t.Kind() != reflect.Struct || isDashType(t) {
		return false
	}
	dummy := reflect.New(t).Elem()
	if (rvalue{dummy}).hasParse() || (rvalue{dummy}).hasCount() {
		return false
	}
	n := 0
	for i := range t.NumField() {
		sf := t.Field(i)
		ft := sf.Type
		_, flatten := sf.Tag.Lookup("flatten")
		if (sf.Anonymous || flatten) && (ft.Kind() == reflect.Struct || ft.Kind() == reflect.Pointer && ft.Elem().Kind() == reflect.Struct) {
			elem := ft
			if elem.Kind() == reflect.Pointer {
				elem = elem.Elem()
			}
			if !isProduct(elem) && !isDashType(elem) {
				return false
			}
			n++
			continue
		}
		if sf.Tag.Get("long") != "" || sf.Tag.Get("short") != "" || sf.Tag.Get("cmd") != "" {
			return false
		}
		if !isConsumable(ft) {
			return false
		}
		n++
	}
	return n > 0
}

func structHasCommandBits(t reflect.Type) bool {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return false
	}
	for i := range t.NumField() {
		sf := t.Field(i)
		ft := sf.Type
		if sf.Tag.Get("long") != "" || sf.Tag.Get("short") != "" || sf.Tag.Get("cmd") != "" {
			return true
		}
		_, flatten := sf.Tag.Lookup("flatten")
		if (sf.Anonymous || flatten) && (ft.Kind() == reflect.Struct || ft.Kind() == reflect.Pointer && ft.Elem().Kind() == reflect.Struct) {
			if structHasCommandBits(ft) {
				return true
			}
			continue
		}
		if ft.Kind() == reflect.Pointer && ft.Elem().Kind() == reflect.Struct {
			elem := ft.Elem()
			if (rvalue{reflect.New(elem)}).hasParse() || (rvalue{reflect.New(elem)}).hasCount() {
				continue
			}
			if structHasCommandBits(elem) || !isProduct(elem) {
				return true
			}
		}
	}
	return false
}

func canStart(t reflect.Type, args []string, mode consumeMode) bool {
	if t.Kind() == reflect.Pointer {
		return canStart(t.Elem(), args, mode)
	}
	if isDashType(t) {
		return len(args) > 0 && args[0] == "--"
	}
	if len(args) == 0 {
		return false
	}
	if args[0] == "--" {
		return false
	}
	if !mode.allPos && isOption(args[0]) {
		return false
	}
	return true
}

func consumeValue(rv reflect.Value, args []string, mode consumeMode) (int, error) {
	if rv.Kind() == reflect.Pointer {
		if !canStart(rv.Type().Elem(), args, mode) {
			if mode.optional {
				return 0, nil
			}
			return 0, fmt.Errorf("%w", ErrMissingValue)
		}
		slot := rvalue{rv}.settable()
		if slot.IsNil() {
			slot.Set(reflect.New(slot.Type().Elem()))
		}
		inner := mode
		inner.optional = false
		return consumeValue(slot.Elem(), args, inner)
	}
	if isDashType(rv.Type()) {
		if len(args) == 0 || args[0] != "--" {
			return 0, fmt.Errorf("%w: expected --", ErrMissingValue)
		}
		return 1, nil
	}
	if (rvalue{rv}).hasParse() {
		if !canStart(rv.Type(), args, mode) {
			return 0, fmt.Errorf("%w", ErrMissingValue)
		}
		if err := (rvalue{rv}).parse(args[0]); err != nil {
			return 0, err
		}
		return 1, nil
	}
	switch rv.Kind() {
	case reflect.Struct:
		return consumeProduct(rv, args, mode)
	case reflect.Array:
		return consumeArray(rv, args, mode)
	case reflect.Slice:
		return consumeSlice(rv, args, mode)
	default:
		return 0, fmt.Errorf("%w: cannot consume %s", ErrInvalidSpec, rv.Type())
	}
}

func consumeProduct(rv reflect.Value, args []string, mode consumeMode) (int, error) {
	t := rv.Type()
	off := 0
	for i := range t.NumField() {
		sf := t.Field(i)
		fv := rv.Field(i)
		if !fv.CanAddr() {
			continue
		}
		_, flatten := sf.Tag.Lookup("flatten")
		if (sf.Anonymous || flatten) && shouldFlatten(fv) {
			n, err := consumeProduct(rvalue{fv}.settable(), args[off:], mode)
			if err != nil {
				return off, err
			}
			off += n
			continue
		}
		n, err := consumeValue(rvalue{fv}.settable(), args[off:], mode)
		if err != nil {
			return off, err
		}
		if n == 0 && !isOptionalType(fv.Type()) {
			return off, fmt.Errorf("%w", ErrMissingValue)
		}
		off += n
	}
	return off, nil
}

func isOptionalType(t reflect.Type) bool {
	return t.Kind() == reflect.Pointer
}

func consumeArray(rv reflect.Value, args []string, mode consumeMode) (int, error) {
	off := 0
	for i := 0; i < rv.Len(); i++ {
		n, err := consumeValue(rvalue{rv.Index(i)}.settable(), args[off:], mode)
		if err != nil {
			return off, err
		}
		if n == 0 {
			return off, fmt.Errorf("%w", ErrMissingValue)
		}
		off += n
	}
	return off, nil
}

func consumeSlice(rv reflect.Value, args []string, mode consumeMode) (int, error) {
	slot := rvalue{rv}.settable()
	off := 0
	for off < len(args) {
		if args[off] == "--" && mode.stopDash {
			break
		}
		if !canStart(slot.Type().Elem(), args[off:], mode) {
			break
		}
		elem := reflect.New(slot.Type().Elem()).Elem()
		n, err := consumeValue(elem, args[off:], mode)
		if err != nil {
			return off, err
		}
		if n == 0 {
			break
		}
		slot.Set(reflect.Append(slot, elem))
		off += n
	}
	return off, nil
}
