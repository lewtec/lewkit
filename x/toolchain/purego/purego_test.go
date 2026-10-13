package purego

import (
	"os"
	"runtime"
	"testing"
)

func TestOpenLibcGetpid(t *testing.T) {
	path, symbol := hostProcessID()
	handle, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	var getpid func() int
	Register(&getpid, handle, symbol)
	if got := getpid(); got != os.Getpid() {
		t.Fatalf("%s %d, process %d", symbol, got, os.Getpid())
	}
}

func hostProcessID() (path, symbol string) {
	switch runtime.GOOS {
	case "darwin":
		return "/usr/lib/libSystem.B.dylib", "getpid"
	case "windows":
		return "kernel32.dll", "GetCurrentProcessId"
	default:
		return "libc.so.6", "getpid"
	}
}
