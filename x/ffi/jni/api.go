package jni

import (
	"errors"
	"fmt"
	"strings"
)

var (
	errEmptyClass  = errors.New("java class name is empty")
	errEmptyMethod = errors.New("java method name is empty")
	errEmptyField  = errors.New("java field name is empty")
	errEmptyAnchor = errors.New("jni anchor class is empty")
	errNilObject   = errors.New("nil java object")
	errNilInvoke   = errors.New("nil java proxy callback")
)

// SetRunner runs later Java calls on the thread that called Bind.
// fn must run its argument on that thread and wait. The Android host passes
// the UI thread queue, which entry.Run pumps on the loader thread.
func SetRunner(fn func(func())) { setRunner(fn) }

// Bind keeps the ClassLoader of anchor. anchor is a Java class the calling
// thread can see, such as "lewkit/Host" or "lewkit.Host". Call Bind from the
// Java thread that loaded the native library, before CallStatic or New.
func Bind(env uintptr, anchor string) error {
	if strings.TrimSpace(anchor) == "" {
		return errEmptyAnchor
	}
	return bind(env, anchor)
}

// CallStatic invokes a public static method. className uses dots or slashes.
func CallStatic(className, method string, args ...any) (any, error) {
	if err := names(className, method); err != nil {
		return nil, err
	}
	v, err := callStatic(className, method, args...)
	if err != nil {
		return nil, fmt.Errorf("%s.%s: %w", dotted(className), method, err)
	}
	return v, nil
}

// New constructs className with a public constructor.
// The caller releases the result.
func New(className string, args ...any) (*Ref, error) {
	if strings.TrimSpace(className) == "" {
		return nil, errEmptyClass
	}
	ref, err := callNew(className, args...)
	if err != nil {
		return nil, fmt.Errorf("new %s: %w", dotted(className), err)
	}
	return ref, nil
}

// Call invokes a public instance method.
func (r *Ref) Call(method string, args ...any) (any, error) {
	if r == nil || r.ptr == 0 {
		return nil, errNilObject
	}
	if strings.TrimSpace(method) == "" {
		return nil, errEmptyMethod
	}
	v, err := callRef(r, method, args...)
	if err != nil {
		return nil, fmt.Errorf("%s.%s: %w", r.class, method, err)
	}
	return v, nil
}

// Class loads className through the bound ClassLoader.
// The caller releases the result. className uses dots or slashes.
func Class(className string) (*Ref, error) {
	if strings.TrimSpace(className) == "" {
		return nil, errEmptyClass
	}
	ref, err := classObject(className)
	if err != nil {
		return nil, fmt.Errorf("class %s: %w", dotted(className), err)
	}
	return ref, nil
}

// StaticField reads a public static field.
func StaticField(className, name string) (any, error) {
	if strings.TrimSpace(className) == "" {
		return nil, errEmptyClass
	}
	if strings.TrimSpace(name) == "" {
		return nil, errEmptyField
	}
	v, err := staticField(className, name)
	if err != nil {
		return nil, fmt.Errorf("%s.%s: %w", dotted(className), name, err)
	}
	return v, nil
}

// Field reads a public instance field.
func (r *Ref) Field(name string) (any, error) {
	if r == nil || r.ptr == 0 {
		return nil, errNilObject
	}
	if strings.TrimSpace(name) == "" {
		return nil, errEmptyField
	}
	v, err := instanceField(r, name)
	if err != nil {
		return nil, fmt.Errorf("%s.%s: %w", r.class, name, err)
	}
	return v, nil
}

// Proxy returns a Java proxy that implements iface.
// invoke receives the method name and arguments. It runs on the Java thread
// that called the interface method, and it may call back into this package.
// A *Ref argument belongs to invoke. Release drops the proxy and invoke.
func Proxy(iface string, invoke func(method string, args []any) (any, error)) (*Ref, error) {
	if strings.TrimSpace(iface) == "" {
		return nil, errEmptyClass
	}
	if invoke == nil {
		return nil, errNilInvoke
	}
	ref, err := makeProxy(iface, invoke)
	if err != nil {
		return nil, fmt.Errorf("proxy %s: %w", dotted(iface), err)
	}
	return ref, nil
}

// Release drops the global reference. A second call does nothing.
func (r *Ref) Release() {
	if r == nil || r.ptr == 0 {
		return
	}
	if !release(r) {
		return
	}
	r.ptr = 0
	r.class = ""
}

func names(className, method string) error {
	if strings.TrimSpace(className) == "" {
		return errEmptyClass
	}
	if strings.TrimSpace(method) == "" {
		return errEmptyMethod
	}
	return nil
}
