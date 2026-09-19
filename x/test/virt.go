package test

import (
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// VirtSize is the process virtual size in bytes.
// Skips on platforms where it cannot be read (Windows).
func VirtSize(tb testing.TB) int64 {
	tb.Helper()
	if runtime.GOOS == "windows" {
		tb.Skip("virt size")
	}
	out, err := exec.Command("ps", "-o", "vsz=", "-p", strconv.Itoa(os.Getpid())).Output()
	if err != nil {
		tb.Skip(err)
	}
	kb, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		tb.Fatal(err)
	}
	return kb * 1024
}
