package jni

import "errors"

// ErrUnavailable means this build cannot call Java.
var ErrUnavailable = errors.New("jni unavailable")

// Ref is one global reference to a Java object.
// Release frees it. Call Release once.
type Ref struct {
	ptr   uintptr
	class string
}

// Class is the runtime class name, such as "java.lang.String".
func (r *Ref) Class() string {
	if r == nil {
		return ""
	}
	return r.class
}
