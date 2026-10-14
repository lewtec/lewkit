package lewtec_logo

import (
	"testing"

	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	test.RenderContains(t, Load().Render, Path)
	require.NotNil(t, Image())
}
