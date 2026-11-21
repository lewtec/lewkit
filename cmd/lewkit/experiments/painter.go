package experiments

import (
	"context"
	"errors"
	stdimage "image"

	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/ndarray"
)

type framePainter struct {
	evaluator ndarray.Evaluator
	pixels    *ndarray.Tensor[uint8]
	param     *ndarray.Tensor[float32]
	width     *ndarray.Tensor[float32]
	height    *ndarray.Tensor[float32]
	scale     float64
}

type frameBuild func(param, width, height *ndarray.Tensor[float32]) (*ndarray.Tensor[float32], error)

func newFramePainter(ctx context.Context, scale float64, build frameBuild) (*framePainter, error) {
	param, err := ndarray.New([]float32{0}, nil)
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
	img, err := build(param, width, height)
	if err != nil {
		return nil, err
	}
	evaluator, err := ndarray.Open(ctx)
	if err != nil {
		return nil, err
	}
	return &framePainter{
		evaluator: evaluator,
		pixels:    img.Cast[uint8](),
		param:     param,
		width:     width,
		height:    height,
		scale:     scale,
	}, nil
}

func (p *framePainter) Draw(ctx context.Context, destination *stdimage.RGBA, elapsed float64) error {
	if p == nil || p.pixels == nil {
		return ndarray.ErrOp
	}
	frameHeight, frameWidth := destination.Rect.Dy(), destination.Rect.Dx()
	if frameHeight < 1 || frameWidth < 1 {
		return nil
	}
	if buf := p.param.Buffer(); len(buf) > 0 {
		buf[0] = float32(elapsed * p.scale)
	}
	if buf := p.width.Buffer(); len(buf) > 0 {
		buf[0] = float32(frameWidth)
	}
	if buf := p.height.Buffer(); len(buf) > 0 {
		buf[0] = float32(frameHeight)
	}
	return window.Present(ctx, p.pixels, p.evaluator, destination)
}

func (p *framePainter) Close() error {
	if p == nil {
		return nil
	}
	var err error
	if p.pixels != nil {
		err = p.pixels.Close()
		p.pixels = nil
	}
	if p.evaluator != nil {
		err = errors.Join(err, p.evaluator.Close())
		p.evaluator = nil
	}
	return err
}
