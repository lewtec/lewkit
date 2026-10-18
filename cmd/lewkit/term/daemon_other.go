//go:build !unix && !windows

package term

import "os/exec"

func detach(_ *exec.Cmd) {}
