package exec

// Cwd returns the working directory of pid.
// Linux reads /proc/<pid>/cwd. Darwin reads the process table.
// The path is the one the kernel reports for that process.
// An unreadable pid returns "", false.
func Cwd(pid int) (string, bool) {
	if pid <= 0 {
		return "", false
	}
	return cwdOf(pid)
}
