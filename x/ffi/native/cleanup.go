package native

import "runtime"

// Arm binds release(arg) to run after owner is unreachable.
// This is the os.Process procedure: arg is a separate allocation,
// and neither release nor arg keeps owner reachable.
// release must be idempotent. Close calls Disarm before release,
// while owner is still reachable.
func Arm[T, S any](owner *T, release func(S), arg S) runtime.Cleanup {
	if owner == nil {
		return runtime.Cleanup{}
	}
	return runtime.AddCleanup(owner, release, arg)
}

// Disarm cancels a cleanup from Arm. owner must stay reachable across
// the cancel, which is what runtime.Cleanup.Stop requires.
func Disarm(owner any, c runtime.Cleanup) {
	c.Stop()
	runtime.KeepAlive(owner)
}
