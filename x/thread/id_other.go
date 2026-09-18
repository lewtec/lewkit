//go:build !darwin && !linux && !windows

package thread

func osThread() uint64 { return 0 }
