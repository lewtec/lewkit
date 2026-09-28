package native

import "unsafe"

// CString returns s followed by a NUL byte.
// The caller keeps the slice alive for the foreign call.
func CString(s string) []byte {
	out := make([]byte, len(s)+1)
	copy(out, s)
	return out
}

// GoString copies a NUL-terminated C string. A zero pointer is empty.
// Reading stops at 1<<20 bytes.
func GoString(p uintptr) string {
	if p == 0 {
		return ""
	}
	n := 0
	for *(*byte)(unsafe.Pointer(p + uintptr(n))) != 0 {
		n++
		if n > 1<<20 {
			break
		}
	}
	if n == 0 {
		return ""
	}
	return string(unsafe.Slice((*byte)(unsafe.Pointer(p)), n))
}
