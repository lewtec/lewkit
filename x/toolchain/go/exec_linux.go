//go:build linux

package gotool

import (
	"os"
	"os/exec"
)

// commandBuilt runs the linked image. The child sees the memfd as
// descriptor 3, and exec looks that descriptor up after the fork.
func commandBuilt(_ string, bin *os.File) *exec.Cmd {
	cmd := exec.Command("/proc/self/fd/3")
	cmd.ExtraFiles = []*os.File{bin}
	return cmd
}
