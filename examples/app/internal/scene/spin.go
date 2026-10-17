package scene

import (
	"math"
	"strconv"

	"github.com/lewtec/lewkit/examples/app/internal/icon"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/http/asset/lucide"
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
	model.minus = spinButton(lucide.NameMinus, fill, ink)
	model.plus = spinButton(lucide.NamePlus, fill, ink)
	label := &gui.Box{
		Width: 96, Height: 56, Radius: 12,
		Fill: &fill, Align: gui.Alignment{0.5, 0.5},
		Child: &gui.Text{
			Value: strconv.FormatFloat(model.scale, 'f', 2, 64),
			Ink:   ink,
		},
	}
	return &gui.Stage{
		Scene: raster,
		Child: &gui.Box{
			Align:   gui.Alignment{X: 0.5, Y: 1},
			Padding: gui.EdgeInsets{Bottom: 36},
			Clip:    true,
			Child:   gui.Row(model.minus, &gui.Box{Width: 12}, label, &gui.Box{Width: 12}, model.plus),
		},
	}
}

func (model *spinModel) bump(delta float64) {
	if model == nil {
		return
	}
	model.scale = math.Round((model.scale+delta)/triangleTurnStep) * triangleTurnStep
}

func spinButton(name lucide.Name, fill, ink gui.RGB) *gui.Box {
	return &gui.Box{
		Width: 56, Height: 56, Radius: 12,
		Fill: &fill, Align: gui.Alignment{0.5, 0.5},
		Child: icon.Node(name, ink, 22),
	}
}
