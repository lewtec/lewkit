//go:build !windows && !android && !linux

package native

func preloadSiblingDeps(string, int) {}
