//go:build !darwin && !linux && !windows

package std

func osThread() uint64 { return 0 }
