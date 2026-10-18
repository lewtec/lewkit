//go:build linux || darwin

package term

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	_ "github.com/lewtec/lewkit/x/driver/exec/prelude"
	"golang.org/x/sys/unix"
)

func startProcess(ctx context.Context, argv []string, dir string, rows, cols int) (*process, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(argv) == 0 || argv[0] == "" {
		return nil, fmt.Errorf("%w: %s", ErrCommand, "empty command")
	}
	master, slave, err := openPty()
	if err != nil {
		return nil, err
	}
	closeMaster := true
	defer func() {
		if closeMaster {
			master.Close()
		}
	}()
	defer slave.Close()
	if err := resizePty(master, rows, cols); err != nil {
		return nil, err
	}
	cmd := execdriver.MustCommand(ctx, argv[0], argv[1:]...)
	cmd.Dir = dir
	cmd.Env = childEnv(rows, cols)
	cmd.Stdin = slave
	cmd.Stdout = slave
	cmd.Stderr = slave
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 0}
	cmd.WaitDelay = 2 * time.Second
	stop := context.AfterFunc(ctx, func() {
		if cmd.Process != nil {
			_ = killTree(cmd.Process.Pid)
		}
	})
	if err := cmd.Start(); err != nil {
		stop()
		if errors.Is(err, exec.ErrNotFound) {
			return nil, fmt.Errorf("%w: %w", ErrCommand, err)
		}
		return nil, fmt.Errorf("%w: %w", ErrPty, err)
	}
	if err := ctx.Err(); err != nil {
		_ = killTree(cmd.Process.Pid)
		return nil, err
	}
	closeMaster = false
	return &process{master: master, cmd: cmd}, nil
}

func childEnv(rows, cols int) []string {
	out := make([]string, 0, len(os.Environ())+3)
	for _, item := range os.Environ() {
		if strings.HasPrefix(item, "LINES=") || strings.HasPrefix(item, "COLUMNS=") || strings.HasPrefix(item, "TERM=") {
			continue
		}
		out = append(out, item)
	}
	out = append(out,
		"TERM=xterm-256color",
		fmt.Sprintf("LINES=%d", rows),
		fmt.Sprintf("COLUMNS=%d", cols),
	)
	return out
}

func (p *process) Resize(rows, cols int) error {
	if p == nil || p.master == nil {
		return fmt.Errorf("%w: %s", ErrPty, "no pty")
	}
	return resizePty(p.master, rows, cols)
}

func resizePty(f *os.File, rows, cols int) error {
	size := &unix.Winsize{Row: uint16(rows), Col: uint16(cols)}
	if err := unix.IoctlSetWinsize(int(f.Fd()), unix.TIOCSWINSZ, size); err != nil {
		return fmt.Errorf("%w: %w", ErrPty, err)
	}
	return nil
}

func (p *process) Close() {
	if p == nil {
		return
	}
	if p.cmd != nil && p.cmd.Process != nil {
		_ = killTree(p.cmd.Process.Pid)
	}
	if p.master != nil {
		_ = p.master.Close()
	}
	p.finish()
}

func killTree(pid int) error {
	err := syscall.Kill(-pid, syscall.SIGKILL)
	if err != nil {
		return syscall.Kill(pid, syscall.SIGKILL)
	}
	return nil
}
