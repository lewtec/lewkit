//go:build !linux && !darwin

package term

func processCwd(int) (string, bool) { return "", false }
