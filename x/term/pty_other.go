//go:build !linux && !darwin

package term

import "context"

func startProcess(context.Context, []string, string, int, int) (*process, error) {
	return nil, ErrPty
}

func (p *process) Resize(int, int) error {
	return ErrPty
}

func (p *process) Close() {
	if p == nil {
		return
	}
	if p.master != nil {
		_ = p.master.Close()
	}
	if p.cmd != nil && p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}
	p.finish()
}
