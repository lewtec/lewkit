// Dead Zone fills the fully shown box with one color on a black window.
//
// The world schedule owns that color. Startup spawns it. On each tick,
// Step feeds the host delta to Time and recolor walks the hue. View
// reads the column. Hint paints the color across the whole box the host
// leaves clear of navbars and notches. The rest of the window stays black.
//
//	go run ./cmd/lewkit release run --config ./examples/deadzone/eletrocromo.json
package main

import (
	"context"
	"time"

	"github.com/lewtec/lewkit/x/app"
	_ "github.com/lewtec/lewkit/x/driver/prelude"
	"github.com/lewtec/lewkit/x/entry"
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
	ink  gui.RGB
}

func newScreen(ctx context.Context) *screen {
	if ctx == nil {
		ctx = context.Background()
	}
	sim := world.New()
	sim.System(world.Startup, spawnFill)
	sim.System(world.Update, recolor)
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

func (s *screen) color() tint {
	fallback := tint{R: 255}
	if s == nil || s.sim == nil || s.sim.World == nil {
		return fallback
	}
	_, color, ok := world.Query[tint](s.sim.World).First(nil)
	if !ok {
		return fallback
	}
	return color
}

func (s *screen) View() gui.Node {
	if s == nil {
		return &gui.Box{Fill: &black}
	}
	color := s.color()
	s.ink = gui.RGB{Red: color.R, Green: color.G, Blue: color.B, Alpha: 255}
	return &gui.Stack{Children: []gui.Node{
		&gui.Box{Fill: &black},
		&gui.Hint{Child: &gui.Box{Fill: &s.ink}},
	}}
}
