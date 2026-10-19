//go:build unix

package daemon

import (
	"os/exec"
	"syscall"
)

// detach makes the child a session leader so it outlives the caller.
// Setpgid stays unset. setpgid rejects a session leader.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
