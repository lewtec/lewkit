package jni

import (
	"errors"
	"fmt"
	"strings"
)

var (
	errEmptyClass  = errors.New("java class name is empty")
	errEmptyMethod = errors.New("java method name is empty")
	errEmptyAnchor = errors.New("jni anchor class is empty")
	errNilObject   = errors.New("nil java object")
)

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
