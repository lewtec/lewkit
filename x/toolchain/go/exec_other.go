//go:build !linux

package gotool

import (
	"os"
	"os/exec"
)

func commandBuilt(path string, _ *os.File) *exec.Cmd {
	return exec.Command(path)
}
