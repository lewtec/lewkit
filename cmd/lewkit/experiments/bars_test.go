package experiments

import (
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/driver/vulkan"
	"github.com/stretchr/testify/require"
)

func TestOpenVulkanEvaluator(t *testing.T) {
	if _, err := vulkan.List(t.Context()); err != nil {
		t.Skip(err)
	}
	evaluator, err := openVulkanEvaluator(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, evaluator.Close()) })
	named, ok := evaluator.(interface{ Name() string })
	require.True(t, ok)
	require.True(t, strings.HasPrefix(named.Name(), "vulkan:"), named.Name())
}
