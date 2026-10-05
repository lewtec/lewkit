// Dead Zone bounces one colored square on a black window.
//
// The world schedule owns the square. Startup spawns it. On each tick,
// Step feeds the host delta to Time, drift moves the square and bounces
// it off the safe box, and recolor walks its hue. View reads those
// columns. Hint keeps the square in the box the host leaves clear of
// navbars and notches. The rest of the window stays black.
//
//	go run ./cmd/lewkit release run --config ./examples/deadzone/eletrocromo.json
package main

import (
	"context"
	"time"

	"github.com/lewtec/lewkit/x/app"
	_ "github.com/lewtec/lewkit/x/driver/prelude"
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/lewtec/lewkit/x/ui/gui"
	"github.com/lewtec/lewkit/x/ui/world"
)

func init() { entry.Bind(runApp) }

func main() { entry.Main(runApp) }

func runApp(ctx context.Context) error {
	return app.App{
		Title:   "Dead Zone",
		Width:   480,
		Height:  800,
		Handler: app.GUI(newScreen(ctx)),
	}.Run(ctx)
}

var black = gui.RGB{Alpha: 255}

type screen struct {
	gui.Dirty
	sim  *world.Sim
	seen time.Duration
	ctx  context.Context
}

func newScreen(ctx context.Context) *screen {
	if ctx == nil {
		ctx = context.Background()
	}
	sim := world.New()
	sim.System(world.Startup, spawnSquare)
	sim.Chain(world.Update, drift, recolor)
	s := &screen{sim: sim, ctx: ctx}
	s.advance(0)
	return s
}

func (s *screen) Init() gui.Cmd { return gui.Tick() }

func (s *screen) Update(msg gui.Msg) (gui.Model, gui.Cmd) {
	if s == nil {
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
	s.advance(dt)
	if tick.Period > 0 {
		return s, gui.Every(tick.Period)
	}
	return s, gui.Every(time.Second / 30)
}

func (s *screen) advance(dt float64) {
	if s == nil || s.sim == nil {
		return
	}
	s.sim.Step(s.ctx, dt)
	s.Dirty = gui.Touch(s.Dirty)
}

func (s *screen) square() (spot, tint) {
	if s == nil || s.sim == nil || s.sim.World == nil {
		return spot{}, tint{R: 255}
	}
	entity, at, ok := world.Query[spot](s.sim.World).First(nil)
	if !ok {
		return spot{}, tint{R: 255}
	}
	color, ok := world.Query[tint](s.sim.World).Get(entity)
	if !ok {
		color = tint{R: 255}
	}
	return at, color
}

func (s *screen) View() gui.Node {
	if s == nil {
		return &gui.Box{Fill: &black}
	}
	at, color := s.square()
	return &gui.Stack{Children: []gui.Node{
		&gui.Box{Fill: &black},
		&gui.Hint{Child: &field{x: at.X, y: at.Y, ink: gui.RGB{Red: color.R, Green: color.G, Blue: color.B, Alpha: 255}}},
	}}
}

// field places the square inside the box Hint measured.
type field struct {
	x, y   float32
	ink    gui.RGB
	placed gui.Positioned
	box    gui.Box
	size   gui.Size
}

func (f *field) Layout(constraints gui.BoxConstraints) gui.Size {
	if f == nil {
		return gui.Size{}
	}
	f.size = constraints.Constrain(gui.Size{Width: constraints.MaxWidth, Height: constraints.MaxHeight})
	side := squareShare * min(f.size.Width, f.size.Height)
	if side > f.size.Width {
		side = f.size.Width
	}
	if side > f.size.Height {
		side = f.size.Height
	}
	spanX := max(f.size.Width-side, 0)
	spanY := max(f.size.Height-side, 0)
	f.box.Width = side
	f.box.Height = side
	f.box.Fill = &f.ink
	f.placed.X = f.x * spanX
	f.placed.Y = f.y * spanY
	f.placed.Child = &f.box
	f.placed.Layout(gui.BoxConstraints{MaxWidth: f.size.Width, MaxHeight: f.size.Height})
	return f.size
}

func (f *field) Paint(origin gui.Offset, clip gui.Rect, picture *gui.Picture) *ndarray.Tensor[float32] {
	if f == nil {
		return nil
	}
	return f.placed.Paint(origin, clip, picture)
}
