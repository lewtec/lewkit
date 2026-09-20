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
}

// Run is tea.Program for a pixel window. The caller opens w.
// Nil evaluator uses [ndarray.CPU].
func Run(ctx context.Context, w window.Window, evaluator ndarray.Evaluator, model Model) error {
	if model == nil {
		return ErrModel
	}
	if w == nil {
		return window.ErrClosed
	}
	if evaluator == nil {
		evaluator = ndarray.CPU
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	r := &runner{ctx: ctx, window: w, evaluator: evaluator, model: model}
	var err error
	r.model, err = finish(r.model, r.model.Init())
	if err != nil {
		return err
	}
	if err := r.step(TickMsg{Size: w.Size()}); err != nil {
		return closed(err)
	}
	events := w.Subscribe(ctx)
	started := time.Now()
	ticks := event.CreateTimer(ctx, time.Second/60)
	for {
		select {
		case <-ctx.Done():
			return closed(context.Cause(ctx))
		case ev, ok := <-events:
			if !ok {
				return nil
			}
			if _, stop := ev.(window.Close); stop {
				next, err := applyMsg(r.model, ev)
				if err == nil {
					r.model = next
				}
				return nil
			}
			if err := r.step(ev); err != nil {
				return closed(err)
			}
		case <-ticks:
			msg := TickMsg{Elapsed: time.Since(started), Size: w.Size()}
			if err := r.step(msg); err != nil {
				return closed(err)
			}
		}
	}
}

func (r *runner) step(msg Msg) error {
	next, err := applyMsg(r.model, msg)
	if err != nil {
		return err
	}
	r.model = next
	view := next.View()
	if view == nil {
		return ErrView
	}
	dst := r.window.Frame()
	if dst == nil {
		return window.ErrClosed
	}
	if err := window.Present(r.ctx, view, r.evaluator, dst); err != nil {
		return err
	}
	return r.window.Draw()
}

func applyMsg(model Model, msg Msg) (Model, error) {
	if model == nil {
		return nil, ErrModel
	}
	next, cmd := model.Update(msg)
	return finish(next, cmd)
}

func finish(model Model, cmd Cmd) (Model, error) {
	if model == nil {
		return nil, ErrModel
	}
	if cmd == nil {
		return model, nil
	}
	msg := cmd()
	if msg == nil {
		return model, nil
	}
	next, follow := model.Update(msg)
	if next == nil {
		return nil, ErrModel
	}
	if follow != nil {
		return next, nil
	}
	return next, nil
}

func closed(err error) error {
	if errors.Is(err, window.ErrClosed) || errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}
