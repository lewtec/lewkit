package vulkan

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDrawSPIRV(t *testing.T) {
	for _, spv := range [][]byte{fillVertSPIRV, fillFragSPIRV, inkVertSPIRV, inkFragSPIRV} {
		require.GreaterOrEqual(t, len(spv), 20)
		require.Zero(t, len(spv)%4)
		require.Equal(t, uint32(0x07230203), binary.LittleEndian.Uint32(spv[:4]))
	}
}
