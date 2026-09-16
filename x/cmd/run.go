package cmd

import (
	"context"
	"reflect"
)

func runSelected(ctx context.Context, v reflect.Value) error {
	return runSelectedAt(ctx, v, false)
}

func runSelectedAt(ctx context.Context, v reflect.Value, cmd bool) error {
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}
	t := v.Type()
	for i := range t.NumField() {
		sf := t.Field(i)
		fv := v.Field(i)
		if sf.Anonymous {
			continue
		}
		if fv.Kind() != reflect.Pointer || fv.IsNil() || fv.Type().Elem().Kind() != reflect.Struct {
			continue
		}
		if _, ok := commandName(sf, fv); ok {
			return runSelectedAt(ctx, rvalue{fv}.settable(), true)
		}
	}
	if run := runMethod(v); run.IsValid() {
		return callRun(ctx, run)
	}
	if cmd || hasCommands(v) {
		return ErrUsage
	}
	return nil
}

func hasCommands(v reflect.Value) bool {
	t := v.Type()
	for i := range t.NumField() {
		sf := t.Field(i)
		fv := v.Field(i)
		if sf.Anonymous {
			continue
		}
		if _, ok := commandName(sf, fv); ok {
			return true
		}
	}
	return false
}

func runMethod(v reflect.Value) reflect.Value {
	m := rvalue{v}.settable().Addr().MethodByName("Run")
	if !m.IsValid() {
		return reflect.Value{}
	}
	mt := m.Type()
	if mt.NumIn() != 1 || mt.In(0) != reflect.TypeFor[context.Context]() {
		return reflect.Value{}
	}
	if mt.NumOut() != 1 || mt.Out(0) != reflect.TypeFor[error]() {
		return reflect.Value{}
	}
	return m
}

func callRun(ctx context.Context, m reflect.Value) error {
	out := m.Call([]reflect.Value{reflect.ValueOf(ctx)})[0].Interface()
	if err, ok := out.(error); ok {
		return err
	}
	return nil
}

func selectedCommand(v reflect.Value, name string) (reflect.Value, string) {
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return v, name
		}
		v = v.Elem()
	}
	if cmd, cmdName, ok := findSelectedCommand(v); ok {
		return selectedCommand(cmd, name+" "+cmdName)
	}
	return v, name
}

func findSelectedCommand(v reflect.Value) (reflect.Value, string, bool) {
	if v.Kind() != reflect.Struct {
		return reflect.Value{}, "", false
	}
	t := v.Type()
	for i := range t.NumField() {
		sf := t.Field(i)
		fv := v.Field(i)
		if !fv.CanAddr() {
			continue
		}
		_, flatten := sf.Tag.Lookup("flatten")
		if (sf.Anonymous || flatten) && shouldFlatten(fv) {
			ev, err := derefStruct(fv)
			if err != nil {
				continue
			}
			if cmd, name, ok := findSelectedCommand(ev); ok {
				return cmd, name, true
			}
			continue
		}
		if fv.Kind() == reflect.Pointer && !fv.IsNil() {
			if name, ok := commandName(sf, fv); ok {
				return fv, name, true
			}
		}
	}
	return reflect.Value{}, "", false
}
