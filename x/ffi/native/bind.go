package native

import (
	"fmt"
	"reflect"
	"strings"
)

// Bind resolves the exported function fields of T from lib and returns the
// filled struct. Each field carries a sym tag naming the C symbol. A tag
// of the form `sym:"name,soft"` leaves that field nil when the symbol is
// missing. T is a struct, and its function fields are exported so this
// package can set them.
func Bind[T any](lib uintptr) (T, error) {
	var out T
	v := reflect.ValueOf(&out).Elem()
	if v.Kind() != reflect.Struct {
		var zero T
		return zero, fmt.Errorf("native.Bind: %s is not a struct", v.Type())
	}
	rt := v.Type()
	for i := 0; i < rt.NumField(); i++ {
		sf := rt.Field(i)
		tag, ok := sf.Tag.Lookup("sym")
		if !ok {
			if sf.Type.Kind() == reflect.Func {
				var zero T
				return zero, fmt.Errorf("native.Bind: %s.%s missing sym tag", rt.Name(), sf.Name)
			}
			continue
		}
		name, soft, err := parseSym(tag)
		if err != nil {
			var zero T
			return zero, fmt.Errorf("native.Bind: %s.%s: %w", rt.Name(), sf.Name, err)
		}
		if sf.Type.Kind() != reflect.Func || !sf.IsExported() {
			var zero T
			return zero, fmt.Errorf("native.Bind: %s.%s is not an exported function", rt.Name(), sf.Name)
		}
		field := v.Field(i)
		if !field.CanSet() {
			var zero T
			return zero, fmt.Errorf("native.Bind: %s.%s cannot be set", rt.Name(), sf.Name)
		}
		if err := bindExport(lib, name, field.Addr().Interface()); err != nil {
			if soft {
				continue
			}
			var zero T
			return zero, err
		}
	}
	return out, nil
}

func parseSym(tag string) (string, bool, error) {
	name, opt, ok := strings.Cut(tag, ",")
	if name == "" {
		return "", false, fmt.Errorf("empty sym tag")
	}
	if !ok {
		return name, false, nil
	}
	if opt != "soft" {
		return "", false, fmt.Errorf("bad sym tag %q", tag)
	}
	return name, true, nil
}

func bindExport(lib uintptr, name string, fnptr any) error {
	if _, err := Symbol(lib, name); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	Func(lib, name, fnptr)
	return nil
}
