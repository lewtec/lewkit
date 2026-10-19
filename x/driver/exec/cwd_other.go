//go:build !linux && !darwin

package exec

func cwdOf(int) (string, bool) { return "", false }
