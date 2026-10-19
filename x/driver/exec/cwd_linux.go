//go:build linux

package exec

import (
	"os"
	"strconv"
)

func cwdOf(pid int) (string, bool) {
	target, err := os.Readlink("/proc/" + strconv.Itoa(pid) + "/cwd")
	if err != nil || target == "" {
		return "", false
	}
	return target, true
}
