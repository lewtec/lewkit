//go:build !android

package entry

// AndroidHost is false outside Android.
func AndroidHost() bool { return false }

// NotifyReady is a no-op where the process has no Android host.
func NotifyReady(string) {}

// ShowSurface is a no-op where the process has no Android host.
func ShowSurface() {}

// NotifyFail is a no-op where the process has no Android host.
func NotifyFail(string) {}
