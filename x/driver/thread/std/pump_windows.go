//go:build windows && (amd64 || arm64)

package std

import (
	"unsafe"

	"github.com/lewtec/lewkit/x/ffi/native"
)

const pmRemove = 1

// msg matches the amd64 and arm64 Windows MSG.
// POINT begins 4 bytes after time, at byte 36. The struct is 48 bytes.
type msg struct {
	hwnd     uintptr
	message  uint32
	pad0     uint32
	wParam   uintptr
	lParam   uintptr
	time     uint32
	ptX, ptY int32
}

var (
	procPeekMessage      = native.ProcOf("user32.dll", "PeekMessageW")
	procTranslateMessage = native.ProcOf("user32.dll", "TranslateMessage")
	procDispatchMessage  = native.ProcOf("user32.dll", "DispatchMessageW")
)

// pumpMessages drains the Win32 queue on this thread.
// Dialogs and windows created here dispatch while Loop is between jobs.
func pumpMessages() {
	for {
		var m msg
		r, _, _ := procPeekMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0, pmRemove)
		if r == 0 {
			return
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
	}
}
