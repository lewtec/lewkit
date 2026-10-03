package jni

import (
	"errors"
	"fmt"
)

var errJavaValue = errors.New("java value")

// AsRef returns v as a Java reference. A null reference is a nil *Ref.
func AsRef(v any, err error) (*Ref, error) {
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}
	ref, ok := v.(*Ref)
	if !ok || ref == nil {
		return nil, fmt.Errorf("%w: %T", errJavaValue, v)
	}
	return ref, nil
}

// Int returns v as an int.
func Int(v any, err error) (int, error) {
	if err != nil {
		return 0, err
	}
	n, ok := v.(int)
	if !ok {
		return 0, fmt.Errorf("%w: %T", errJavaValue, v)
	}
	return n, nil
}

// Text returns v as a string.
func Text(v any, err error) (string, error) {
	if err != nil {
		return "", err
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("%w: %T", errJavaValue, v)
	}
	return s, nil
}

// Bool returns v as a bool.
func Bool(v any, err error) (bool, error) {
	if err != nil {
		return false, err
	}
	b, ok := v.(bool)
	if !ok {
		return false, fmt.Errorf("%w: %T", errJavaValue, v)
	}
	return b, nil
}

// Float returns v as a float64. A float32 is widened.
func Float(v any, err error) (float64, error) {
	if err != nil {
		return 0, err
	}
	switch n := v.(type) {
	case float32:
		return float64(n), nil
	case float64:
		return n, nil
	default:
		return 0, fmt.Errorf("%w: %T", errJavaValue, v)
	}
}
