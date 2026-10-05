//go:build !windows

package vulkan

import "github.com/lewtec/lewkit/x/ffi/native"

func windowsLibNames() []string { return []string{"vulkan-1.dll"} }

func openLoader() (uintptr, error) {
	return native.OpenChain(native.Lazy, libNames()...)
}
