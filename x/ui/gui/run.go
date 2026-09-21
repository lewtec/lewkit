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
	ctx       context.Context
	window    window.Window
	evaluator ndarray.Evaluator
	model     Model
	fps       event.FPS
	hz        float64
	started   time.Time
	cmds      chan Msg
	picture   *Picture
	view      *ndarray.Tensor[uint8]
	sig       uint64
	last      uint64
	dirty     bool
}

const defaultSlots = 64

// Run is the Elm loop. Update runs on each message. View returns a
// [Node]; Run paints it through [Picture] on the display ticker.
func Run(ctx context.Context, host window.Window, evaluator ndarray.Evaluator, model Model) error {
	return run(ctx, host, evaluator, model, 0)
}

func run(ctx context.Context, host window.Window, evaluator ndarray.Evaluator, model Model, slots int) error {
	if model == nil {
		return ErrModel
	}
	if host == nil {
		return window.ErrClosed
	}
	if evaluator == nil {
		evaluator = ndarray.CPU
	}
	if slots <= 0 {
		slots = defaultSlots
	}
	picture, err := NewPicture(slots)
	if err != nil {
		return err
	}
	r := &runner{ctx: ctx, window: host, evaluator: evaluator, model: model, started: time.Now(), picture: picture}
	return r.loop()
}

func (r *runner) loop() error {
	ctx, cancel := context.WithCancel(r.ctx)
	defer cancel()
	r.ctx = ctx
	r.cmds = make(chan Msg, 16)
	events := r.window.Subscribe(ctx)
	r.dirty = true
	if err := r.flush(true); err != nil {
		return err
	}
	r.spawn(r.model.Init())
	period := r.window.FramePeriod()
	if period <= 0 {
		period = window.DefaultFramePeriod
	}
	ticker := time.NewTicker(period)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return closed(context.Cause(ctx))
		case ev, ok := <-events:
			if !ok {
				return nil
			}
			if err := r.handle(ev); err != nil {
				return err
			}
			if _, stop := ev.(window.Close); stop {
				return nil
			}
			select {
			case <-ticker.C:
				if err := r.flush(false); err != nil {
					return err
				}
			default:
			}
		case msg := <-r.cmds:
			if err := r.handle(msg); err != nil {
				return err
			}
			select {
			case <-ticker.C:
				if err := r.flush(false); err != nil {
					return err
				}
			default:
			}
		case <-ticker.C:
			if err := r.flush(false); err != nil {
				return err
			}
		}
	}
}

func (r *runner) decorate(msg Msg) Msg {
	t, ok := msg.(TickMsg)
	if !ok {
		return msg
	}
	t.Elapsed = time.Since(r.started)
	t.Size = r.window.Size()
	t.FPS = r.hz
	t.Period = r.window.FramePeriod()
	return t
}

func (r *runner) handle(msg Msg) error {
	msg = r.decorate(msg)
	if r.model == nil {
		return ErrModel
	}
	next, cmd := r.model.Update(msg)
	if next == nil {
		return ErrModel
	}
	r.model = next
	r.spawn(cmd)
	r.dirty = true
	if _, stop := msg.(window.Close); stop {
		return nil
	}
	switch msg.(type) {
	case window.Expose, window.Resize:
		return r.flush(true)
	}
	return nil
}

func (r *runner) spawn(cmd Cmd) {
	if cmd == nil {
		return
	}
	go func() {
		msg := cmd()
		if msg == nil {
			return
		}
		select {
		case r.cmds <- msg:
		case <-r.ctx.Done():
		}
	}()
}

func (r *runner) render() error {
	root := r.model.View()
	if root == nil {
		return ErrView
	}
	size := r.window.Size()
	t, err := r.picture.Render(root, Size{float32(size.X), float32(size.Y)})
	if err != nil {
		return err
	}
	if t == nil {
		return ErrView
	}
	r.view = t
	r.sig = r.picture.frameSig()
	return nil
}

func (r *runner) flush(force bool) error {
	if !force && !r.dirty {
		return nil
	}
	if err := r.render(); err != nil {
		return err
	}
	r.dirty = false
	if !force && r.sig != 0 && r.sig == r.last {
		return nil
	}
	if r.view == nil {
		return ErrView
	}
	destination := r.window.Frame()
	if destination == nil {
		return window.ErrClosed
	}
	err := window.Present(r.ctx, r.view, r.evaluator, destination)
	if drawErr := r.window.Draw(); err != nil {
		return err
	} else if drawErr != nil {
		return drawErr
	}
	r.hz = r.fps.Get()
	r.last = r.sig
	return nil
}

func closed(err error) error {
	if errors.Is(err, window.ErrClosed) || errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}
