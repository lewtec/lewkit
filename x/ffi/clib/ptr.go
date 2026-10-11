package clib

import "unsafe"

// Ptr turns a register word into a pointer.
// The word is an address the C caller passed. Reinterpreting the bits
// keeps the conversion in the form the compiler accepts for an address
// that was never a Go pointer.
func Ptr(n uintptr) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(&n))
}
