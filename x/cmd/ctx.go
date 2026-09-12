package cmd

import (
	"context"
	"fmt"
	"reflect"
	"strings"
)

type bagKey struct{}

type valueBag map[string]any

// Get returns the value stored under key. It panics if the context has no
// value bag, the key is missing, or v is not a T.
func Get[T any](ctx context.Context, key string) T {
	b, ok := ctx.Value(bagKey{}).(valueBag)
	if !ok {
		panic("cmd: context has no values")
	}
	v, ok := b[key]
	if !ok {
		panic(`cmd: context key "` + key + `" not set`)
	}
	t, ok := v.(T)
	if !ok {
		panic(fmt.Sprintf("cmd: context key %q is %T, not %T", key, v, *new(T)))
	}
	return t
}

func withValues(ctx context.Context) context.Context {
	if _, ok := ctx.Value(bagKey{}).(valueBag); ok {
		return ctx
	}
	return context.WithValue(ctx, bagKey{}, valueBag{})
}

func put(ctx context.Context, key string, v any) {
	b, ok := ctx.Value(bagKey{}).(valueBag)
	if !ok {
		panic("cmd: context has no values")
	}
	b[key] = v
}

func ctxName(sf reflect.StructField) string {
	ctx, ok := sf.Tag.Lookup("ctx")
	if !ok {
		return ""
	}
	if ctx != "" {
		return ctx
	}
	if long := sf.Tag.Get("long"); long != "" {
		return long
	}
	return strings.ToLower(sf.Name)
}

func bind(ctx context.Context, v reflect.Value) {
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return
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
			bind(ctx, ev)
			continue
		}
		if key := ctxName(sf); key != "" && (fv.Kind() != reflect.Pointer || !fv.IsNil()) {
			put(ctx, key, storedValue(rvalue{fv}.settable()))
		}
		if fv.Kind() == reflect.Pointer && !fv.IsNil() && fv.Type().Elem().Kind() == reflect.Struct {
			if _, ok := commandName(sf, fv); ok {
				bind(ctx, rvalue{fv}.settable())
			}
		}
	}
}

func storedValue(fv reflect.Value) any {
	switch fv.Kind() {
	case reflect.Slice, reflect.Array:
		return unwrapSeq(fv)
	}
	if (rvalue{fv}).hasValue() {
		return (rvalue{fv}).callValue()
	}
	return fv.Interface()
}

func unwrapSeq(fv reflect.Value) any {
	outT, ok := (rvalue{reflect.New(fv.Type().Elem())}).valueType()
	if !ok {
		return fv.Interface()
	}
	n := fv.Len()
	var out reflect.Value
	if fv.Kind() == reflect.Array {
		out = reflect.New(reflect.ArrayOf(n, outT)).Elem()
	} else {
		out = reflect.MakeSlice(reflect.SliceOf(outT), n, n)
	}
	for i := range n {
		out.Index(i).Set(reflect.ValueOf((rvalue{fv.Index(i)}).callValue()))
	}
	return out.Interface()
}

func (v rvalue) valueType() (reflect.Type, bool) {
	m := v.ptr().MethodByName("Value")
	if !m.IsValid() {
		return nil, false
	}
	t := m.Type()
	if t.NumIn() != 0 || t.NumOut() != 1 {
		return nil, false
	}
	return t.Out(0), true
}

func (v rvalue) hasValue() bool {
	_, ok := v.valueType()
	return ok
}

func (v rvalue) callValue() any {
	return v.ptr().MethodByName("Value").Call(nil)[0].Interface()
}
