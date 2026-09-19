package vulkan

import (
	"testing"

	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestListAndOpen(t *testing.T) {
	handles, err := List(t.Context())
	if err != nil {
		t.Skip(err)
	}
	require.NotEmpty(t, handles)
	gpu, err := handles[0].Open(t.Context())
	require.NoError(t, err)
	test.CloseOnCleanup(t, gpu)
	require.NotEmpty(t, gpu.Name())
	require.NotEmpty(t, gpu.Vendor())
	require.True(t, gpu.Type() <= DeviceTypeSoftware)
	require.NotNil(t, gpu.Native())
}
