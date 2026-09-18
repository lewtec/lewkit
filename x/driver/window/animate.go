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
// Resize paints immediately so the new Frame is filled.
func Animate(ctx context.Context, w Window, period time.Duration, paint Paint) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	events := w.Subscribe(ctx)
	started := time.Now()
	if period <= 0 {
		period = time.Second / 60
	}
	tick := time.NewTicker(period)
	defer tick.Stop()
	if err := paintFrame(w, paint, 0); err != nil {
		return closed(err)
	}
	for {
		resized, err := drain(events)
		if err != nil {
			return closed(err)
		}
		if resized {
			if err := paintFrame(w, paint, time.Since(started)); err != nil {
				return closed(err)
			}
		}
		select {
		case <-ctx.Done():
			return closed(context.Cause(ctx))
		case ev, ok := <-events:
			if !ok {
				return nil
			}
			if err := closedEvent(ev); err != nil {
				return closed(err)
			}
			if _, ok := ev.(Resize); ok {
				if err := paintFrame(w, paint, time.Since(started)); err != nil {
					return closed(err)
				}
			}
		case <-tick.C:
			if err := paintFrame(w, paint, time.Since(started)); err != nil {
				return closed(err)
			}
		}
	}
}

func paintFrame(w Window, paint Paint, elapsed time.Duration) error {
	dst := w.Frame()
	if dst == nil {
		return ErrClosed
	}
	if err := paint(dst, elapsed); err != nil {
		return err
	}
	return w.Draw()
}

func drain(events <-chan Event) (bool, error) {
	resized := false
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				return resized, ErrClosed
			}
			if err := closedEvent(ev); err != nil {
				return resized, err
			}
			if _, ok := ev.(Resize); ok {
				resized = true
			}
		default:
			return resized, nil
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
