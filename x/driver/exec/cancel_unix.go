//go:build unix

package exec

import (
	"os"
	"os/exec"
	"syscall"

	"golang.org/x/term"
)

func prepareCancel(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	// A new process group is a background group for the terminal. Reading it
	// stops the child with SIGTTIN, and writing it can stop the child with
	// SIGTTOU when TOSTOP is set. Keep terminal commands in the foreground
	// group; cancel then falls back to the direct process.
	if !attachedToTerminal(cmd) {
		cmd.SysProcAttr.Setpgid = true
	}
}

func attachedToTerminal(cmd *exec.Cmd) bool {
	for _, stream := range []any{cmd.Stdin, cmd.Stdout, cmd.Stderr} {
		file, ok := stream.(*os.File)
		if ok && term.IsTerminal(int(file.Fd())) {
			return true
		}
	}
	return false
}

func trackProcess(*exec.Cmd) {}

func finishCancel(*exec.Cmd) {}

func cancelProcess(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	// Kill the process group while it is still the group we started.
	// After Wait, disarmCancel skips this so a reused pid is not signalled.
	if cmd.SysProcAttr != nil && cmd.SysProcAttr.Setpgid {
		if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err == nil {
			return
		}
	}
	_ = cmd.Process.Kill()
}
