package ndarray

import (
	"testing"

	"github.com/lewtec/lewkit/x/ffi/vulkan"
	"github.com/stretchr/testify/require"
)

func TestVulkanOfferID(t *testing.T) {
	infos := []vulkan.Info{
		{Index: 0, Vendor: "amd"},
		{Index: 1, Vendor: "nvidia"},
		{Index: 2, Vendor: "llvmpipe"},
	}
	require.Equal(t, "ndarray_vulkan:amd", vulkanOfferID(infos, 0))
	require.Equal(t, "ndarray_vulkan:nvidia", vulkanOfferID(infos, 1))
	require.Equal(t, "ndarray_vulkan:llvmpipe", vulkanOfferID(infos, 2))

	two := []vulkan.Info{{Vendor: "amd"}, {Vendor: "amd"}}
	require.Equal(t, "ndarray_vulkan:amd:0", vulkanOfferID(two, 0))
	require.Equal(t, "ndarray_vulkan:amd:1", vulkanOfferID(two, 1))
}

func TestVulkanOfferWeight(t *testing.T) {
	require.Equal(t, 50, vulkanOfferWeight(vulkan.Info{Vendor: "amd"}))
	require.Equal(t, 50, vulkanOfferWeight(vulkan.Info{Vendor: "nvidia"}))
	require.Equal(t, 0, vulkanOfferWeight(vulkan.Info{Vendor: "llvmpipe"}))
	require.Equal(t, 0, vulkanOfferWeight(vulkan.Info{Vendor: "cpu"}))
}
