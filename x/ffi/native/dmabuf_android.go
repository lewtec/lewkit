//go:build android

package native

// DMABufSmoke is the desktop EGL probe. Android keeps its own GL stack.
func DMABufSmoke() bool { return true }
