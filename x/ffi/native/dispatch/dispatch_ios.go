//go:build ios

package dispatch

/*
#include <dispatch/dispatch.h>
#include <pthread.h>

extern void lewkitDispatchMain(void);

static void lewkit_dispatch_thunk(void *ctx) {
	(void)ctx;
	lewkitDispatchMain();
}

static void lewkitRunOnMain(void) {
	if (pthread_main_np() != 0) {
		lewkitDispatchMain();
		return;
	}
	dispatch_sync_f(dispatch_get_main_queue(), 0, lewkit_dispatch_thunk);
}
*/
import "C"
import "sync"

// A purego callback on the UIKit main thread is not a cgo entry.
// openSurface lays the view out before it returns, and that calls back
// into Go. The nested call has to be a cgo export on the same stack.
var (
	callMu sync.Mutex
	callFn func()
)

//export lewkitDispatchMain
func lewkitDispatchMain() {
	if callFn != nil {
		callFn()
	}
}

// OnMain runs fn on the UIKit main thread.
// A call that is already there runs fn directly.
func OnMain(fn func()) {
	if fn == nil {
		return
	}
	if C.pthread_main_np() != 0 {
		fn()
		return
	}
	callMu.Lock()
	defer callMu.Unlock()
	callFn = fn
	C.lewkitRunOnMain()
	callFn = nil
}
