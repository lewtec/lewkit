package window

import (
	"context"
	"errors"
	"image"
	"time"
)

// Paint fills the back buffer. Animate calls Draw after each Paint.
type Paint func(dst *image.RGBA, elapsed time.Duration) error

// Drive is the immediate-mode paint loop used by [Animate]. fn gets a
// nil Event on the first frame and on each tick; other calls pass the
// host Event. Close is delivered once, then Drive returns. After a host
// event, a pending tick is taken so a flood of Resize cannot starve paint.
func Drive(ctx context.Context, w Window, period time.Duration, fn func(ev Event, elapsed time.Duration) error) error {
	if w == nil || fn == nil {
		return ErrClosed
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	events := w.Subscribe(ctx)
	started := time.Now()
	if period <= 0 {
		period = w.FramePeriod()
	}
	if period <= 0 {
		period = DefaultFramePeriod
	}
	ticker := time.NewTicker(period)
	defer ticker.Stop()
	if err := fn(nil, 0); err != nil {
		return closed(err)
	}
	for {
		select {
		case <-ctx.Done():
			return closed(context.Cause(ctx))
		case ev, ok := <-events:
			if !ok {
				return nil
			}
			elapsed := time.Since(started)
			if _, stop := ev.(Close); stop {
				err := fn(ev, elapsed)
				if err != nil {
					return closed(err)
				}
				return nil
			}
			if err := fn(ev, elapsed); err != nil {
				return closed(err)
			}
			select {
			case <-ticker.C:
				if err := fn(nil, time.Since(started)); err != nil {
					return closed(err)
				}
			default:
			}
		case <-ticker.C:
			if err := fn(nil, time.Since(started)); err != nil {
				return closed(err)
			}
		}
	}
}

// Animate runs paint at period until ctx is done or the window closes.
// Resize paints immediately so the new Frame is filled. Other events
// are ignored.
func Animate(ctx context.Context, w Window, period time.Duration, paint Paint) error {
	return Drive(ctx, w, period, func(ev Event, elapsed time.Duration) error {
		if ev != nil {
			if _, ok := ev.(Resize); !ok {
				return nil
			}
		}
		return paintFrame(w, paint, elapsed)
	})
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

func closed(err error) error {
	if errors.Is(err, ErrClosed) || errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}
