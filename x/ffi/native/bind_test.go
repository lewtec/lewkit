package native_test

import (
	"runtime"
	"testing"

	"github.com/lewtec/lewkit/x/ffi/native"
	"github.com/stretchr/testify/require"
)

func TestBindFillsStructFields(t *testing.T) {
	path := ""
	switch runtime.GOOS {
	case "darwin":
		path = "/usr/lib/libSystem.B.dylib"
	case "linux":
		path = "libc.so.6"
	default:
		t.Skip("no libc path")
	}
	lib, err := native.Open(path, native.Lazy)
	require.NoError(t, err)

	type libc struct {
		Strlen  func(s *byte) uintptr   `sym:"strlen"`
		Malloc  func(n uintptr) uintptr `sym:"malloc,soft"`
		Missing func()                  `sym:"lewkit_missing_symbol,soft"`
	}
	got, err := native.Bind[libc](lib)
	require.NoError(t, err)
	require.NotNil(t, got.Strlen)
	require.NotNil(t, got.Malloc)
	require.Nil(t, got.Missing)
	text := append([]byte("hi"), 0)
	require.Equal(t, uintptr(2), got.Strlen(&text[0]))

	type missing struct {
		Nope func() `sym:"lewkit_missing_symbol"`
	}
	_, err = native.Bind[missing](lib)
	require.Error(t, err)
}

func TestBindRejectsANonStruct(t *testing.T) {
	_, err := native.Bind[int](0)
	require.Error(t, err)

	type bare struct {
		F func()
	}
	_, err = native.Bind[bare](0)
	require.Error(t, err)

	type bad struct {
		F func() `sym:"strlen,maybe"`
	}
	_, err = native.Bind[bad](0)
	require.Error(t, err)
}
