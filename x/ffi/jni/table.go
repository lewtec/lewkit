package jni

import "unsafe"

// jniVersion is the JNINativeInterface slot of GetVersion.
// The four slots before it are reserved and null.
const jniVersion = 4

func untag(p uintptr) uintptr { return p & 0x00ffffffffffffff }

// functionTable splits a JNIEnv from its JNINativeInterface.
// ART's JNIEnv is the C++ object: its first word points at the table, and
// that pointer may carry a tag in the top byte. A table starts with four
// null reserved slots and GetVersion. self keeps the caller's pointer,
// including that tag, because CheckJNI compares it bitwise.
func functionTable(raw uintptr, load func(uintptr) uintptr) (self, tab uintptr, ok bool) {
	if raw == 0 || load == nil {
		return 0, 0, false
	}
	base := untag(raw)
	if t := untag(load(base)); jniTable(t, load) {
		return raw, t, true
	}
	if jniTable(base, load) {
		return raw, base, true
	}
	return 0, 0, false
}

func jniTable(p uintptr, load func(uintptr) uintptr) bool {
	if p == 0 {
		return false
	}
	step := unsafe.Sizeof(uintptr(0))
	for i := uintptr(0); i < jniVersion; i++ {
		if load(p+i*step) != 0 {
			return false
		}
	}
	return load(p+jniVersion*step) != 0
}
