package daynight

import "context"

// Changes sends current once, then each value that differs from the last.
// It closes out when ctx is done.
func Changes(ctx context.Context, current Mode, next <-chan Mode) <-chan Mode {
	out := make(chan Mode, 1)
	out <- current
	go func() {
		defer close(out)
		last := current
		for {
			select {
			case <-ctx.Done():
				return
			case scheme, ok := <-next:
				if !ok {
					return
				}
				if scheme == last {
					continue
				}
				last = scheme
				select {
				case out <- scheme:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out
}
