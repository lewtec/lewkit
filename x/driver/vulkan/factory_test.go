package vulkan

import (
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	ffivulkan "github.com/lewtec/lewkit/x/ffi/vulkan"
	"github.com/stretchr/testify/require"
)

func TestOfferID(t *testing.T) {
	infos := []ffivulkan.Info{
		{Index: 0, Vendor: "amd", Kind: ffivulkan.KindIntegrated},
		{Index: 1, Vendor: "nvidia", Kind: ffivulkan.KindDedicated},
		{Index: 2, Vendor: "llvmpipe", Kind: ffivulkan.KindSoftware},
	}
	require.Equal(t, "vulkan:amd:integrated", offerID(infos, 0))
	require.Equal(t, "vulkan:nvidia:dedicated", offerID(infos, 1))
	require.Equal(t, "vulkan:llvmpipe:software", offerID(infos, 2))

	two := []ffivulkan.Info{
		{Vendor: "amd", Kind: ffivulkan.KindDedicated},
		{Vendor: "amd", Kind: ffivulkan.KindDedicated},
	}
	require.Equal(t, "vulkan:amd:dedicated:0", offerID(two, 0))
	require.Equal(t, "vulkan:amd:dedicated:1", offerID(two, 1))
}

func TestOfferWeight(t *testing.T) {
	require.Equal(t, 50, offerWeight(ffivulkan.Info{Kind: ffivulkan.KindIntegrated}))
	require.Equal(t, 70, offerWeight(ffivulkan.Info{Kind: ffivulkan.KindDedicated}))
	require.Equal(t, 0, offerWeight(ffivulkan.Info{Kind: ffivulkan.KindSoftware}))
	require.Equal(t, 20, offerWeight(ffivulkan.Info{Kind: ffivulkan.KindVirtual}))
}

func TestKindArg(t *testing.T) {
	type args struct {
		Kind cmd.EnumArg[Kind]
	}
	got, err := cmd.Parse[args]("dedicated")
	require.NoError(t, err)
	require.Equal(t, KindDedicated, got.Kind.Value())
	_, err = cmd.Parse[args]("discrete")
	require.ErrorIs(t, err, cmd.ErrInvalidArgument)
}
