package cmd

import (
	"context"
	"reflect"
)

func runSelected(ctx context.Context, v reflect.Value) error {
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
			return runSelected(ctx, rvalue{fv}.settable())
		}
	}
	return callRun(ctx, v)
}

func callRun(ctx context.Context, v reflect.Value) error {
	m := rvalue{v}.settable().Addr().MethodByName("Run")
	if !m.IsValid() {
		return nil
	}
	mt := m.Type()
	if mt.NumIn() != 1 || mt.In(0) != reflect.TypeFor[context.Context]() {
		return nil
	}
	if mt.NumOut() != 1 || mt.Out(0) != reflect.TypeFor[error]() {
		return nil
	}
	out := m.Call([]reflect.Value{reflect.ValueOf(ctx)})[0].Interface()
	if err, ok := out.(error); ok {
		return err
	}
	return nil
}
