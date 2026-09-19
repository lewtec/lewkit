package vulkan

import (
	"testing"

	ffivulkan "github.com/lewtec/lewkit/x/ffi/vulkan"
	"github.com/stretchr/testify/require"
)

func TestOfferID(t *testing.T) {
	infos := []ffivulkan.Info{
		{Index: 0, Vendor: "amd"},
		{Index: 1, Vendor: "nvidia"},
		{Index: 2, Vendor: "llvmpipe"},
	}
	require.Equal(t, "vulkan:amd", offerID(infos, 0))
	require.Equal(t, "vulkan:nvidia", offerID(infos, 1))
	require.Equal(t, "vulkan:llvmpipe", offerID(infos, 2))

	two := []ffivulkan.Info{{Vendor: "amd"}, {Vendor: "amd"}}
	require.Equal(t, "vulkan:amd:0", offerID(two, 0))
	require.Equal(t, "vulkan:amd:1", offerID(two, 1))
}

func TestOfferWeight(t *testing.T) {
	require.Equal(t, 50, offerWeight(ffivulkan.Info{Vendor: "amd"}))
	require.Equal(t, 50, offerWeight(ffivulkan.Info{Vendor: "nvidia"}))
	require.Equal(t, 0, offerWeight(ffivulkan.Info{Vendor: "llvmpipe"}))
	require.Equal(t, 0, offerWeight(ffivulkan.Info{Vendor: "cpu"}))
}
