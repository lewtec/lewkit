//go:build windows

package thread

import "syscall"

var getCurrentThreadId = syscall.NewLazyDLL("kernel32.dll").NewProc("GetCurrentThreadId")

func osThread() uint64 {
	id, _, _ := getCurrentThreadId.Call()
	return uint64(id)
}
