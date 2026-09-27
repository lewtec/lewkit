package driver

import "sync/atomic"

// appMode is set for the life of an app process. Memory drivers stay
// incompatible so a release build cannot pretend to open a surface.
var appMode atomic.Bool

// SetAppMode marks this process as an app.
func SetAppMode(on bool) { appMode.Store(on) }

// AppMode reports whether [SetAppMode] is on.
func AppMode() bool { return appMode.Load() }
