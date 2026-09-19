package image

import (
	"context"
	"errors"
	stdimage "image"

	"github.com/lewtec/lewkit/x/ndarray"
)

// Painter draws Triangle by resizing a Tensor. The kernel is compiled once.
type Painter struct {
	evaluator   ndarray.Evaluator
	triangle    *ndarray.Tensor
	turn        *ndarray.Tensor
	width       *ndarray.Tensor
	height      *ndarray.Tensor
	buffer      []float32
	frameHeight int
	frameWidth  int
}

// New compiles TriangleDynamic once. Open picks a registered evaluator.
func New(ctx context.Context) (*Painter, error) {
	turn, err := ndarray.New([]float32{0}, nil)
	if err != nil {
		return nil, err
	}
	width, err := ndarray.New([]float32{1}, nil)
	if err != nil {
		return nil, err
	}
	height, err := ndarray.New([]float32{1}, nil)
	if err != nil {
		return nil, err
	}
	triangle, err := TriangleDynamic(turn, width, height)
	if err != nil {
		return nil, err
	}
	evaluator, err := ndarray.Open(ctx)
	if err != nil {
		return nil, err
	}
	return &Painter{
		evaluator: evaluator,
		triangle:  triangle,
		turn:      turn,
		width:     width,
		height:    height,
	}, nil
}

// Draw renders one frame into destination.
func (p *Painter) Draw(ctx context.Context, destination *stdimage.RGBA, turn float64) error {
	if p == nil || p.triangle == nil {
		return ndarray.ErrOp
	}
	frameHeight, frameWidth := destination.Rect.Dy(), destination.Rect.Dx()
	if frameHeight < 1 || frameWidth < 1 {
		return nil
	}
	if buf := p.turn.Buffer(); len(buf) > 0 {
		buf[0] = float32(turn)
	}
	if buf := p.width.Buffer(); len(buf) > 0 {
		buf[0] = float32(frameWidth)
	}
	if buf := p.height.Buffer(); len(buf) > 0 {
		buf[0] = float32(frameHeight)
	}
	if frameHeight != p.frameHeight || frameWidth != p.frameWidth {
		if err := p.triangle.Resize(ndarray.Shape{frameHeight, frameWidth, 4}); err != nil {
			return err
		}
		p.frameHeight, p.frameWidth = frameHeight, frameWidth
	}
	size := frameHeight * frameWidth * 4
	if cap(p.buffer) < size {
		p.buffer = make([]float32, size)
	} else {
		p.buffer = p.buffer[:size]
	}
	if err := p.triangle.Eval(ctx, p.evaluator, p.buffer); err != nil {
		return err
	}
	Write(destination, p.buffer)
	return nil
}

// Close releases the triangle kernel/session and device.
func (p *Painter) Close() error {
	if p == nil {
		return nil
	}
	var err error
	if p.triangle != nil {
		err = p.triangle.Close()
		p.triangle = nil
	}
	if p.evaluator != nil {
		err = errors.Join(err, p.evaluator.Close())
		p.evaluator = nil
	}
	return err
}
