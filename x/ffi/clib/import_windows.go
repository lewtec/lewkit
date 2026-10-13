//go:build windows && (amd64 || arm64 || 386)

package clib

import "unsafe"

// The Windows loader fills these before the first exported call.
// The entry assembly calls them with the host ABI. %N is the stdcall
// argument count the 386 linker uses; amd64 and arm64 ignore it.

//go:cgo_import_dynamic github.com/lewtec/lewkit/x/ffi/clib._CreatePipe CreatePipe%4 "kernel32.dll"
//go:cgo_import_dynamic github.com/lewtec/lewkit/x/ffi/clib._WriteFile WriteFile%5 "kernel32.dll"
//go:cgo_import_dynamic github.com/lewtec/lewkit/x/ffi/clib._WaitForSingleObject WaitForSingleObject%2 "kernel32.dll"

type stdFunction unsafe.Pointer

var (
	_CreatePipe          stdFunction
	_WriteFile           stdFunction
	_WaitForSingleObject stdFunction
)
