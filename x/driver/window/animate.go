package window

import (
	"context"
	"errors"
	"image"
	"time"
)

// Paint fills the back buffer. Animate calls Draw after each Paint.
type Paint func(dst *image.RGBA, elapsed time.Duration) error

// Animate runs paint at period until ctx is done or the window closes.
// Events other than Close are ignored; the next tick sees the new Frame size.
func Animate(ctx context.Context, w Window, period time.Duration, paint Paint) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	evs := w.Subscribe(ctx)
	t0 := time.Now()
	if period <= 0 {
		period = time.Second / 60
	}
	tick := time.NewTicker(period)
	defer tick.Stop()
	if err := frame(w, paint, 0); err != nil {
		return closed(err)
	}
	for {
		if err := drain(evs); err != nil {
			return closed(err)
		}
		select {
		case <-ctx.Done():
			return closed(context.Cause(ctx))
		case ev, ok := <-evs:
			if !ok {
				return nil
			}
			if err := closedEvent(ev); err != nil {
				return closed(err)
			}
		case <-tick.C:
			if err := frame(w, paint, time.Since(t0)); err != nil {
				return closed(err)
			}
		}
	}
}

func frame(w Window, paint Paint, elapsed time.Duration) error {
	dst := w.Frame()
	if dst == nil {
		return ErrClosed
	}
	if err := paint(dst, elapsed); err != nil {
		return err
	}
	return w.Draw()
}

func drain(evs <-chan Event) error {
	for {
		select {
		case ev, ok := <-evs:
			if !ok {
				return ErrClosed
			}
			if err := closedEvent(ev); err != nil {
				return err
			}
		default:
			return nil
		}
	}
}

func closedEvent(ev Event) error {
	if _, ok := ev.(Close); ok {
		return ErrClosed
	}
	return nil
}

func closed(err error) error {
	if errors.Is(err, ErrClosed) || errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}
