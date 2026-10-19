//go:build !unix && !windows

package daemon

import "os/exec"

func detach(_ *exec.Cmd) {}
