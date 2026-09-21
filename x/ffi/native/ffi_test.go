package native

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenAndSymbol(t *testing.T) {
	path := libcPath()
	if path == "" {
		t.Skip("no libc path")
	}
	lib, err := Open(path, Lazy)
	require.NoError(t, err)
	sym, err := Symbol(lib, "malloc")
	require.NoError(t, err)
	require.NotZero(t, sym)
}

func libcPath() string {
	switch runtime.GOOS {
	case "darwin":
		return "/usr/lib/libSystem.B.dylib"
	case "linux":
		return "libc.so.6"
	default:
		return ""
	}
}
