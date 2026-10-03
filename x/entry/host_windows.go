//go:build windows

package entry

import (
	"syscall"
	"unsafe"

	"github.com/lewtec/lewkit/x/ffi/native"
	"github.com/lewtec/lewkit/x/release"
)

var (
	procSetDpiContext = native.ProcOf("user32.dll", "SetProcessDpiAwarenessContext")
	procSetDpiAware   = native.ProcOf("user32.dll", "SetProcessDPIAware")
	procSetAppID      = native.ProcOf("shell32.dll", "SetCurrentProcessExplicitAppUserModelID")
	procGetConsole    = native.ProcOf("kernel32.dll", "GetConsoleWindow")
)

func prepareHost() {
	// DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 is -4.
	if r, _, _ := procSetDpiContext.Call(^uintptr(3)); r == 0 {
		procSetDpiAware.Call()
	}
	id, err := release.AppID()
	if err != nil || len(id) == 0 || len(id) > 128 {
		return
	}
	ptr, err := syscall.UTF16PtrFromString(id)
	if err != nil {
		return
	}
	procSetAppID.Call(uintptr(unsafe.Pointer(ptr)))
}

func windowsGUI() bool {
	hwnd, _, _ := procGetConsole.Call()
	return hwnd == 0
}
