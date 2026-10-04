package vulkan

// UIThread runs work on the UI thread.
// The driver facade installs the thread driver. This package does not import x/driver.
type UIThread interface {
	Do(func())
	Bound() bool
	// ProcessMain reports whether this goroutine is on the UI thread.
	ProcessMain() bool
	OnIdle(func())
}

var uiThread UIThread

// SetUIThread installs the process main thread. A nil hook runs Do inline
// and reports the thread as unbound.
func SetUIThread(t UIThread) { uiThread = t }

func uiDo(fn func()) {
	if fn == nil {
		return
	}
	if uiThread == nil {
		fn()
		return
	}
	uiThread.Do(fn)
}

func uiBound() bool {
	if uiThread == nil {
		return false
	}
	return uiThread.Bound()
}

func uiProcessMain() bool {
	if uiThread == nil {
		return false
	}
	return uiThread.ProcessMain()
}

func uiOnIdle(fn func()) {
	if uiThread == nil || fn == nil {
		return
	}
	uiThread.OnIdle(fn)
}
