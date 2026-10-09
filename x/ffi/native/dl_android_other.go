//go:build android && !cgo && !arm64

package native

const arm64Loader = false

func libcDlopen(path *byte, mode int) uintptr {
	_, _ = path, mode
	return 0
}

func libcDlsym(handle uintptr, name *byte) uintptr {
	_, _ = handle, name
	return 0
}

func libcDlerror() *byte { return nil }

func libcDlclose(handle uintptr) int {
	_ = handle
	return -1
}

// Register reports that this architecture has no C call bridge.
func Register(fnptr any, addr uintptr) {
	_, _ = fnptr, addr
	panic("native: android dynamic calls are arm64")
}
