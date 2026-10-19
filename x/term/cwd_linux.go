//go:build linux

package term

import (
	"os"
	"strconv"
)

func processCwd(pid int) (string, bool) {
	if pid <= 0 {
		return "", false
	}
	target, err := os.Readlink("/proc/" + strconv.Itoa(pid) + "/cwd")
	if err != nil || target == "" {
		return "", false
	}
	return target, true
}
