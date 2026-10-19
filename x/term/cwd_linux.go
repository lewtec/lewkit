//go:build linux

package term

import "os"

func processCwd(pid int) (string, bool) {
	if pid <= 0 {
		return "", false
	}
	target, err := os.Readlink("/proc/" + itoa(pid) + "/cwd")
	if err != nil || target == "" {
		return "", false
	}
	return target, true
}
