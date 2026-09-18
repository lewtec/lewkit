package thread

import "runtime"

func init() {
	// AppKit's nextEvent must run on the process main thread.
	// Lock before main() so the main goroutine cannot migrate off it.
	runtime.LockOSThread()
	processMainTID = osThread()
}

var processMainTID uint64
