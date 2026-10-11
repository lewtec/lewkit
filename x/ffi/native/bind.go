package native

import (
	"fmt"
	"reflect"
	"strings"
)

// Bind loads the first library in names and returns T filled from it.
// names are sonames or paths, searched the same way as OpenChain.
// flags 0 means Lazy. Each exported func field of T carries a sym tag
// naming the C symbol. A tag of the form `sym:"name,soft"` leaves that
// field nil when the symbol is missing.
func Bind[T any](flags int, names ...string) (T, error) {
	return BindIn[T](flags, nil, names...)
}

// BindIn is Bind with extra directories searched before the default chain.
func BindIn[T any](flags int, extra []string, names ...string) (T, error) {
	var zero T
	if err := checkBind(reflect.TypeOf(&zero).Elem()); err != nil {
		return zero, err
	}
	lib, err := OpenIn(flags, extra, names...)
	if err != nil {
		return zero, err
	}
	if err := fillBind(reflect.ValueOf(&zero).Elem(), lib); err != nil {
		var fail T
		return fail, err
	}
	return zero, nil
}

func checkBind(rt reflect.Type) error {
	if rt.Kind() != reflect.Struct {
		return fmt.Errorf("native.Bind: %s is not a struct", rt)
	}
	for i := 0; i < rt.NumField(); i++ {
		sf := rt.Field(i)
		tag, ok := sf.Tag.Lookup("sym")
		if !ok {
			if sf.Type.Kind() == reflect.Func {
				return fmt.Errorf("native.Bind: %s.%s missing sym tag", rt.Name(), sf.Name)
			}
			continue
		}
		if _, _, err := parseSym(tag); err != nil {
			return fmt.Errorf("native.Bind: %s.%s: %w", rt.Name(), sf.Name, err)
		}
		if sf.Type.Kind() != reflect.Func || !sf.IsExported() {
			return fmt.Errorf("native.Bind: %s.%s is not an exported function", rt.Name(), sf.Name)
		}
	}
	return nil
}

func fillBind(v reflect.Value, lib uintptr) error {
	rt := v.Type()
	for i := 0; i < rt.NumField(); i++ {
		sf := rt.Field(i)
		tag, ok := sf.Tag.Lookup("sym")
		if !ok {
			continue
		}
		name, soft, err := parseSym(tag)
		if err != nil {
			return fmt.Errorf("native.Bind: %s.%s: %w", rt.Name(), sf.Name, err)
		}
		field := v.Field(i)
		if !field.CanSet() {
			return fmt.Errorf("native.Bind: %s.%s cannot be set", rt.Name(), sf.Name)
		}
		if err := bindExport(lib, name, field.Addr().Interface()); err != nil {
			if soft {
				continue
			}
			return err
		}
	}
	return nil
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
