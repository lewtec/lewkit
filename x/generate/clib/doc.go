// Package clib writes a cgo-free shared library from a C header.
// The generated package imports x/ffi/clib. This package does not.
// Importing the runtime would start its pipe from every lewkit process.
// A struct of integers, bools, and pointers, up to 16 bytes, passes and
// returns in register eightbytes on SysV amd64 and AAPCS64. Windows amd64
// does that for a struct of at most 8 bytes. A pointer to a struct is a
// Go struct pointer on every host.
// The host table is the gate. BuildFor returns ErrHost when the pair has
// no internal PIE shared library or the generated export constraint omits
// that pair. amd64 and arm64 link through the embedded compiler.
// windows/386 uses the host go command. A linked image missing an export
// name is an error.
package clib
