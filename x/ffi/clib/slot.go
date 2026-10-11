//go:build ((linux || android) && (amd64 || arm64 || 386 || arm)) || ((windows || darwin) && (amd64 || arm64 || 386))

package clib

// Slot is the C call the caller parks on.
// The assembly entry writes it on the C stack. A goroutine reads it and
// stores the result. The layout is eight integer words, eight float
// registers, and the completion word. Every field is an integer so the
// garbage collector does not scan the C stack. The offsets are the same
// on every arch this file builds for: uint64 stays 8 bytes, and the two
// leading int32 fields already fill its alignment.
type Slot struct {
	ID   int32
	_    int32
	Args [8]uint64
	XMM  [8]uint64
	Ret  uint64
	RetX uint64
	Done uint32
	Kind uint32
}
