package experiments

import (
	"math"
	"strconv"

	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/ui/gui"
)

type spinModel struct {
	*frameModel
	minus, plus *gui.Box
}

func newSpinModel(scale float64, width, height int, build frameBuild) (*spinModel, error) {
	frame, err := newFrameModel(scale, width, height, build)
	if err != nil {
		return nil, err
	}
	return &spinModel{frameModel: frame}, nil
}

func (model *spinModel) Update(msg gui.Msg) (gui.Model, gui.Cmd) {
	if model == nil || model.frameModel == nil {
		return model, nil
	}
	if pointer, ok := msg.(window.Pointer); ok && pointer.Button == 1 && pointer.Pressed {
		if model.minus != nil && model.minus.Contains(pointer.Pos) {
			model.bump(-triangleTurnStep)
		}
		if model.plus != nil && model.plus.Contains(pointer.Pos) {
			model.bump(triangleTurnStep)
		}
	}
	_, cmd := model.frameModel.Update(msg)
	return model, cmd
}

func (model *spinModel) View() gui.Node {
	if model == nil || model.frameModel == nil {
		return nil
	}
	raster := model.frameModel.View()
	ink := gui.RGB{255, 255, 255, 255}
	fill := gui.RGB{16, 16, 16, 210}
	model.minus = spinButton("-", fill, ink)
	model.plus = spinButton("+", fill, ink)
	label := &gui.Box{
		Width: 96, Height: 56, Radius: 12,
		Fill: &fill, Align: gui.Alignment{0.5, 0.5},
		Child: &gui.Text{
			Value: strconv.FormatFloat(model.scale, 'f', 2, 64),
			Ink:   ink,
		},
	}
	const rowW, rowH float32 = 232, 56
	x := (float32(model.size.X) - rowW) / 2
	y := float32(model.size.Y) - rowH - 36
	if y < 16 {
		y = 16
	}
	return &gui.Stack{Children: []gui.Node{
		raster,
		&gui.Positioned{
			X: x, Y: y,
			Child: gui.Row(model.minus, &gui.Box{Width: 12}, label, &gui.Box{Width: 12}, model.plus),
		},
	}}
}

func (model *spinModel) bump(delta float64) {
	if model == nil {
		return
	}
	model.scale = math.Round((model.scale+delta)/triangleTurnStep) * triangleTurnStep
}

func spinButton(label string, fill, ink gui.RGB) *gui.Box {
	return &gui.Box{
		Width: 56, Height: 56, Radius: 12,
		Fill: &fill, Align: gui.Alignment{0.5, 0.5},
		Child: &gui.Text{Value: label, Ink: ink},
	}
}
