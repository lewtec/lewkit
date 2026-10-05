// Chess plays the Caballero Coll tutorial on the world frame.
//
// https://caballerocoll.com/blog/bevy-chess-tutorial/
// The tutorial text is MIT licensed. BoardPlugin chains select, move,
// select piece, reset, despawn, and color. PiecesPlugin spawns the
// pieces at startup. The CPU plugin places its system after reset and
// before despawn, so that order holds whichever plugin was added first.
// A click is a resource the host replaces. The accepted click and the
// reset are messages a later system reads. The view
// is the article's scene: a camera above the side of the board, one
// plane per square, and pieces that slide toward their square. The
// picture is a raster. This package does not load a mesh kit.
//
//	go run ./cmd/lewkit release run --config ./examples/chess/eletrocromo.json
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
	"github.com/lewtec/lewkit/x/ui/gui"
	"github.com/lewtec/lewkit/x/ui/world"
)

func init() { entry.Bind(runApp) }

func main() { entry.Main(runApp) }

func runApp(ctx context.Context) error {
	return app.App{
		Title:   "Chess",
		Width:   640,
		Height:  720,
		Handler: app.GUI(newScreen(ctx)),
	}.Run(ctx)
}

type screen struct {
	gui.Dirty
	sim        *world.Sim
	frame      *image.RGBA
	picture    *ndarray.Tensor[float32]
	size       image.Point
	seen       time.Duration
	ctx        context.Context
	depth      []float32
	cover      []uint8
	boxes      [64]pixelBox
	tones      [64]shade
	drawn      uint64
	caption    string
	fullPaints int
}

func newScreen(ctx context.Context) *screen {
	if ctx == nil {
		ctx = context.Background()
	}
	sim := world.New()
	sim.Plugin(world.Group(boardPlugin, piecesPlugin, cpuPlugin))
	s := &screen{sim: sim, size: image.Pt(640, 720), ctx: ctx}
	s.advance(0)
	return s
}

func (s *screen) Init() gui.Cmd { return gui.Tick() }

func (s *screen) Update(msg gui.Msg) (gui.Model, gui.Cmd) {
	if s == nil {
		return s, nil
	}
	switch message := msg.(type) {
	case window.Resize:
		s.adopt(message.Size)
		s.paint()
		s.Dirty = gui.Touch(s.Dirty)
		return s, nil
	case window.Pointer:
		s.track(message)
		s.advance(0)
		return s, nil
	case gui.TickMsg:
		s.adopt(message.Size)
		dt := 0.0
		if message.Elapsed > s.seen {
			dt = (message.Elapsed - s.seen).Seconds()
		}
		s.seen = message.Elapsed
		s.advance(dt)
		if message.Period > 0 {
			return s, gui.Every(message.Period)
		}
		return s, gui.Every(time.Second / 30)
	default:
		return s, nil
	}
}

func (s *screen) adopt(size image.Point) {
	if s == nil || size.X < 16 || size.Y < 16 {
		return
	}
	s.size = size
}

// advance runs one frame. Hover and selection recolor those squares.
// A moved piece draws the mesh again.
func (s *screen) advance(dt float64) {
	if s == nil || s.sim == nil {
		return
	}
	s.sim.Step(s.ctx, dt)
	mismatched := s.frame == nil || s.frame.Bounds().Dx() != s.size.X || s.frame.Bounds().Dy() != s.size.Y
	if s.picture == nil || mismatched || s.geometry() != s.drawn {
		s.paint()
		s.Dirty = gui.Touch(s.Dirty)
		return
	}
	dirty := false
	if text := s.captionText(); text != s.caption {
		s.caption = text
		dirty = true
	}
	next := readShades(s.sim.World)
	var changed [64]int
	n := 0
	for i := range next {
		if next[i] != s.tones[i] {
			changed[n] = i
			n++
		}
	}
	if n > 0 {
		s.tones = next
		s.recolor(changed[:n])
		dirty = true
	}
	if dirty {
		s.Dirty = gui.Touch(s.Dirty)
	}
}

func (s *screen) captionText() string {
	if s == nil || s.sim == nil || s.sim.World == nil {
		return ""
	}
	line, ok := world.Read[banner](s.sim.World)
	if !ok {
		return ""
	}
	return line.text
}

// geometry is a fingerprint of the pieces and their sliding poses.
func (s *screen) geometry() uint64 {
	if s == nil || s.sim == nil || s.sim.World == nil {
		return 0
	}
	var h uint64 = 14695981039346656037
	poses := world.Query[pose](s.sim.World)
	world.Query[piece](s.sim.World).Read(func(e world.Entity, p piece) {
		h ^= uint64(p.color) + 1
		h *= 1099511628211
		h ^= uint64(p.kind) + 1
		h *= 1099511628211
		h ^= uint64(p.x) + 1
		h *= 1099511628211
		h ^= uint64(p.y) + 1
		h *= 1099511628211
		if at, ok := poses.Get(e); ok {
			h ^= uint64(math.Float32bits(at.x))
			h *= 1099511628211
			h ^= uint64(math.Float32bits(at.z))
			h *= 1099511628211
		}
	})
	return h
}

func (s *screen) View() gui.Node {
	if s == nil {
		return &gui.Stack{}
	}
	return s.view()
}
