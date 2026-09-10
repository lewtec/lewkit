package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/lewtec/lewkit/x/release"
)

// Run prints help or version when asked, then Setup, then the selected
// command's Run(ctx) error if it has one. T must be App or embed App.
func Run[T any](ctx context.Context, args T) error {
	root := reflect.ValueOf(&args).Elem()
	app, err := appOf(root)
	if err != nil {
		return err
	}
	switch {
	case app.Help():
		text, err := Usage[T](filepath.Base(os.Args[0]))
		if err != nil {
			return err
		}
		_, err = fmt.Fprint(os.Stdout, text)
		return err
	case app.WantVersion():
		_, err := fmt.Fprintln(os.Stdout, release.Version())
		return err
	}
	if err := app.Setup(ctx); err != nil {
		return err
	}
	return runSelected(ctx, root)
}

func appOf(v reflect.Value) (*App, error) {
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil, fmt.Errorf("%w: nil args", ErrInvalidSpec)
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("%w: args must be a struct", ErrInvalidSpec)
	}
	if v.Type() == reflect.TypeFor[App]() {
		return v.Addr().Interface().(*App), nil
	}
	t := v.Type()
	for i := range t.NumField() {
		sf := t.Field(i)
		if !sf.Anonymous {
			continue
		}
		switch sf.Type {
		case reflect.TypeFor[App]():
			return v.Field(i).Addr().Interface().(*App), nil
		case reflect.TypeFor[*App]():
			f := v.Field(i)
			if f.IsNil() {
				return nil, fmt.Errorf("%w: nil App", ErrInvalidSpec)
			}
			return f.Interface().(*App), nil
		}
		if sf.Type.Kind() == reflect.Struct {
			if app, err := appOf(v.Field(i)); err == nil {
				return app, nil
			}
		}
	}
	return nil, fmt.Errorf("%w: missing App", ErrInvalidSpec)
}

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
	if v.Type() == reflect.TypeFor[App]() {
		return nil
	}
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
