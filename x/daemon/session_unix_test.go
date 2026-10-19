//go:build unix

package daemon_test

import "golang.org/x/sys/unix"

func childSession() int {
	sid, err := unix.Getsid(0)
	if err != nil {
		return -1
	}
	return sid
}
