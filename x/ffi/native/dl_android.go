//go:build android && !cgo && arm64

package native

// libdl.so is the NDK library that exports dlopen. libc.so does not, so a
// NEEDED entry for libc leaves the loader unable to locate dlopen.
//go:cgo_import_dynamic libc_dlopen dlopen "libdl.so"
//go:cgo_import_dynamic libc_dlsym dlsym "libdl.so"
//go:cgo_import_dynamic libc_dlerror dlerror "libdl.so"
//go:cgo_import_dynamic libc_dlclose dlclose "libdl.so"
//go:cgo_import_dynamic _ _ "libdl.so"

const arm64Loader = true

func libcDlopen(path *byte, mode int) uintptr
func libcDlsym(handle uintptr, name *byte) uintptr
func libcDlerror() *byte
func libcDlclose(handle uintptr) int
