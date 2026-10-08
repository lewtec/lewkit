package d3d12

import "fmt"

// pushWords is the compute root-constant count: n, d0..d3, then span.
// span is groupsX * threads. The HLSL linear id is tid.y * span + tid.x.
const pushWords = 6

// DispatchGrid splits a linear workgroup count into Direct3D 12's
// per-dimension limit. X is the full count until that limit; further
// groups stack on Y. Both dimensions stay within the limit, and X*Y
// covers every group. A one-row grid leaves SV_DispatchThreadID.y at 0,
// so the linear id matches the old tid.x dispatch.
func DispatchGrid(groups int) (x, y uint32, err error) {
	if groups < 1 {
		return 0, 0, nil
	}
	g := int64(groups)
	limit := int64(maxGroup)
	if g <= limit {
		return uint32(groups), 1, nil
	}
	if g > limit*limit {
		return 0, 0, fmt.Errorf("%w: groups", ErrSize)
	}
	yv := (g + limit - 1) / limit
	xv := (g + yv - 1) / yv
	return uint32(xv), uint32(yv), nil
}
