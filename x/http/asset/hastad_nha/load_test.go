package hastad_nha

import (
	"testing"

	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	test.RenderContains(t, Load().Render, Path)
	require.Equal(t, []byte("ID3"), Bytes()[:3])
}
