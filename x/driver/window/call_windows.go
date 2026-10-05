//go:build windows

package window

import (
	"sync"
	"sync/atomic"

	"github.com/lewtec/lewkit/x/ffi/native"
)

// uiMessage asks the window procedure to run a function on its thread.
// A modal shell dialog has to start from DispatchMessage, not from a
// cross-thread SendMessage, or the owner stays on the wrong thread.
const uiMessage = 0x8000 + 51

type uiJob struct {
	fn   func()
	done chan struct{}
}

var (
	uiWindows sync.Map // hwnd -> struct{}
	latestUI  atomic.Uintptr
	uiJobs    sync.Map // id -> *uiJob
	uiSeq     atomic.Uint64

	procPostMessageW             = native.ProcOf("user32.dll", "PostMessageW")
	procGetWindowThreadProcessId = native.ProcOf("user32.dll", "GetWindowThreadProcessId")
	procGetCurrentThreadId       = native.ProcOf("kernel32.dll", "GetCurrentThreadId")
)

// RegisterUI marks hwnd as a window whose procedure runs DeliverUI.
func RegisterUI(hwnd uintptr) {
	if hwnd == 0 {
		return
	}
	uiWindows.Store(hwnd, struct{}{})
	latestUI.Store(hwnd)
}

// ForgetUI drops a window registered with RegisterUI.
func ForgetUI(hwnd uintptr) {
	if hwnd == 0 {
		return
	}
	uiWindows.Delete(hwnd)
	latestUI.CompareAndSwap(hwnd, 0)
}

// OwnsUI reports whether hwnd was registered and not forgotten.
func OwnsUI(hwnd uintptr) bool {
	if hwnd == 0 {
		return false
	}
	_, ok := uiWindows.Load(hwnd)
	return ok
}

// LatestUI is the most recently registered window that is still registered.
func LatestUI() uintptr {
	if hwnd := latestUI.Load(); OwnsUI(hwnd) {
		return hwnd
	}
	var found uintptr
	uiWindows.Range(func(key, _ any) bool {
		found, _ = key.(uintptr)
		return false
	})
	return found
}

// CallUI runs fn on the thread that pumps hwnd and waits until it returns.
// It reports false when hwnd is not a registered UI window.
func CallUI(hwnd uintptr, fn func()) bool {
	if fn == nil || !OwnsUI(hwnd) {
		return false
	}
	if onUIThread(hwnd) {
		fn()
		return true
	}
	id := uintptr(uiSeq.Add(1))
	job := &uiJob{fn: fn, done: make(chan struct{})}
	uiJobs.Store(id, job)
	posted, _, _ := procPostMessageW.Call(hwnd, uiMessage, id, 0)
	if posted == 0 {
		uiJobs.Delete(id)
		return false
	}
	<-job.done
	return true
}

// DeliverUI runs a job posted by CallUI. The window procedure calls it
// before its own messages. A handled message returns true.
func DeliverUI(msg, wparam uintptr) bool {
	if msg != uiMessage {
		return false
	}
	value, ok := uiJobs.LoadAndDelete(wparam)
	if !ok {
		return true
	}
	job := value.(*uiJob)
	defer close(job.done)
	job.fn()
	return true
}

func onUIThread(hwnd uintptr) bool {
	tid, _, _ := procGetWindowThreadProcessId.Call(hwnd, 0)
	if tid == 0 {
		return false
	}
	our, _, _ := procGetCurrentThreadId.Call()
	return tid == our
}
