//go:build windows && (amd64 || arm64)

package std

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
)

func TestMsgMatchesWindowsLayout(t *testing.T) {
	var message msg
	require.Equal(t, uintptr(36), unsafe.Offsetof(message.ptX))
	require.Equal(t, uintptr(48), unsafe.Sizeof(message))
}
