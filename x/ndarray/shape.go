package ndarray

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// Shape is a tensor's dimensions. Rank is len(Shape). A single -1 is allowed
// as a reshape wildcard and is replaced by Infer.
type Shape []int

// Rank is the number of axes.
func (s Shape) Rank() int {
	return len(s)
}

// Size is the number of cells. Live shapes have dims >= 0.
func (s Shape) Size() int {
	n := 1
	for _, dim := range s {
		n *= dim
	}
	return n
}

// Clone is a copy of the dimensions.
func (s Shape) Clone() Shape {
	if s == nil {
		return nil
	}
	return slices.Clone(s)
}

// With appends dims and returns a new shape.
func (s Shape) With(dims ...int) Shape {
	out := make(Shape, len(s)+len(dims))
	copy(out, s)
	copy(out[len(s):], dims)
	return out
}

// WindowLength is the floor count of kernel windows along one axis.
// A negative span is an empty axis.
func WindowLength(in, before, after, kernel, stride int) int {
	if stride < 1 {
		stride = 1
	}
	span := in + before + after - kernel
	if span < 0 {
		return 0
	}
	return span/stride + 1
}

// CoverPad grows after so out strided windows of kernel fit in in+before+after.
func CoverPad(in, kernel, stride, out, before, after int) (int, int) {
	if stride <= 0 {
		stride = 1
	}
	need := kernel - 1 + out*stride
	have := in + before + after
	if need > have {
		after += need - have
	}
	return before, after
}

// PadRank reshapes t to rank by prefixing axes of length 1.
// A rank below len(t.Shape()) is [ErrShape]. Equal rank returns t.
func PadRank[T Number](t *Tensor[T], rank int) (*Tensor[T], error) {
	shape := t.Shape()
	if len(shape) == rank {
		return t, nil
	}
	if len(shape) > rank {
		return nil, ErrShape
	}
	padded := make(Shape, rank)
	copy(padded[rank-len(shape):], shape)
	for i := 0; i < rank-len(shape); i++ {
		padded[i] = 1
	}
	return t.Reshape(padded)
}

// Equal reports the same rank and dimensions.
func (s Shape) Equal(other Shape) bool {
	return slices.Equal(s, other)
}

// HasZero reports a zero-length axis.
func (s Shape) HasZero() bool {
	return slices.Contains(s, 0)
}

// String is [d0 d1 …].
func (s Shape) String() string {
	var b strings.Builder
	b.WriteByte('[')
	for i, dim := range s {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(strconv.Itoa(dim))
	}
	b.WriteByte(']')
	return b.String()
}

// Infer replaces one -1 with size / product(rest). Other dims must be >= 0.
func (s Shape) Infer(size int) (Shape, error) {
	hole := -1
	n := 1
	for i, dim := range s {
		if dim == -1 {
			if hole >= 0 {
				return nil, fmt.Errorf("%w: more than one -1 in %v", ErrShape, s)
			}
			hole = i
			continue
		}
		if dim < 0 {
			return nil, fmt.Errorf("%w: %v", ErrShape, s)
		}
		n *= dim
	}
	out := s.Clone()
	if hole >= 0 {
		if n == 0 {
			if size != 0 {
				return nil, fmt.Errorf("%w: %v of %d", ErrSize, s, size)
			}
			out[hole] = 0
			return out, nil
		}
		if size%n != 0 {
			return nil, fmt.Errorf("%w: %v of %d", ErrSize, s, size)
		}
		out[hole] = size / n
		return out, nil
	}
	if n != size {
		return nil, fmt.Errorf("%w: %v of %d", ErrSize, s, size)
	}
	return out, nil
}

func (s Shape) check() error {
	for _, dim := range s {
		if dim < 0 {
			return fmt.Errorf("%w: negative dim %v", ErrShape, s)
		}
	}
	return nil
}
