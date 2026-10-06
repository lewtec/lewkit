// DVD bounces a logo inside the usable box, the way a disc menu used to.
//
// The world schedule owns the logo. Startup spawns it. On each tick,
// drift moves it and turns it around at the edges of the fully shown box.
// The axes travel at different speeds. A wall hit picks a new speed for
// that axis and changes the color. Both walls at once, a corner, turn it
// white.
// Hint is that box. A light rectangle plots its edges, and the area a
// navbar or notch covers stays outside the rectangle.
//
//	go run ./cmd/lewkit release run --config ./examples/dvd/eletrocromo.json
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
		Title:   "DVD",
		Width:   960,
		Height:  540,
		Handler: app.GUI(newScreen(ctx)),
	}.Run(ctx)
}

const plotWidth = float32(3)

var (
	outside = gui.RGB{Red: 28, Green: 24, Blue: 32, Alpha: 255}
	inside  = gui.RGB{Alpha: 255}
	plot    = gui.RGB{Red: 236, Green: 236, Blue: 240, Alpha: 255}
	wordInk = gui.RGB{Red: 255, Green: 255, Blue: 255, Alpha: 255}
)

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
	sim.System(world.Startup, spawnLogo)
	sim.System(world.Update, drift)
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

func (s *screen) logo() (spot, ink) {
	fallback := dvdColors[0]
	if s == nil || s.sim == nil || s.sim.World == nil {
		return spot{}, fallback
	}
	entity, at, ok := world.Query[spot](s.sim.World).First(nil)
	if !ok {
		return spot{}, fallback
	}
	color, ok := world.Query[ink](s.sim.World).Get(entity)
	if !ok {
		color = fallback
	}
	return at, color
}

func (s *screen) View() gui.Node {
	if s == nil {
		return &gui.Box{Fill: &outside}
	}
	at, color := s.logo()
	return &gui.Stack{Children: []gui.Node{
		&gui.Box{Fill: &outside},
		&gui.Hint{Child: &stage{
			x: at.X,
			y: at.Y,
			ink: gui.RGB{
				Red: color.R, Green: color.G, Blue: color.B, Alpha: 255,
			},
		}},
	}}
}

// stage is the usable box. The light bars are its rectangle. The logo
// travels inside that rectangle.
type stage struct {
	x, y   float32
	ink    gui.RGB
	size   gui.Size
	root   gui.Stack
	logo   gui.Box
	placed gui.Positioned
	word   gui.Text
}

func (s *stage) Layout(constraints gui.BoxConstraints) gui.Size {
	if s == nil {
		return gui.Size{}
	}
	s.size = constraints.Constrain(gui.Size{Width: constraints.MaxWidth, Height: constraints.MaxHeight})
	thickness := plotWidth
	if limit := min(s.size.Width, s.size.Height) / 2; thickness > limit {
		thickness = limit
	}
	logoW := min(s.size.Width*logoShare, s.size.Width)
	logoH := logoW * 0.4
	if logoH > s.size.Height {
		logoH = s.size.Height
	}
	spanX := max(s.size.Width-logoW, 0)
	spanY := max(s.size.Height-logoH, 0)
	s.word = gui.Text{Value: "DVD", Cursor: -1, Ink: wordInk}
	s.logo = gui.Box{
		Width:  logoW,
		Height: logoH,
		Fill:   &s.ink,
		Align:  gui.Alignment{X: 0.5, Y: 0.5},
		Clip:   true,
		Child:  &s.word,
	}
	s.placed = gui.Positioned{X: s.x * spanX, Y: s.y * spanY, Child: &s.logo}
	s.root = gui.Stack{Children: []gui.Node{
		&gui.Box{Width: s.size.Width, Height: s.size.Height, Fill: &inside},
		bar(0, 0, s.size.Width, thickness),
		bar(0, s.size.Height-thickness, s.size.Width, thickness),
		bar(0, 0, thickness, s.size.Height),
		bar(s.size.Width-thickness, 0, thickness, s.size.Height),
		&s.placed,
	}}
	s.root.Layout(gui.BoxConstraints{
		MinWidth: s.size.Width, MinHeight: s.size.Height,
		MaxWidth: s.size.Width, MaxHeight: s.size.Height,
	})
	return s.size
}

func bar(x, y, width, height float32) *gui.Positioned {
	return &gui.Positioned{X: x, Y: y, Child: &gui.Box{Width: width, Height: height, Fill: &plot}}
}

func (s *stage) Paint(origin gui.Offset, clip gui.Rect, picture *gui.Picture) *ndarray.Tensor[float32] {
	if s == nil {
		return nil
	}
	return s.root.Paint(origin, clip, picture)
}
