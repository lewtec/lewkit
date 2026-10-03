//go:build !windows || (!amd64 && !arm64)

package std

func pumpMessages() {}
