package experiments

import (
	"image"
	"time"

	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/lewtec/lewkit/x/ui/gui"
)

type frameModel struct {
	scale                float64
	param, width, height *ndarray.Tensor[float32]
	pixels               *ndarray.Tensor[float32]
	elapsed              time.Duration
	size                 image.Point
}

func newFrameModel(scale float64, width, height int, build frameBuild) (*frameModel, error) {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	param, err := ndarray.New([]float32{0}, nil)
	if err != nil {
		return nil, err
	}
	widthTensor, err := ndarray.New([]float32{float32(width)}, nil)
	if err != nil {
		return nil, err
	}
	heightTensor, err := ndarray.New([]float32{float32(height)}, nil)
	if err != nil {
		return nil, err
	}
	pixels, err := build(param, widthTensor, heightTensor)
	if err != nil {
		return nil, err
	}
	return &frameModel{
		scale: scale, param: param, width: widthTensor, height: heightTensor, pixels: pixels,
		size: image.Pt(width, height),
	}, nil
}

func (model *frameModel) Init() gui.Cmd { return gui.Tick() }

func (model *frameModel) Update(msg gui.Msg) (gui.Model, gui.Cmd) {
	if model == nil {
		return model, nil
	}
	switch message := msg.(type) {
	case gui.TickMsg:
		model.elapsed = message.Elapsed
		if message.Size.X > 0 && message.Size.Y > 0 {
			model.size = message.Size
		}
		model.write()
		return model, gui.Every(message.Period)
	case window.Resize:
		if message.Size.X > 0 && message.Size.Y > 0 {
			model.size = message.Size
			model.write()
		}
	}
	return model, nil
}

func (model *frameModel) View() gui.Node {
	if model == nil || model.pixels == nil {
		return nil
	}
	return &gui.Raster{Pixels: model.pixels}
}

func (model *frameModel) write() {
	if model == nil {
		return
	}
	if buf := model.param.Buffer(); len(buf) > 0 {
		buf[0] = float32(model.elapsed.Seconds() * model.scale)
	}
	if model.size.X < 1 || model.size.Y < 1 {
		return
	}
	if buf := model.width.Buffer(); len(buf) > 0 {
		buf[0] = float32(model.size.X)
	}
	if buf := model.height.Buffer(); len(buf) > 0 {
		buf[0] = float32(model.size.Y)
	}
}
