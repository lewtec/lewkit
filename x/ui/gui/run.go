package gui

import (
	"context"
	"errors"
	"image"
	"time"

	"github.com/lewtec/lewkit/x/driver/daynight"
	"github.com/lewtec/lewkit/x/driver/ndeval"
	"github.com/lewtec/lewkit/x/driver/vulkan"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/event"
	"github.com/lewtec/lewkit/x/ndarray"
)

// display is what the loop asks the host for. Image windows and the swapchain
// screen both satisfy it. The screen is not a window.Window.
type display interface {
	Size() image.Point
	FramePeriod() time.Duration
	Subscribe(ctx context.Context) <-chan window.Event
	present(ctx context.Context, view *ndarray.Tensor[uint8], evaluator ndarray.Evaluator) error
}

// listPainter draws the recorded fills instead of evaluating the fused kernel.
type listPainter interface {
	presentList(ctx context.Context, picture *Picture) error
}

type imageDisplay struct{ window.Window }

func (d imageDisplay) present(ctx context.Context, view *ndarray.Tensor[uint8], evaluator ndarray.Evaluator) error {
	return window.Show(ctx, d.Window, view, evaluator)
}

type bridgeDisplay struct {
	window.Window
	screen    vulkan.Screen
	evaluator ndarray.Evaluator
}

func (d bridgeDisplay) present(ctx context.Context, view *ndarray.Tensor[uint8], _ ndarray.Evaluator) error {
	return ndeval.Paint(ctx, d.evaluator, view, d.screen)
}

func (d bridgeDisplay) presentList(ctx context.Context, picture *Picture) error {
	if picture == nil {
		return ErrView
	}
	size := d.Size()
	var ink []byte
	if picture.hadInk && picture.inkRGBA != nil {
		ink = picture.inkRGBA.Pix
	}
	under, err := picture.rasterBytes(ctx, d.evaluator, size.X, size.Y)
	if err != nil {
		return err
	}
	return drawFills(ctx, d.screen, picture.fills, under, ink, size.X, size.Y)
}

type runner struct {
	ctx           context.Context
	host          display
	evaluator     ndarray.Evaluator
	model         Model
	fps           event.FPS
	hertz         float64
	started       time.Time
	commands      chan Msg
	picture       *Picture
	view          *ndarray.Tensor[uint8]
	signature     uint64
	lastSignature uint64
	dirty         bool
}

// Run is the Elm loop. Update runs on each message. View returns a
// [Node]; Run paints it through [Picture] on the display ticker.
func Run(ctx context.Context, host window.Window, evaluator ndarray.Evaluator, model Model) error {
	if host == nil {
		return window.ErrClosed
	}
	if screen, gpu, err := bridge(ctx, host); err == nil {
		defer screen.Close()
		defer gpu.Close()
		return run(ctx, bridgeDisplay{Window: host, screen: screen, evaluator: gpu}, nil, model)
	}
	return run(ctx, imageDisplay{host}, evaluator, model)
}

func run(ctx context.Context, host display, evaluator ndarray.Evaluator, model Model) error {
	if model == nil {
		return ErrModel
	}
	if host == nil {
		return window.ErrClosed
	}
	if evaluator == nil {
		evaluator = ndarray.CPU
	}
	picture, err := NewPicture()
	if err != nil {
		return err
	}
	runner := &runner{ctx: ctx, host: host, evaluator: evaluator, model: model, started: time.Now(), picture: picture}
	return runner.loop()
}

func (runner *runner) loop() error {
	ctx, cancel := context.WithCancel(runner.ctx)
	defer cancel()
	runner.ctx = ctx
	runner.commands = make(chan Msg, 16)
	events := runner.host.Subscribe(ctx)
	runner.dirty = true
	applied := daynight.Dark
	if scheme, err := daynight.Current(ctx); err == nil {
		applied = scheme
		if err := runner.handle(ModeMsg{Mode: scheme}); err != nil {
			return err
		}
	}
	if changes, err := daynight.Watch(ctx); err == nil {
		go func() {
			first := true
			for scheme := range changes {
				if first {
					first = false
					if scheme == applied {
						continue
					}
				}
				select {
				case runner.commands <- ModeMsg{Mode: scheme}:
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	if err := runner.flush(true); err != nil {
		return err
	}
	runner.spawn(runner.model.Init())
	period := runner.host.FramePeriod()
	if period <= 0 {
		period = window.DefaultFramePeriod
	}
	ticker := time.NewTicker(period)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return closed(context.Cause(ctx))
		case event, ok := <-events:
			if !ok {
				return nil
			}
			if err := runner.handle(event); err != nil {
				return err
			}
			if _, stop := event.(window.Close); stop {
				return nil
			}
			select {
			case <-ticker.C:
				if err := runner.flush(false); err != nil {
					return err
				}
			default:
			}
		case msg := <-runner.commands:
			if err := runner.handle(msg); err != nil {
				return err
			}
			if _, stop := msg.(window.Close); stop {
				return nil
			}
			if _, tick := msg.(TickMsg); tick {
				stop, err := runner.drainLatest()
				if err != nil {
					return err
				}
				if stop {
					return nil
				}
				if err := runner.flush(false); err != nil {
					return err
				}
				ticker.Reset(period)
				continue
			}
			select {
			case <-ticker.C:
				if err := runner.flush(false); err != nil {
					return err
				}
			default:
			}
		case <-ticker.C:
			if err := runner.flush(false); err != nil {
				return err
			}
		}
	}
}

// drainLatest applies ticks already queued. A newer frame replaces one
// that has not been presented yet.
func (runner *runner) drainLatest() (bool, error) {
	for {
		select {
		case newer := <-runner.commands:
			if err := runner.handle(newer); err != nil {
				return false, err
			}
			if _, stop := newer.(window.Close); stop {
				return true, nil
			}
		default:
			return false, nil
		}
	}
}

func (runner *runner) decorate(msg Msg) Msg {
	tick, ok := msg.(TickMsg)
	if !ok {
		return msg
	}
	tick.Elapsed = time.Since(runner.started)
	tick.Size = runner.host.Size()
	tick.FPS = runner.hertz
	tick.Period = runner.host.FramePeriod()
	return tick
}

func (runner *runner) handle(msg Msg) error {
	msg = runner.decorate(msg)
	if runner.model == nil {
		return ErrModel
	}
	next, cmd := runner.model.Update(msg)
	if next == nil {
		return ErrModel
	}
	runner.model = next
	runner.spawn(cmd)
	if changed, ok := next.(interface{ Consume() bool }); ok {
		if changed.Consume() {
			runner.dirty = true
		}
	} else {
		runner.dirty = true
	}
	if _, stop := msg.(window.Close); stop {
		return nil
	}
	switch msg.(type) {
	case window.Expose, window.Resize:
		return runner.flush(true)
	}
	return nil
}

func (runner *runner) spawn(cmd Cmd) {
	if cmd == nil {
		return
	}
	go func() {
		msg := cmd()
		if msg == nil {
			return
		}
		select {
		case runner.commands <- msg:
		case <-runner.ctx.Done():
		}
	}()
}

func (runner *runner) render() error {
	root := runner.model.View()
	if root == nil {
		return ErrView
	}
	size := runner.host.Size()
	pixels, err := runner.picture.Render(root, Size{float32(size.X), float32(size.Y)})
	if err != nil {
		return err
	}
	if pixels == nil {
		if runner.picture.recordOnly {
			runner.view = nil
			runner.signature = runner.picture.frameSig()
			return nil
		}
		return ErrView
	}
	runner.view = pixels
	runner.signature = runner.picture.frameSig()
	return nil
}

func (runner *runner) flush(force bool) error {
	if !force && !runner.dirty {
		return nil
	}
	_, listed := runner.host.(listPainter)
	runner.picture.recordOnly = listed
	if err := runner.render(); err != nil {
		runner.picture.recordOnly = false
		return err
	}
	runner.picture.recordOnly = false
	runner.dirty = false
	if !force && runner.picture.raster == nil && runner.signature != 0 && runner.signature == runner.lastSignature {
		return nil
	}
	if listed && runner.view != nil && runner.picture.raster != nil {
		if err := runner.host.present(runner.ctx, runner.view, runner.evaluator); err != nil {
			return err
		}
		runner.hertz = runner.fps.Get()
		runner.lastSignature = runner.signature
		return nil
	}
	if listed {
		if err := runner.host.(listPainter).presentList(runner.ctx, runner.picture); err != nil {
			return err
		}
		runner.hertz = runner.fps.Get()
		runner.lastSignature = runner.signature
		return nil
	}
	if runner.view == nil {
		return ErrView
	}
	if err := runner.host.present(runner.ctx, runner.view, runner.evaluator); err != nil {
		return err
	}
	runner.hertz = runner.fps.Get()
	runner.lastSignature = runner.signature
	return nil
}

func closed(err error) error {
	if errors.Is(err, window.ErrClosed) || errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}
