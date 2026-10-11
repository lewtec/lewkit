// Package clib writes a cgo-free shared library from a C header.
// The generated package imports x/ffi/clib. This package does not.
// Importing the runtime would start its pipe from every lewkit process.
package clib
