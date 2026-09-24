package coreaudio

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
)

func TestCoreAudioLayout(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip()
	}
	var desc streamDesc
	require.Equal(t, uintptr(40), unsafe.Sizeof(desc))
	require.Equal(t, uintptr(8), unsafe.Offsetof(desc.formatID))
	var addr propAddr
	require.Equal(t, uintptr(12), unsafe.Sizeof(addr))
	require.Equal(t, uint32(0x64657623), fourcc("dev#"))
	require.Equal(t, uint32(0x6C70636D), fourcc("lpcm"))
}
