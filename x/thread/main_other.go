//go:build !darwin

package thread

// ProcessMain reports whether this goroutine is on the process main OS thread.
func ProcessMain() bool {
	id := osThread()
	return id != 0 && id == processMainTID
}
