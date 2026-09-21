package experiments

import "github.com/lewtec/lewkit/x/ndarray"

type rasterArgs struct {
	seed   *ndarray.Tensor[float32]
	width  *ndarray.Tensor[float32]
	height *ndarray.Tensor[float32]
	shape  ndarray.Shape
}

func rasterAt(h, w int, seed *ndarray.Tensor[float32]) (rasterArgs, error) {
	if h < 1 || w < 1 {
		return rasterArgs{}, ndarray.ErrShape
	}
	if seed == nil {
		seed = ndarray.Const(float32(0))
	}
	return rasterArgs{
		seed:   seed,
		width:  ndarray.Const(float32(w)),
		height: ndarray.Const(float32(h)),
		shape:  ndarray.Shape{h, w, 4},
	}, nil
}

func rasterDynamic(seed, width, height *ndarray.Tensor[float32]) rasterArgs {
	return rasterArgs{
		seed:   seed,
		width:  width,
		height: height,
		shape:  ndarray.Shape{1, 1, 4},
	}
}
