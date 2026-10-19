//go:build !unix

package daemon_test

func childSession() int { return -1 }
