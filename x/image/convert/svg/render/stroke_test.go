package render

import (
	"testing"

	"github.com/lewtec/lewkit/x/image/convert/svg"
	"github.com/stretchr/testify/require"
)

func TestStrokeRectRingDetectsBox(t *testing.T) {
	r := svg.NewRect().WithWidth(100).WithHeight(80)
	p, ok := flattenRect(r)
	require.True(t, ok)
	rp, ok := strokeRectRing(p, 2)
	require.True(t, ok)
	require.GreaterOrEqual(t, len(rp.segs), 8)
}
