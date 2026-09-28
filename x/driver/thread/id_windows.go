//go:build windows

package thread

import "github.com/lewtec/lewkit/x/ffi/native"

var getCurrentThreadId = native.ProcOf("kernel32.dll", "GetCurrentThreadId")

func osThread() uint64 {
	id, r2, err := getCurrentThreadId.Call()
	if id == 0 && r2 == 0 && err != nil {
		return 0
	}
	return uint64(id)
}
