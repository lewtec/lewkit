package test

import (
	"os"
	"testing"

	"github.com/shirou/gopsutil/v4/process"
)

// VirtSize is the process virtual size in bytes.
func VirtSize(tb testing.TB) int64 {
	tb.Helper()
	proc, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		tb.Fatal(err)
	}
	info, err := proc.MemoryInfo()
	if err != nil {
		tb.Fatal(err)
	}
	return int64(info.VMS)
}
