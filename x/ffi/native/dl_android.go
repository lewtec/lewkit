//go:build android && !cgo && arm64

package native

//go:cgo_import_dynamic libc_dlopen dlopen "libc.so"
//go:cgo_import_dynamic libc_dlsym dlsym "libc.so"
//go:cgo_import_dynamic libc_dlerror dlerror "libc.so"
//go:cgo_import_dynamic libc_dlclose dlclose "libc.so"
//go:cgo_import_dynamic _ _ "libc.so"

const arm64Loader = true

func libcDlopen(path *byte, mode int) uintptr
func libcDlsym(handle uintptr, name *byte) uintptr
func libcDlerror() *byte
func libcDlclose(handle uintptr) int
