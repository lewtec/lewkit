//go:build linux && amd64

package clib

// Slot is the C call the caller parks on.
// The assembly entry writes it on the C stack. A goroutine reads it and
// stores the result. The layout is the SysV register file plus the futex
// word the caller waits on. Every field is an integer so the garbage
// collector does not scan the C stack.
type Slot struct {
	ID   int32
	_    int32
	Args [6]uintptr
	XMM  [8]uint64
	Ret  uintptr
	RetX uint64
	Done uint32
	_    uint32
}
