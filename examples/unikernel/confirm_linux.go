//go:build linux

package main

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// waitEnter reads the screen console and stdin until Enter, then powers off.
// tty0 is the keyboard on a graphical VM. stdin is the serial console.
func waitEnter() {
	const msg = "press enter to confirm\n"
	fmt.Print(msg)
	fds := []unix.PollFd{{Fd: int32(os.Stdin.Fd()), Events: unix.POLLIN}}
	tty, err := os.OpenFile("/dev/tty0", os.O_RDWR, 0)
	if err == nil {
		defer tty.Close()
		_, _ = tty.WriteString(msg)
		fds = append(fds, unix.PollFd{Fd: int32(tty.Fd()), Events: unix.POLLIN})
	}
	buf := make([]byte, 32)
	for {
		_, err := unix.Poll(fds, -1)
		if err == unix.EINTR {
			continue
		}
		if err != nil {
			return
		}
		live := false
		for i := range fds {
			if fds[i].Fd < 0 {
				continue
			}
			live = true
			if fds[i].Revents&(unix.POLLIN|unix.POLLHUP|unix.POLLERR) == 0 {
				continue
			}
			var n int
			var readErr error
			if tty != nil && fds[i].Fd == int32(tty.Fd()) {
				n, readErr = tty.Read(buf)
			} else {
				n, readErr = os.Stdin.Read(buf)
			}
			for _, b := range buf[:n] {
				if b == '\n' || b == '\r' {
					_ = unix.Reboot(unix.LINUX_REBOOT_CMD_POWER_OFF)
					return
				}
			}
			if readErr != nil {
				fds[i].Fd = -1
			}
		}
		if !live {
			return
		}
	}
}
