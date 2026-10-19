//go:build darwin

package term

import (
	"bytes"
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

func openPty() (master *os.File, slave *os.File, err error) {
	master, err = os.OpenFile("/dev/ptmx", os.O_RDWR|unix.O_NOCTTY|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %w", ErrPty, err)
	}
	fd := int(master.Fd())
	if err := unix.IoctlSetInt(fd, unix.TIOCPTYGRANT, 0); err != nil {
		master.Close()
		return nil, nil, fmt.Errorf("%w: %w", ErrPty, err)
	}
	if err := unix.IoctlSetInt(fd, unix.TIOCPTYUNLK, 0); err != nil {
		master.Close()
		return nil, nil, fmt.Errorf("%w: %w", ErrPty, err)
	}
	name, err := darwinPtyName(fd)
	if err != nil {
		master.Close()
		return nil, nil, err
	}
	slave, err = os.OpenFile(name, os.O_RDWR|unix.O_NOCTTY|unix.O_CLOEXEC, 0)
	if err != nil {
		master.Close()
		return nil, nil, fmt.Errorf("%w: %w", ErrPty, err)
	}
	return master, slave, nil
}

func darwinPtyName(fd int) (string, error) {
	var buf [128]byte
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), uintptr(unix.TIOCPTYGNAME), uintptr(unsafe.Pointer(&buf[0])))
	if errno != 0 {
		return "", fmt.Errorf("%w: %w", ErrPty, errno)
	}
	n := bytes.IndexByte(buf[:], 0)
	if n <= 0 {
		return "", fmt.Errorf("%w: %s", ErrPty, "pty name")
	}
	return string(buf[:n]), nil
}
