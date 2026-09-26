package power

import (
	"bytes"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMagicPacket(t *testing.T) {
	hw, err := net.ParseMAC("01:02:03:04:05:06")
	require.NoError(t, err)
	packet, err := magicPacket(hw)
	require.NoError(t, err)
	require.Len(t, packet, 102)
	require.Equal(t, []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}, packet[:6])
	for i := 1; i <= 16; i++ {
		require.True(t, bytes.Equal(packet[i*6:(i+1)*6], hw))
	}
	_, err = magicPacket(net.HardwareAddr{1, 2, 3})
	require.Error(t, err)
}

func TestWakeRejectsMAC(t *testing.T) {
	err := Wake(t.Context(), "not-a-mac")
	require.Error(t, err)
}
