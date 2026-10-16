package render

import (
	"testing"

	"github.com/lewtec/lewkit/x/image/convert/svg"
	"github.com/stretchr/testify/require"
)

func TestRoundCapPaintsDot(t *testing.T) {
	t.Parallel()
	img, err := Decode([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24"><path d="M12 20h.01" fill="none" stroke="#000000" stroke-width="2" stroke-linecap="round"/></svg>`))
	require.NoError(t, err)
	_, _, _, a := img.At(12, 20).RGBA()
	require.NotZero(t, a)
}

func TestStrokeRectRingDetectsBox(t *testing.T) {
	r := svg.NewRect().WithWidth(100).WithHeight(80)
	p, ok := flattenRect(r)
	require.True(t, ok)
	rp, ok := strokeRectRing(p, 2)
	require.True(t, ok)
	require.GreaterOrEqual(t, len(rp.segs), 8)
}
