package gui

import (
	"context"
	"errors"
	"time"

	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/event"
	"github.com/lewtec/lewkit/x/ndarray"
)

type runner struct {
	ctx           context.Context
	window        window.Window
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
	return run(ctx, host, evaluator, model)
}

func run(ctx context.Context, host window.Window, evaluator ndarray.Evaluator, model Model) error {
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
	runner := &runner{ctx: ctx, window: host, evaluator: evaluator, model: model, started: time.Now(), picture: picture}
	return runner.loop()
}

func (runner *runner) loop() error {
	ctx, cancel := context.WithCancel(runner.ctx)
	defer cancel()
	runner.ctx = ctx
	runner.commands = make(chan Msg, 16)
	events := runner.window.Subscribe(ctx)
	runner.dirty = true
	if err := runner.flush(true); err != nil {
		return err
	}
	runner.spawn(runner.model.Init())
	period := runner.window.FramePeriod()
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

func (runner *runner) decorate(msg Msg) Msg {
	tick, ok := msg.(TickMsg)
	if !ok {
		return msg
	}
	tick.Elapsed = time.Since(runner.started)
	tick.Size = runner.window.Size()
	tick.FPS = runner.hertz
	tick.Period = runner.window.FramePeriod()
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
	runner.dirty = true
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
	size := runner.window.Size()
	pixels, err := runner.picture.Render(root, Size{float32(size.X), float32(size.Y)})
	if err != nil {
		return err
	}
	if pixels == nil {
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
	if err := runner.render(); err != nil {
		return err
	}
	runner.dirty = false
	if !force && runner.signature != 0 && runner.signature == runner.lastSignature {
		return nil
	}
	if runner.view == nil {
		return ErrView
	}
	if err := window.Show(runner.ctx, runner.window, runner.view, runner.evaluator); err != nil {
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
