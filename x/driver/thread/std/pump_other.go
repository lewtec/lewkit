//go:build (!linux && !windows) || android || (windows && !amd64 && !arm64)

package std

func pumpMessages() {}
