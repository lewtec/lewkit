package ndarray

import (
	"fmt"
	"slices"
)

// view is one address map: offset + sum(coord[i]*strides[i]), clipped by mask.
type view struct {
	shape   []int
	strides []int
	offset  int
	mask    [][2]int
}

func stridesFor(shape []int) []int {
	if len(shape) == 0 {
		return nil
	}
	st := make([]int, len(shape))
	acc := 1
	for i := len(shape) - 1; i >= 0; i-- {
		if shape[i] == 1 {
			st[i] = 0
		} else {
			st[i] = acc
		}
		acc *= shape[i]
	}
	return st
}

func canonStrides(shape, strides []int) []int {
	out := slices.Clone(strides)
	for i, s := range shape {
		if s == 1 {
			out[i] = 0
		}
	}
	return out
}

func hasZero(shape []int) bool {
	return slices.Contains(shape, 0)
}

func prod(shape []int) int {
	n := 1
	for _, s := range shape {
		n *= s
	}
	return n
}

func create(in view) (view, error) {
	shape, strides, offset, mask := in.shape, in.strides, in.offset, in.mask
	for _, s := range shape {
		if s < 0 {
			return view{}, fmt.Errorf("%w: negative dim %v", ErrShape, shape)
		}
	}
	if strides == nil {
		strides = stridesFor(shape)
	} else {
		if len(strides) != len(shape) {
			return view{}, fmt.Errorf("%w: strides %v for %v", ErrShape, strides, shape)
		}
		strides = canonStrides(shape, strides)
	}
	if hasZero(shape) {
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
	return prod(v.shape)
}

func (v view) contiguous() bool {
	if v.offset != 0 || v.mask != nil {
		return false
	}
	if hasZero(v.shape) {
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

func unravel(shape []int, i int) []int {
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
	if !isPerm(axes, len(v.shape)) {
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

func isPerm(axes []int, n int) bool {
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

func (v view) expand(shape []int) (view, error) {
	if len(shape) != len(v.shape) {
		return view{}, fmt.Errorf("%w: %v into %v", ErrExpand, v.shape, shape)
	}
	for i, s := range v.shape {
		if s != shape[i] && s != 1 {
			return view{}, fmt.Errorf("%w: %v into %v", ErrExpand, v.shape, shape)
		}
	}
	if hasZero(v.shape) {
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
		nmask := make([][2]int, len(arg))
		for i, a := range arg {
			mx, my := v.mask[i][0], v.mask[i][1]
			nmask[i] = [2]int{
				max(0, min(mx-a[0], a[1]-a[0])),
				max(0, min(my-a[0], a[1]-a[0])),
			}
		}
		if mask != nil {
			for i := range nmask {
				nmask[i] = [2]int{
					max(nmask[i][0], mask[i][0]),
					min(nmask[i][1], mask[i][1]),
				}
			}
		}
		mask = nmask
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

func mergeDims(shape, strides []int, mask [][2]int) []merged {
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
		s, st := shape[i], strides[i]
		if s == 1 {
			continue
		}
		last := ret[len(ret)-1]
		if merging || last.stride == s*st {
			pre := last.real * s
			if merging {
				pre = s
			}
			ret[len(ret)-1] = merged{last.size * s, st, pre}
		} else {
			ret = append(ret, merged{s, st, s})
		}
		if mask != nil {
			merging = mask[i][1]-mask[i][0] == 1
		} else {
			merging = s == 1
		}
	}
	return ret
}

func reshapeMask(mask [][2]int, oldShape, newShape []int) ([][2]int, bool) {
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
	pop := func(s *[]int, def int) int {
		if len(*s) == 0 {
			return def
		}
		v := (*s)[0]
		*s = (*s)[1:]
		return v
	}
	popM := func(s *[][2]int) [2]int {
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
	m := popM(&rMasks)
	for len(out) < len(newShape) {
		l, r := m[0], m[1]
		next := newDim * curr
		switch {
		case oldDim == next:
			out = append(out, [2]int{l / curr, (r-1)/curr + 1})
			curr = 1
			oldDim = pop(&rShape, 1)
			newDim = pop(&rNew, 1)
			m = popM(&rMasks)
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
			nextM := popM(&rMasks)
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

func (v view) reshape(newShape []int) (view, bool) {
	if slices.Equal(v.shape, newShape) {
		return v, true
	}
	if hasZero(v.shape) {
		nv, err := create(view{shape: newShape})
		return nv, err == nil
	}
	if len(newShape) == 0 && v.mask != nil {
		for _, m := range v.mask {
			if m[0] == m[1] {
				return view{}, false
			}
		}
	}
	if v.contiguous() {
		nv, err := create(view{shape: newShape})
		return nv, err == nil
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
	nv, err := create(view{shape: newShape, strides: newStrides, offset: v.offset + extra, mask: newMask})
	return nv, err == nil
}

func reversed(s []int) []int {
	out := slices.Clone(s)
	slices.Reverse(out)
	return out
}
