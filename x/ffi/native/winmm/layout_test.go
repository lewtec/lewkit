package winmm

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
)

func TestWaveLayout(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip()
	}
	var hdr waveHdr
	require.Equal(t, uintptr(48), unsafe.Sizeof(hdr))
	require.Equal(t, uintptr(0), unsafe.Offsetof(hdr.data))
	require.Equal(t, uintptr(8), unsafe.Offsetof(hdr.length))
	require.Equal(t, uintptr(16), unsafe.Offsetof(hdr.user))
	require.Equal(t, uintptr(24), unsafe.Offsetof(hdr.flags))
	require.Equal(t, uintptr(32), unsafe.Offsetof(hdr.next))
	require.Equal(t, uintptr(40), unsafe.Offsetof(hdr.reserved))

	var format waveFormat
	require.Equal(t, uintptr(0), unsafe.Offsetof(format.tag))
	require.Equal(t, uintptr(2), unsafe.Offsetof(format.channels))
	require.Equal(t, uintptr(4), unsafe.Offsetof(format.rate))
	require.Equal(t, uintptr(8), unsafe.Offsetof(format.avg))
	require.Equal(t, uintptr(12), unsafe.Offsetof(format.block))
	require.Equal(t, uintptr(14), unsafe.Offsetof(format.bits))
	require.Equal(t, uintptr(16), unsafe.Offsetof(format.extra))

	var caps waveCaps
	require.Equal(t, uintptr(84), unsafe.Sizeof(caps))
	require.Equal(t, uintptr(8), unsafe.Offsetof(caps.name))
	require.Equal(t, uintptr(72), unsafe.Offsetof(caps.formats))
	require.Equal(t, "Speakers", utf16z([]uint16{'S', 'p', 'e', 'a', 'k', 'e', 'r', 's', 0, 'x'}))
}
