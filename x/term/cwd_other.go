//go:build !linux

package term

func processCwd(int) (string, bool) { return "", false }
