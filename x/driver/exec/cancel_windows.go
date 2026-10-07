//go:build windows

package exec

import "os/exec"

func prepareCancel(cmd *exec.Cmd) {}

func cancelProcess(cmd *exec.Cmd) {
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}
