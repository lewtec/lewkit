// Package clib writes a cgo-free shared library from a C header.
// The generated package imports x/ffi/clib. This package does not.
// Importing the runtime would start its pipe from every lewkit process.
// The host table is the gate. BuildFor returns ErrHost when the pair has
// no internal PIE shared library or the generated export constraint omits
// that pair. A linked image missing an export name is an error.
package clib
