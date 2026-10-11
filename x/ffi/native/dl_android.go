//go:build android && !cgo && arm64 && appmode

package native

import "unsafe"

// libdl.so is the NDK library that exports dlopen. libc.so does not, so a
// NEEDED entry for libc leaves the loader unable to locate dlopen.
//go:cgo_import_dynamic libc_dlopen dlopen "libdl.so"
//go:cgo_import_dynamic libc_dlsym dlsym "libdl.so"
//go:cgo_import_dynamic libc_dlerror dlerror "libdl.so"
//go:cgo_import_dynamic libc_dlclose dlclose "libdl.so"
//go:cgo_import_dynamic _ _ "libdl.so"

const arm64Loader = true

// The loader runs in the dynamic linker, which uses more stack than the
// calling goroutine reserved. callCOnG0 reads eight argument words even
// when the C function takes fewer, so each argv is eight words.
// Hook.call starts the worker after the library load. Before that, a
// direct call stays on this thread: JNI_OnLoad cannot start the pthread,
// and a cloned M must not dlopen.

func cThreadAlive() bool

func libcDlopenDirect(path *byte, mode int) uintptr
func libcDlsymDirect(handle uintptr, name *byte) uintptr
func libcDlerrorDirect() *byte
func libcDlcloseDirect(handle uintptr) int

func libcDlopenPC() uintptr
func libcDlsymPC() uintptr
func libcDlerrorPC() uintptr
func libcDlclosePC() uintptr

//go:nosplit
func libcDlopen(path *byte, mode int) uintptr {
	if !cThreadAlive() {
		return libcDlopenDirect(path, mode)
	}
	var argv [8]uintptr
	argv[0] = uintptr(unsafe.Pointer(path))
	argv[1] = uintptr(mode)
	r, _, _ := callC(libcDlopenPC(), &argv[0], 2)
	return r
}

//go:nosplit
func libcDlsym(handle uintptr, name *byte) uintptr {
	if !cThreadAlive() {
		return libcDlsymDirect(handle, name)
	}
	var argv [8]uintptr
	argv[0] = handle
	argv[1] = uintptr(unsafe.Pointer(name))
	r, _, _ := callC(libcDlsymPC(), &argv[0], 2)
	return r
}

//go:nosplit
func libcDlerror() *byte {
	if !cThreadAlive() {
		return libcDlerrorDirect()
	}
	var argv [8]uintptr
	r, _, _ := callC(libcDlerrorPC(), &argv[0], 0)
	return (*byte)(unsafe.Pointer(r))
}

//go:nosplit
func libcDlclose(handle uintptr) int {
	if !cThreadAlive() {
		return libcDlcloseDirect(handle)
	}
	var argv [8]uintptr
	argv[0] = handle
	r, _, _ := callC(libcDlclosePC(), &argv[0], 1)
	return int(r)
}
