//go:build windows

package term

import (
	"os/exec"
	"syscall"
)

// detachedProcess is Windows DETACHED_PROCESS. syscall does not export it.
const detachedProcess = 0x00000008

func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | detachedProcess,
		HideWindow:    true,
	}
}
