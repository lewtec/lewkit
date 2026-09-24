package experiments

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"math"

	"github.com/lewtec/lewkit/x/cmd"
	_ "github.com/lewtec/lewkit/x/driver/prelude"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/ffi/native/vulkan"
	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/lewtec/lewkit/x/ui/gui"
)

// Compute is `lewkit experiments window compute`.
type Compute struct {
	shader   *cmd.StringArg  `help:"SPIR-V or GLSL path; example.comp if omitted"`
	smoke    cmd.Flag        `long:"smoke" help:"print the 4-byte smoke shader instead of a window"`
	width    cmd.IntArg[int] `long:"width" default:"640" help:"output width"`
	height   cmd.IntArg[int] `long:"height" default:"480" help:"output height"`
	local    cmd.IntArg[int] `long:"local" default:"8" help:"workgroup edge; must match the shader local_size"`
	bindings cmd.IntArg[int] `long:"bindings" default:"2" help:"storage-buffer count (example.comp uses pixels and params)"`
}

func (Compute) Description() string {
	return "run a compute shader in a window (embedded GLSL example by default)"
}

func (c *Compute) Run(ctx context.Context) error {
	if c.smoke.Value() {
		return runSmoke(ctx)
	}
	return runDemo(ctx, c.runWindow)
}

func runSmoke(ctx context.Context) error {
	device, err := vulkan.Open(ctx)
	if err != nil {
		return err
	}
	defer device.Close()
	buf, err := device.Buffer(4)
	if err != nil {
		return err
	}
	defer buf.Close()
	shader, err := device.Shader(ctx, vulkan.SmokeSPIRV(), 1)
	if err != nil {
		return err
	}
	defer shader.Close()
	if err := device.Run(shader, 1, 1, 1, buf); err != nil {
		return err
	}
	out := make([]byte, 4)
	if err := buf.Read(out); err != nil {
		return err
	}
	fmt.Printf("%s: smoke -> %d\n", device.Name(), binary.LittleEndian.Uint32(out))
	return nil
}

func (c *Compute) runWindow(ctx context.Context) error {
	path := ""
	if c.shader != nil {
		path = c.shader.Value()
	}
	spirv, err := loadShader(ctx, path)
	if err != nil {
		return err
	}
	binds := c.bindings.Value()
	local := c.local.Value()
	if path == "" {
		binds = 2
		local = 8
	}
	if binds < 1 {
		return errBindings
	}
	if local < 1 {
		return errLocal
	}
	device, err := vulkan.Open(ctx)
	if err != nil {
		return err
	}
	shader, err := device.Shader(ctx, spirv, binds)
	if err != nil {
		device.Close()
		return err
	}
	title := "lewkit compute"
	if path != "" {
		title = path
	}
	model := &computeModel{
		device: device, shader: shader, local: local, binds: binds,
		size: image.Pt(c.width.Value(), c.height.Value()),
	}
	if err := model.dispatch(); err != nil {
		_ = model.Close()
		return err
	}
	defer model.Close()
	return runGUI(ctx, "compute", gui.Options{
		Title:  title,
		Width:  c.width.Value(),
		Height: c.height.Value(),
	}, model)
}

type computeModel struct {
	device         *vulkan.Device
	shader         *vulkan.Shader
	pixels, params *vulkan.Buffer
	frame          *ndarray.Tensor[float32]
	raw            []byte
	local, binds   int
	size           image.Point
	elapsed        float64
	frameN         uint32
	err            error
}

func (model *computeModel) Init() gui.Cmd { return gui.Tick() }

func (model *computeModel) Update(msg gui.Msg) (gui.Model, gui.Cmd) {
	if model == nil || model.err != nil {
		return model, nil
	}
	switch message := msg.(type) {
	case gui.TickMsg:
		model.elapsed = message.Elapsed.Seconds()
		if message.Size.X > 0 && message.Size.Y > 0 {
			model.size = message.Size
		}
		model.err = model.dispatch()
		return model, gui.Every(message.Period)
	case window.Resize:
		if message.Size.X > 0 && message.Size.Y > 0 {
			model.size = message.Size
			model.err = model.dispatch()
		}
	}
	return model, nil
}

func (model *computeModel) View() gui.Node {
	if model == nil || model.frame == nil {
		return &gui.Box{}
	}
	return &gui.Raster{Pixels: model.frame}
}

func (model *computeModel) dispatch() error {
	width, height := model.size.X, model.size.Y
	if width < 1 || height < 1 || model.device == nil || model.shader == nil {
		return nil
	}
	if model.pixels == nil || len(model.raw) != width*height*4 {
		if model.pixels != nil {
			_ = model.pixels.Close()
			model.pixels = nil
		}
		buffer, err := model.device.Buffer(width * height * 4)
		if err != nil {
			return err
		}
		model.pixels = buffer
		if model.binds >= 2 && model.params == nil {
			params, err := model.device.Buffer(16)
			if err != nil {
				return err
			}
			model.params = params
		}
		model.raw = make([]byte, width*height*4)
		values := make([]float32, width*height*4)
		frame, err := ndarray.New(values, ndarray.Shape{height, width, 4})
		if err != nil {
			return err
		}
		model.frame = frame
	}
	if model.params != nil {
		var packed [16]byte
		binary.LittleEndian.PutUint32(packed[0:], uint32(width))
		binary.LittleEndian.PutUint32(packed[4:], uint32(height))
		binary.LittleEndian.PutUint32(packed[8:], math.Float32bits(float32(model.elapsed)))
		binary.LittleEndian.PutUint32(packed[12:], model.frameN)
		if err := model.params.Write(packed[:]); err != nil {
			return err
		}
	}
	buffers := []*vulkan.Buffer{model.pixels}
	if model.params != nil {
		buffers = append(buffers, model.params)
	}
	groupsX := uint32((width + model.local - 1) / model.local)
	groupsY := uint32((height + model.local - 1) / model.local)
	if err := model.device.Run(model.shader, groupsX, groupsY, 1, buffers...); err != nil {
		return err
	}
	if err := model.pixels.Read(model.raw); err != nil {
		return err
	}
	values := model.frame.Buffer()
	for i, pixel := range model.raw {
		values[i] = float32(pixel)
	}
	model.frameN++
	return nil
}

func (model *computeModel) Close() error {
	if model == nil {
		return nil
	}
	var err error
	if model.pixels != nil {
		err = model.pixels.Close()
	}
	if model.params != nil {
		err = errors.Join(err, model.params.Close())
	}
	if model.shader != nil {
		err = errors.Join(err, model.shader.Close())
	}
	if model.device != nil {
		err = errors.Join(err, model.device.Close())
	}
	return err
}

var (
	errBindings = errors.New("bindings must be >= 1")
	errLocal    = errors.New("local must be >= 1")
)
