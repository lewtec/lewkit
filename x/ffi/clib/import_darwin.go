//go:build darwin && (amd64 || arm64)

package clib

// The dynamic linker binds these to libSystem before the first call.
// The entry assembly calls them with the C ABI.

//go:cgo_import_dynamic libc_pipe pipe "/usr/lib/libSystem.B.dylib"
//go:cgo_import_dynamic libc_write write "/usr/lib/libSystem.B.dylib"
//go:cgo_import_dynamic libc_usleep usleep "/usr/lib/libSystem.B.dylib"
//go:cgo_import_dynamic libc_error __error "/usr/lib/libSystem.B.dylib"
