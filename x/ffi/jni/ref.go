package jni

import (
	"github.com/lewtec/lewkit/x/errs"
)

// ErrUnavailable means this build cannot call Java.
var ErrUnavailable = errs.Wrap(errs.ErrUnavailable, "jni unavailable")

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

// Peer is the JNI object pointer. It stays valid until Release.
func (r *Ref) Peer() uintptr {
	if r == nil {
		return 0
	}
	return r.ptr
}
