package present

import (
	"context"
	"errors"

	"github.com/lewtec/lewkit/x/driver"
)

// Open attaches the highest-weight screen that accepts this surface.
// A driver that rejects the surface is skipped. The next one is tried.
func Open(ctx context.Context, kind int, a, b uintptr, width, height int) (Screen, error) {
	handles, err := driver.List[Driver](ctx)
	if err != nil {
		return nil, err
	}
	var last error
	for _, handle := range handles {
		opener, err := handle.Open(ctx)
		if err != nil {
			last = err
			continue
		}
		screen, err := opener.Open(ctx, kind, a, b, width, height)
		if err != nil {
			last = err
			continue
		}
		return screen, nil
	}
	if last == nil {
		last = driver.ErrUnavailable
	}
	return nil, last
}

// Lost reports whether err is [ErrLost].
func Lost(err error) bool { return errors.Is(err, ErrLost) }
