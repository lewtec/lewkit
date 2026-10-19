package term

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
)

// process is the pane's pseudoterminal and child.
type process struct {
	master *os.File
	cmd    *exec.Cmd
	once   sync.Once
	code   int
}

func (p *process) Write(b []byte) error {
	if p == nil || p.master == nil {
		return fmt.Errorf("%w: %s", ErrPty, "no pty")
	}
	_, err := p.master.Write(b)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrTermClosed, err)
	}
	return nil
}

func (p *process) Read(b []byte) (int, error) {
	if p == nil || p.master == nil {
		return 0, io.EOF
	}
	return p.master.Read(b)
}

func (p *process) pid() int {
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return 0
	}
	return p.cmd.Process.Pid
}

func (p *process) finish() int {
	if p == nil || p.cmd == nil {
		return 1
	}
	p.once.Do(func() {
		p.code = exitCode(p.cmd.Wait())
	})
	return p.code
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return 1
}
