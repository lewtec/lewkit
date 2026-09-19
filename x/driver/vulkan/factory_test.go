package vulkan

import (
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	ffivulkan "github.com/lewtec/lewkit/x/ffi/vulkan"
	"github.com/stretchr/testify/require"
)

func TestOfferID(t *testing.T) {
	infos := []ffivulkan.Info{
		{Index: 0, Vendor: "amd", Type: ffivulkan.DeviceTypeIntegrated},
		{Index: 1, Vendor: "nvidia", Type: ffivulkan.DeviceTypeDedicated},
		{Index: 2, Vendor: "llvmpipe", Type: ffivulkan.DeviceTypeSoftware},
	}
	require.Equal(t, "vulkan:amd:integrated", offerID(infos, 0))
	require.Equal(t, "vulkan:nvidia:dedicated", offerID(infos, 1))
	require.Equal(t, "vulkan:llvmpipe:software", offerID(infos, 2))

	two := []ffivulkan.Info{
		{Vendor: "amd", Type: ffivulkan.DeviceTypeDedicated},
		{Vendor: "amd", Type: ffivulkan.DeviceTypeDedicated},
	}
	require.Equal(t, "vulkan:amd:dedicated:0", offerID(two, 0))
	require.Equal(t, "vulkan:amd:dedicated:1", offerID(two, 1))
}

func TestOfferWeight(t *testing.T) {
	require.Equal(t, 50, offerWeight(ffivulkan.Info{Type: ffivulkan.DeviceTypeIntegrated}))
	require.Equal(t, 70, offerWeight(ffivulkan.Info{Type: ffivulkan.DeviceTypeDedicated}))
	require.Equal(t, 0, offerWeight(ffivulkan.Info{Type: ffivulkan.DeviceTypeSoftware}))
	require.Equal(t, 20, offerWeight(ffivulkan.Info{Type: ffivulkan.DeviceTypeVirtual}))
}

func TestDeviceTypeArg(t *testing.T) {
	type args struct {
		Type cmd.EnumArg[DeviceType]
	}
	got, err := cmd.Parse[args]("dedicated")
	require.NoError(t, err)
	require.Equal(t, DeviceTypeDedicated, got.Type.Value())
	parsed, err := ParseDeviceType("software")
	require.NoError(t, err)
	require.Equal(t, DeviceTypeSoftware, parsed)
	require.Equal(t, uint32(4), parsed.Uint32())
	_, err = cmd.Parse[args]("discrete")
	require.ErrorIs(t, err, cmd.ErrInvalidArgument)
	_, err = ParseDeviceType("discrete")
	require.ErrorIs(t, err, ffivulkan.ErrDeviceType)
}
