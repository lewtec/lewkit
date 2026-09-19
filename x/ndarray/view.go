package ndarray

import (
	"fmt"
	"slices"
)

// view is one address map: offset + sum(coord[i]*strides[i]), clipped by mask.
type view struct {
	shape   Shape
	strides []int
	offset  int
	mask    [][2]int
}

func stridesFor(shape Shape) []int {
	if len(shape) == 0 {
		return nil
	}
	strides := make([]int, len(shape))
	acc := 1
	for i := len(shape) - 1; i >= 0; i-- {
		if shape[i] == 1 {
			strides[i] = 0
		} else {
			strides[i] = acc
		}
		acc *= shape[i]
	}
	return strides
}

func canonicalStrides(shape Shape, strides []int) []int {
	out := slices.Clone(strides)
	for i, dim := range shape {
		if dim == 1 {
			out[i] = 0
		}
	}
	return out
}

func create(in view) (view, error) {
	shape, strides, offset, mask := in.shape, in.strides, in.offset, in.mask
	if err := shape.check(); err != nil {
		return view{}, err
	}
	if strides == nil {
		strides = stridesFor(shape)
	} else {
		if len(strides) != len(shape) {
			return view{}, fmt.Errorf("%w: strides %v for %v", ErrShape, strides, shape)
		}
		strides = canonicalStrides(shape, strides)
	}
	if shape.HasZero() {
		return view{
			shape:   slices.Clone(shape),
			strides: zeros(len(shape)),
		}, nil
	}
	if mask != nil {
		if len(mask) != len(shape) {
			return view{}, fmt.Errorf("%w: mask rank %d for %v", ErrShape, len(mask), shape)
		}
		noop := true
		for i, m := range mask {
			if m != [2]int{0, shape[i]} {
				noop = false
				break
			}
		}
		if noop {
			mask = nil
		}
	}
	if mask != nil {
		elim := make([]bool, len(mask))
		empty := false
		anyElim := false
		for i, m := range mask {
			if m[0] >= m[1] {
				empty = true
				elim[i] = true
				anyElim = true
				continue
			}
			if m[0]+1 >= m[1] {
				elim[i] = true
				anyElim = true
			}
		}
		if empty {
			return view{
				shape:   slices.Clone(shape),
				strides: zeros(len(shape)),
				mask:    zerosMask(len(shape)),
			}, nil
		}
		if anyElim {
			for i, e := range elim {
				if e {
					offset += strides[i] * mask[i][0]
					strides[i] = 0
				}
			}
		}
		mask = slices.Clone(mask)
	}
	return view{
		shape:   slices.Clone(shape),
		strides: strides,
		offset:  offset,
		mask:    mask,
	}, nil
}

func zeros(n int) []int {
	if n == 0 {
		return nil
	}
	return make([]int, n)
}

func zerosMask(n int) [][2]int {
	if n == 0 {
		return nil
	}
	return make([][2]int, n)
}

func (v view) size() int {
	return v.shape.Size()
}

func (v view) contiguous() bool {
	if v.offset != 0 || v.mask != nil {
		return false
	}
	if v.shape.HasZero() {
		return true
	}
	return slices.Equal(v.strides, stridesFor(v.shape))
}

func (v view) index(coords []int) (int, bool) {
	off := v.offset
	ok := true
	for i, c := range coords {
		off += c * v.strides[i]
		if v.mask != nil && (c < v.mask[i][0] || c >= v.mask[i][1]) {
			ok = false
		}
	}
	return off, ok
}

func unravel(shape Shape, i int) []int {
	coords := make([]int, len(shape))
	acc := 1
	for d := len(shape) - 1; d >= 0; d-- {
		s := shape[d]
		if s == 0 {
			coords[d] = 0
			continue
		}
		coords[d] = (i / acc) % s
		acc *= s
	}
	return coords
}

func (v view) permute(axes []int) (view, error) {
	if !isPermutation(axes, len(v.shape)) {
		return view{}, fmt.Errorf("%w: permute %v of rank %d", ErrAxis, axes, len(v.shape))
	}
	shape := make([]int, len(axes))
	strides := make([]int, len(axes))
	var mask [][2]int
	if v.mask != nil {
		mask = make([][2]int, len(axes))
	}
	for i, a := range axes {
		shape[i] = v.shape[a]
		strides[i] = v.strides[a]
		if mask != nil {
			mask[i] = v.mask[a]
		}
	}
	return create(view{shape: shape, strides: strides, offset: v.offset, mask: mask})
}

func isPermutation(axes []int, n int) bool {
	if len(axes) != n {
		return false
	}
	seen := make([]bool, n)
	for _, a := range axes {
		if a < 0 || a >= n || seen[a] {
			return false
		}
		seen[a] = true
	}
	return true
}

func (v view) expand(shape Shape) (view, error) {
	if len(shape) != len(v.shape) {
		return view{}, fmt.Errorf("%w: %v into %v", ErrExpand, v.shape, shape)
	}
	for i, s := range v.shape {
		if s != shape[i] && s != 1 {
			return view{}, fmt.Errorf("%w: %v into %v", ErrExpand, v.shape, shape)
		}
	}
	if v.shape.HasZero() {
		return create(view{shape: shape})
	}
	var mask [][2]int
	if v.mask != nil {
		mask = make([][2]int, len(shape))
		for i, m := range v.mask {
			if v.shape[i] != shape[i] {
				if m != [2]int{0, 1} {
					mask[i] = [2]int{0, 0}
				} else {
					mask[i] = [2]int{0, shape[i]}
				}
			} else {
				mask[i] = m
			}
		}
	}
	return create(view{shape: shape, strides: v.strides, offset: v.offset, mask: mask})
}

func (v view) flip(which []bool) (view, error) {
	if len(which) != len(v.shape) {
		return view{}, fmt.Errorf("%w: flip rank %d for %v", ErrAxis, len(which), v.shape)
	}
	offset := v.offset
	strides := slices.Clone(v.strides)
	var mask [][2]int
	if v.mask != nil {
		mask = make([][2]int, len(v.shape))
	}
	for i, f := range which {
		if !f {
			if mask != nil {
				mask[i] = v.mask[i]
			}
			continue
		}
		offset += (v.shape[i] - 1) * v.strides[i]
		strides[i] = -v.strides[i]
		if mask != nil {
			mask[i] = [2]int{v.shape[i] - v.mask[i][1], v.shape[i] - v.mask[i][0]}
		}
	}
	return create(view{shape: v.shape, strides: strides, offset: offset, mask: mask})
}

func (v view) resize(arg [][2]int, mask [][2]int) (view, error) {
	offset := v.offset
	for i, a := range arg {
		offset += v.strides[i] * a[0]
	}
	if v.mask != nil {
		newMask := make([][2]int, len(arg))
		for i, a := range arg {
			mx, my := v.mask[i][0], v.mask[i][1]
			newMask[i] = [2]int{
				max(0, min(mx-a[0], a[1]-a[0])),
				max(0, min(my-a[0], a[1]-a[0])),
			}
		}
		if mask != nil {
			for i := range newMask {
				newMask[i] = [2]int{
					max(newMask[i][0], mask[i][0]),
					min(newMask[i][1], mask[i][1]),
				}
			}
		}
		mask = newMask
	}
	shape := make([]int, len(arg))
	for i, a := range arg {
		shape[i] = a[1] - a[0]
	}
	return create(view{shape: shape, strides: v.strides, offset: offset, mask: mask})
}

func (v view) pad(arg [][2]int) (view, error) {
	if len(arg) != len(v.shape) {
		return view{}, fmt.Errorf("%w: pad rank %d for %v", ErrPad, len(arg), v.shape)
	}
	changed := false
	for _, a := range arg {
		if a[0] < 0 || a[1] < 0 {
			return view{}, fmt.Errorf("%w: %v", ErrPad, arg)
		}
		if a[0] != 0 || a[1] != 0 {
			changed = true
		}
	}
	if !changed {
		return v, nil
	}
	zv := make([][2]int, len(arg))
	mask := make([][2]int, len(arg))
	for i, a := range arg {
		zv[i] = [2]int{-a[0], v.shape[i] + a[1]}
		mask[i] = [2]int{a[0], v.shape[i] + a[0]}
	}
	return v.resize(zv, mask)
}

func (v view) shrink(arg [][2]int) (view, error) {
	if len(arg) != len(v.shape) {
		return view{}, fmt.Errorf("%w: shrink rank %d for %v", ErrShrink, len(arg), v.shape)
	}
	for i, a := range arg {
		if a[0] < 0 || a[1] < a[0] || a[1] > v.shape[i] {
			return view{}, fmt.Errorf("%w: %v of %v", ErrShrink, arg, v.shape)
		}
	}
	return v.resize(arg, nil)
}

type merged struct {
	size, stride, real int
}

func mergeDims(shape Shape, strides []int, mask [][2]int) []merged {
	if len(shape) == 0 {
		return nil
	}
	real0 := shape[0]
	if strides[0] == 0 {
		real0 = 0
	}
	ret := []merged{{shape[0], strides[0], real0}}
	merging := shape[0] == 1
	if mask != nil {
		merging = mask[0][1]-mask[0][0] == 1
	}
	for i := 1; i < len(shape); i++ {
		size, stride := shape[i], strides[i]
		if size == 1 {
			continue
		}
		last := ret[len(ret)-1]
		if merging || last.stride == size*stride {
			pre := last.real * size
			if merging {
				pre = size
			}
			ret[len(ret)-1] = merged{last.size * size, stride, pre}
		} else {
			ret = append(ret, merged{size, stride, size})
		}
		if mask != nil {
			merging = mask[i][1]-mask[i][0] == 1
		} else {
			merging = size == 1
		}
	}
	return ret
}

func reshapeMask(mask [][2]int, oldShape, newShape Shape) ([][2]int, bool) {
	if mask == nil {
		out := make([][2]int, len(newShape))
		for i, s := range newShape {
			out[i] = [2]int{0, s}
		}
		return out, true
	}
	if len(newShape) == 0 {
		return nil, true
	}
	rMasks := slices.Clone(mask)
	slices.Reverse(rMasks)
	rShape := slices.Clone(oldShape)
	slices.Reverse(rShape)
	rNew := slices.Clone(newShape)
	slices.Reverse(rNew)
	pop := func(s *Shape, def int) int {
		if len(*s) == 0 {
			return def
		}
		v := (*s)[0]
		*s = (*s)[1:]
		return v
	}
	popMask := func(s *[][2]int) [2]int {
		if len(*s) == 0 {
			return [2]int{0, 1}
		}
		v := (*s)[0]
		*s = (*s)[1:]
		return v
	}
	var out [][2]int
	curr := 1
	oldDim := pop(&rShape, 1)
	newDim := pop(&rNew, 1)
	m := popMask(&rMasks)
	for len(out) < len(newShape) {
		l, r := m[0], m[1]
		next := newDim * curr
		switch {
		case oldDim == next:
			out = append(out, [2]int{l / curr, (r-1)/curr + 1})
			curr = 1
			oldDim = pop(&rShape, 1)
			newDim = pop(&rNew, 1)
			m = popMask(&rMasks)
		case oldDim > next:
			if next == 0 || oldDim%next != 0 {
				return nil, false
			}
			if (l%next != 0 || r%next != 0) && l/next != (r-1)/next {
				return nil, false
			}
			out = append(out, [2]int{(l % next) / curr, ((r-1)%next)/curr + 1})
			curr = next
			newDim = pop(&rNew, 1)
		default:
			nextM := popMask(&rMasks)
			if m != [2]int{0, oldDim} && l != r && nextM[1]-nextM[0] != 1 {
				return nil, false
			}
			m = [2]int{nextM[0]*oldDim + l, (nextM[1]-1)*oldDim + r}
			oldDim *= pop(&rShape, 1)
		}
	}
	slices.Reverse(out)
	return out, true
}

func (v view) reshape(newShape Shape) (view, bool) {
	if slices.Equal(v.shape, newShape) {
		return v, true
	}
	if v.shape.HasZero() {
		newView, err := create(view{shape: newShape})
		return newView, err == nil
	}
	if len(newShape) == 0 && v.mask != nil {
		for _, m := range v.mask {
			if m[0] == m[1] {
				return view{}, false
			}
		}
	}
	if v.contiguous() {
		newView, err := create(view{shape: newShape})
		return newView, err == nil
	}
	rStrides := make([]int, 0, len(newShape))
	rNew := slices.Clone(newShape)
	slices.Reverse(rNew)
	ri := 0
	for _, md := range slices.Backward(mergeDims(v.shape, v.strides, v.mask)) {
		acc := 1
		for acc <= md.size && acc != md.size {
			if ri >= len(rNew) || rNew[ri] <= 0 {
				break
			}
			newDim := rNew[ri]
			ri++
			rStrides = append(rStrides, md.stride*acc)
			acc *= newDim
			if acc >= md.real {
				md.stride = 0
			}
		}
		if acc != md.size {
			return view{}, false
		}
	}
	newStrides := make([]int, len(newShape))
	copy(newStrides[len(newShape)-len(rStrides):], reversed(rStrides))
	newMask, ok := reshapeMask(v.mask, v.shape, newShape)
	if !ok {
		return view{}, false
	}
	extra := 0
	if v.mask != nil {
		for i, m := range v.mask {
			extra += m[0] * v.strides[i]
		}
	}
	for i, m := range newMask {
		extra -= m[0] * newStrides[i]
	}
	newView, err := create(view{shape: newShape, strides: newStrides, offset: v.offset + extra, mask: newMask})
	return newView, err == nil
}

func reversed(s []int) []int {
	out := slices.Clone(s)
	slices.Reverse(out)
	return out
}
