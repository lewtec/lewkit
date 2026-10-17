package test

import (
	"os"
	"runtime"
)

// HostPerm is the permission bits the host filesystem stores for mode.
// Windows keeps only the read-only attribute, so a writable mode is 0666
// and a read-only mode is 0444.
func HostPerm(mode os.FileMode) os.FileMode {
	mode = mode.Perm()
	if runtime.GOOS != "windows" {
		return mode
	}
	if mode&0o200 != 0 {
		return 0o666
	}
	return 0o444
}

// Executable reports whether mode is an executable file on this host.
// Windows does not store the Unix execute bit. A writable regular file
// there is the mode a 0755 write leaves behind.
func Executable(mode os.FileMode) bool {
	if runtime.GOOS == "windows" {
		return mode.Perm()&0o200 != 0
	}
	return mode&0o111 != 0
}
