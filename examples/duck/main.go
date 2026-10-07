// Duck turns an embedded mallard scan on the credits card and loops the clip.
//
// The world schedule owns the turn. A tick calls Step with the host
// delta, the next system advances yaw from Time, and the system after
// that samples the scan into a backdrop tensor the size of the window.
// View mounts that tensor as a Raster under the credit nodes. Text and
// fills are the card. The picture pipeline paints the window.
//
//	go run ./cmd/lewkit release run --config ./examples/duck/eletrocromo.json
package main

import (
	"context"
	"image"
	"math"
	"time"

	"github.com/lewtec/lewkit/x/app"
	_ "github.com/lewtec/lewkit/x/driver/prelude"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lewtec/lewkit/x/ndarray"
	_ "github.com/lewtec/lewkit/x/sound/prelude"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/lewkit/x/ui/gui"
	"github.com/lewtec/lewkit/x/ui/world"
)

func init() { entry.Bind(runApp) }

func main() { entry.Main(context.Background(), runApp) }

// turnsPerLoop matches the reference gif: one revolution across 6.8s.
const turnsPerLoop = 1 / 6.8

func runApp(ctx context.Context) error {
	// The clip keeps this context's values and ends when it ends.
	// Closing the window cancels the child before the session waits.
	ctx, stop := startClip(ctx, nil)
	defer stop()
	model, err := newScreen(ctx)
	if err != nil {
		return err
	}
	return app.App{
		Title:   "Duck",
		Width:   996,
		Height:  560,
		Handler: app.GUI(model),
	}.Run(ctx)
}

// startClip schedules play on a child of ctx. stop cancels that child
// and not ctx. A nil play uses the embedded clip.
func startClip(ctx context.Context, play func(context.Context) error) (context.Context, context.CancelFunc) {
	if play == nil {
		play = playLoop
	}
	ctx, stop := context.WithCancel(ctx)
	taskgroup.Go(ctx, "duck", taskgroup.IO, func(ctx context.Context, _ *taskgroup.Status) error {
		return play(ctx)
	})
	return ctx, stop
}

type spin struct {
	Yaw  float64
	Rate float64
}

type screen struct {
	gui.Dirty
	sim    *world.Sim
	mesh   mesh
	frame  *image.RGBA
	pixels *ndarray.Tensor[float32]
	size   image.Point
	seen   time.Duration
	ctx    context.Context
	fault  error
}

func newScreen(ctx context.Context) (*screen, error) {
	model, err := pato()
	if err != nil {
		return nil, err
	}
	sim := world.New()
	world.Init(sim.World, spin{Rate: turnsPerLoop})
	s := &screen{
		sim:  sim,
		mesh: model,
		size: image.Pt(996, 560),
		ctx:  ctx,
	}
	sim.Chain(world.Update, turn, s.sampleSpin)
	sim.Step(ctx, 0)
	if s.fault != nil {
		return nil, s.fault
	}
	return s, nil
}

func turn(_ context.Context, w *world.World) {
	clock, ok := world.Read[world.Time](w)
	sp := world.Mut[spin](w)
	if !ok || sp == nil {
		return
	}
	sp.Yaw += clock.Delta * sp.Rate * 2 * math.Pi
}

func (s *screen) sampleSpin(_ context.Context, w *world.World) {
	if s == nil {
		return
	}
	sp, ok := world.Read[spin](w)
	if !ok {
		s.fault = s.sample(0)
		return
	}
	s.fault = s.sample(sp.Yaw)
}

func (s *screen) Init() gui.Cmd { return gui.Tick() }

func (s *screen) Update(msg gui.Msg) (gui.Model, gui.Cmd) {
	if s == nil {
		return s, nil
	}
	if resized, ok := msg.(window.Resize); ok {
		if resized.Size != s.size {
			s.adopt(resized.Size)
			s.advance(0)
		}
		return s, nil
	}
	tick, ok := msg.(gui.TickMsg)
	if !ok {
		return s, nil
	}
	dt := 0.0
	if tick.Elapsed > s.seen {
		dt = (tick.Elapsed - s.seen).Seconds()
	}
	s.seen = tick.Elapsed
	s.adopt(tick.Size)
	s.advance(dt)
	return s, gui.Every(tick.Period)
}

func (s *screen) adopt(size image.Point) {
	if s == nil || size.X < 2 || size.Y < 2 {
		return
	}
	s.size = size
}

func (s *screen) advance(dt float64) {
	if s == nil || s.sim == nil {
		return
	}
	s.sim.Step(s.ctx, dt)
	if s.fault == nil {
		s.Dirty = gui.Touch(s.Dirty)
	}
}

func (s *screen) View() gui.Node {
	if s == nil || s.pixels == nil {
		return nil
	}
	w, h := s.windowSize()
	return &gui.Stack{Children: []gui.Node{
		&gui.Raster{Pixels: s.pixels},
		cardNodes(float32(w), float32(h)),
	}}
}

// sample writes one scan frame into a backdrop the size of the window.
// The rasterizer stays inside the sample cap; the tensor matches the
// window so the present cast does not wrap the rows. Credit nodes are
// not in this buffer.
func (s *screen) sample(yaw float64) error {
	w, h := s.windowSize()
	rw, rh := fit(w, h, 996, 560)
	if s.frame == nil || s.frame.Bounds().Dx() != rw || s.frame.Bounds().Dy() != rh {
		s.frame = image.NewRGBA(image.Rect(0, 0, rw, rh))
	}
	paintDuck(s.frame, s.mesh, yaw)
	return s.upload(rw, rh, w, h)
}

// upload scales the scan into the picture backdrop.
func (s *screen) upload(srcW, srcH, dstW, dstH int) error {
	if s.pixels == nil || len(s.pixels.Shape()) != 3 || s.pixels.Shape()[0] != dstH || s.pixels.Shape()[1] != dstW {
		if s.pixels != nil {
			_ = s.pixels.Close()
		}
		next, err := ndarray.New(make([]float32, dstW*dstH*4), ndarray.Shape{dstH, dstW, 4})
		if err != nil {
			s.pixels = nil
			return err
		}
		s.pixels = next
	}
	buf := s.pixels.Buffer()
	pix := s.frame.Pix
	stride := s.frame.Stride
	if srcW == dstW && srcH == dstH {
		for y := 0; y < dstH; y++ {
			row := pix[y*stride : y*stride+dstW*4]
			dst := buf[y*dstW*4 : (y+1)*dstW*4]
			for i, p := range row {
				dst[i] = float32(p)
			}
		}
		return nil
	}
	for y := 0; y < dstH; y++ {
		sy := y * srcH / dstH
		row := pix[sy*stride : sy*stride+srcW*4]
		dst := buf[y*dstW*4 : (y+1)*dstW*4]
		for x := 0; x < dstW; x++ {
			o := (x * srcW / dstW) * 4
			i := x * 4
			dst[i] = float32(row[o])
			dst[i+1] = float32(row[o+1])
			dst[i+2] = float32(row[o+2])
			dst[i+3] = float32(row[o+3])
		}
	}
	return nil
}

func (s *screen) windowSize() (int, int) {
	w, h := 2, 2
	if s != nil {
		if s.size.X > w {
			w = s.size.X
		}
		if s.size.Y > h {
			h = s.size.Y
		}
	}
	return w, h
}

// fit shrinks w×h so it sits inside maxW×maxH, keeping the aspect.
func fit(w, h, maxW, maxH int) (int, int) {
	if w < 2 {
		w = 2
	}
	if h < 2 {
		h = 2
	}
	if w <= maxW && h <= maxH {
		return w, h
	}
	sx := float64(maxW) / float64(w)
	sy := float64(maxH) / float64(h)
	scale := sx
	if sy < scale {
		scale = sy
	}
	rw := int(float64(w) * scale)
	rh := int(float64(h) * scale)
	if rw < 2 {
		rw = 2
	}
	if rh < 2 {
		rh = 2
	}
	return rw, rh
}
