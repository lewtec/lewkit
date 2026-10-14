package vulkan

import (
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	drvvulkan "github.com/lewtec/lewkit/x/driver/vulkan"
	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestKindIDsFollowDeviceTypes(t *testing.T) {
	var zero drvvulkan.DeviceType
	for _, kind := range zero.Values() {
		got := kindFactory{kind: kind.String()}
		require.Equal(t, "ndeval_vulkan:"+kind.String(), got.ID())
		require.Equal(t, 0, got.Weight())
	}
	require.Equal(t, 50, gpuFactory{}.Weight())
}

func TestPickDevice(t *testing.T) {
	handles := []driver.Handle[drvvulkan.Device]{
		{ID: "vulkan:nvidia:dedicated"},
		{ID: "vulkan:mesa:software"},
		{ID: "vulkan:amd:integrated"},
	}
	got, err := pickDevice(handles, "integrated")
	require.NoError(t, err)
	require.Equal(t, "vulkan:amd:integrated", got.ID)

	got, err = pickDevice(handles[:2], "integrated")
	require.NoError(t, err)
	require.Equal(t, "vulkan:nvidia:dedicated", got.ID)

	_, err = pickDevice(handles[1:2], "integrated")
	require.EqualError(t, err, "no integrated vulkan device")

	_, err = pickDevice(handles, "virtual")
	require.EqualError(t, err, "no virtual vulkan device")

	_, err = pickDevice(handles, "nope")
	require.Error(t, err)
}

func TestListKeepsWeight50VulkanAheadOfKinds(t *testing.T) {
	ctx := t.Context()
	if _, err := drvvulkan.List(ctx); err != nil {
		test.Skip(t, err)
	}
	handles, err := driver.List[ndarray.Evaluator](ctx)
	require.NoError(t, err)
	var saw bool
	for _, handle := range handles {
		if !strings.HasPrefix(handle.ID, "ndeval_vulkan") {
			continue
		}
		require.Equal(t, "ndeval_vulkan", handle.ID)
		saw = true
		break
	}
	require.True(t, saw)
}
